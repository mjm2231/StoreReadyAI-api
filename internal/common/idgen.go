package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ------------------------------
// Snowflake (41 位时间戳 | 10 位节点 | 12 位序列)
// ------------------------------
// 参考资料：
// - 41 位毫秒时间戳，10 位节点 ID，12 位序列号（共 64 位）。
//   参见 Twitter/X Snowflake 格式。支持约 69 年，1024 个节点，单节点每毫秒 4096 个 ID。
//   https://en.wikipedia.org/wiki/Snowflake_ID
// - Sonyflake 另一种位布局（39/16/8）— 此处采用经典 Snowflake。
//   https://github.com/sony/sonyflake
// - 时钟回退处理策略（等待或使用回退位）。此处选择等待策略。
//   https://leapcell.io/blog/distributed-id-generation-snowflake

const (
	bitsTime = 41
	bitsNode = 10
	bitsSeq  = 12

	maxNode = (1 << bitsNode) - 1
	maxSeq  = (1 << bitsSeq) - 1
)

// 选择一个纪元时间：2025-01-01T00:00:00Z
var defaultEpoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

type snowflake struct {
	mu       sync.Mutex
	epochMs  int64
	nodeID   uint16 // 0..1023
	lastMs   int64
	sequence uint16 // 0..4095
}

var (
	sf   *snowflake
	once sync.Once
)

// InitIDs 初始化全局 Snowflake 生成器。
// 当 nodeID < 0 或 > 1023 时返回错误；nodeID == -1 时自动从主机名派生。
func InitIDs(nodeID int, epoch time.Time) error {
	e := epoch
	if e.IsZero() {
		e = defaultEpoch
	}
	id := uint16(0)
	if nodeID < 0 {
		// 从环境变量或主机名自动派生
		id = deriveNodeID()
	} else {
		if nodeID > maxNode {
			return fmt.Errorf("nodeID 超出范围 [0,%d]: %d", maxNode, nodeID)
		}
		id = uint16(nodeID)
	}
	once.Do(func() { sf = &snowflake{epochMs: e.UnixMilli(), nodeID: id} })
	if sf == nil {
		return errors.New("InitIDs 失败 (sf 为空)")
	}
	return nil
}

// MustInitIDs 初始化失败时触发 panic（用于进程启动阶段）。
func MustInitIDs(nodeID int, epoch time.Time) {
	if err := InitIDs(nodeID, epoch); err != nil {
		panic(err)
	}
}

// NextID 返回新的 64 位 Snowflake ID。
func NextID() uint64 {
	if sf == nil {
		// 默认延迟初始化：节点自动，默认纪元
		_ = InitIDs(-1, time.Time{})
	}
	return sf.next()
}

func (s *snowflake) next() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	nowMs := time.Now().UnixMilli()
	if nowMs < s.lastMs {
		// 时钟回拨 — 阻塞直到追上（简单且安全的策略）
		delay := s.lastMs - nowMs
		time.Sleep(time.Duration(delay) * time.Millisecond)
		nowMs = time.Now().UnixMilli()
	}

	ts := nowMs - s.epochMs
	if ts < 0 {
		// 系统时间早于纪元
		ts = 0
	}

	if ts == s.lastMs-s.epochMs {
		// 同一毫秒内
		s.sequence = (s.sequence + 1) & maxSeq
		if s.sequence == 0 {
			// 毫秒内序列号溢出，等待下一毫秒
			ts = s.waitNext(s.lastMs) - s.epochMs
		}
	} else {
		// 新的毫秒
		s.sequence = 0
		// 保持 ts 不变
	}

	s.lastMs = s.epochMs + ts

	id := (uint64(ts) << (bitsNode + bitsSeq)) |
		(uint64(s.nodeID) << bitsSeq) |
		uint64(s.sequence)
	return id
}

func (s *snowflake) waitNext(last int64) int64 {
	for {
		n := time.Now().UnixMilli()
		if n > last {
			return n
		}
		// 轻微休眠减少 CPU 占用的忙等待
		time.Sleep(50 * time.Microsecond)
	}
}

// deriveNodeID 从环境变量或主机名派生 10 位节点 ID。
// 优先级：环境变量 SNOWFLAKE_NODE_ID -> hash(hostname + pid) & 1023
func deriveNodeID() uint16 {
	if v := strings.TrimSpace(os.Getenv("SNOWFLAKE_NODE_ID")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= maxNode {
			return uint16(n)
		}
	}
	h := fnv.New32a()
	host, _ := os.Hostname()
	_, _ = h.Write([]byte(host))
	_, _ = h.Write([]byte("#"))
	_, _ = h.Write([]byte(strconv.Itoa(os.Getpid())))
	return uint16(h.Sum32() & maxNode)
}

// ------------------------------
// 领域特定辅助函数
// ------------------------------

// NextTenantID 返回租户实体的 Snowflake ID。
func NextTenantID() uint64 { return NextID() }

// NextUserID 返回 Snowflake ID 及短人类友好 UID（8 字符，Crockford Base32）用于展示。
func NextUserID() (id uint64, uid string) {
	id = NextID()
	uid = MakePrettyCode(id, 'U', 8)
	return
}

// NextPetID 返回 Snowflake ID 及短人类友好 PID（8 字符，Crockford Base32）用于展示。
func NextPetID() (id uint64, pid string) {
	id = NextID()
	pid = MakePrettyCode(id, 'P', 8)
	return
}

// ------------------------------
// 美化码（Crockford Base32，不含 I,L,O,U）
// ------------------------------
// 设计理念：完整 Snowflake 转 Base32 需最多 13 字符。为美观易记，
// 从 Snowflake ID 的稳定哈希（40 位）派生 8 字符码，
// 采用 Crockford Base32 编码并以连字符分组。数据库存唯一索引。
// 参考：Crockford Base32 字母表及可选校验（默认不使用额外符号）。
// - https://www.baeldung.com/cs/crockfords-base32-encoding
// - https://en.wikipedia.org/wiki/Base32#Crockford's_Base32

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ" // 不含 I, L, O, U

// MakePrettyCode 将 64 位 id 转为带前缀的短码，如 "U-AB3D-9K2M"。
// width 为 Base32 字符数（不含前缀和连字符），推荐 8~10。
func MakePrettyCode(id uint64, prefix rune, width int) string {
	if width <= 0 {
		width = 8
	}
	v := hash40(id) // 40 位有效载荷
	code := encodeBase32Fixed(v, width)
	// 每 4 字符分组提升可读性
	if len(code) > 4 {
		code = code[:4] + "-" + code[4:]
	}
	if prefix != 0 {
		return string(prefix) + "-" + code
	}
	return code
}

func hash40(id uint64) uint64 {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], id)
	h := fnv.New64a()
	_, _ = h.Write(b[:])
	v := h.Sum64() & ((1 << 40) - 1)
	return v
}

func encodeBase32Fixed(v uint64, width int) string {
	// 生成固定长度 width 的编码，左侧补 '0'
	buf := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		buf[i] = crockford[v&31]
		v >>= 5
	}
	return string(buf)
}

// 可选：计算 Crockford mod-37 校验字符用于完整字符串校验（默认未使用）。
// 规范引入 5 个额外符号 *~$=U 用于校验；此处内部使用避免界面混淆。
// https://www.crockford.com/base32.html
func crockfordChecksumMod37(num uint64) rune {
	alphabet := crockford + "*~$=U" // 32+5
	return rune(alphabet[num%37])
}

// ------------------------------
// 工具函数
// ------------------------------

// DecodePretty 去除前缀和连字符，返回纯符号字符串用于校验；不支持反向映射。
func DecodePretty(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.ReplaceAll(s, "-", "")
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s = s[:i]
	}
	return s
}

// IDTime 从 Snowflake ID 中提取生成时间。
func IDTime(id uint64) time.Time {
	// 时间戳位于高位
	ts := int64(id >> (bitsNode + bitsSeq))
	return time.UnixMilli(ts + defaultEpoch.UnixMilli())
}

// ParseNode 从 Snowflake ID 中提取节点 ID。
func ParseNode(id uint64) uint16 {
	return uint16((id >> bitsSeq) & maxNode)
}

// ParseSeq 从 Snowflake ID 中提取序列号。
func ParseSeq(id uint64) uint16 { return uint16(id & maxSeq) }
