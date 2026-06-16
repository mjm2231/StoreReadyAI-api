package security

import (
	"sort"
	"strings"
	"time"
)

// Severity 风险信号严重级别。
//
// 约定：
//   - info：仅记录
//   - low：低风险（轻度异常/轻度限流）
//   - medium：中风险（需要验证码/风控升级）
//   - high：高风险（短期封禁/明显攻击）
//   - critical：严重风险（疑似入侵/系统级异常）
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Signal 一次风险信号。
//
// name：信号名（建议可枚举，例如 ip_blocked、ua_scanner、login_failed_burst）
// score：该信号贡献的风险分（>=0）
// severity：严重级别
// tags：用于聚合分析的标签（例如 route=/auth/login、dim=ip）
// meta：附加信息（可用于审计/安全事件，建议少量）
// ts：信号发生时间
//
// 注意：
//   - 该结构用于“请求内/短周期”的风险分聚合，不做长期画像；长期画像建议落库/队列处理。
type Signal struct {
	Name     string
	Score    int64
	Severity Severity
	Tags     map[string]string
	Meta     map[string]any
	TS       time.Time
}

// RiskScore 风控分聚合器（面向一次请求或一个短周期窗口）。
//
// 设计目标：
//   - 轻量：不依赖外部组件
//   - 可组合：多个中间件/规则都可向同一实例 Add 信号
//   - 可解释：能输出 reasons 便于调试/审计
//
// 用法示例：
//
//	rs := security.NewRiskScore()
//	rs.AddN("ua_scanner", 60, security.SeverityHigh, map[string]string{"route": "/"}, nil)
//	rs.AddN("ip_burst", 30, security.SeverityMedium, nil, nil)
//	total := rs.Total()
//	action := security.Decide(total, security.Thresholds{Light:50, Medium:90, Heavy:140})
type RiskScore struct {
	signals []Signal
	total   int64
	maxSev  Severity
}

// NewRiskScore 创建一个新的 RiskScore。
func NewRiskScore() *RiskScore {
	return &RiskScore{maxSev: SeverityInfo}
}

// Add 添加一个信号。
func (r *RiskScore) Add(s Signal) {
	if r == nil {
		return
	}
	if s.Score < 0 {
		s.Score = 0
	}
	if s.TS.IsZero() {
		s.TS = time.Now()
	}
	if strings.TrimSpace(string(s.Severity)) == "" {
		s.Severity = SeverityInfo
	}
	r.signals = append(r.signals, s)
	r.total += s.Score
	if severityRank(s.Severity) > severityRank(r.maxSev) {
		r.maxSev = s.Severity
	}
}

// AddN 便捷添加：用字段快速构造 Signal。
func (r *RiskScore) AddN(name string, score int64, sev Severity, tags map[string]string, meta map[string]any) {
	r.Add(Signal{
		Name:     strings.TrimSpace(name),
		Score:    score,
		Severity: sev,
		Tags:     tags,
		Meta:     meta,
		TS:       time.Now(),
	})
}

// Total 返回累计风险分。
func (r *RiskScore) Total() int64 {
	if r == nil {
		return 0
	}
	return r.total
}

// MaxSeverity 返回最高严重级别。
func (r *RiskScore) MaxSeverity() Severity {
	if r == nil {
		return SeverityInfo
	}
	return r.maxSev
}

// Signals 返回原始信号（拷贝）。
func (r *RiskScore) Signals() []Signal {
	if r == nil {
		return nil
	}
	out := make([]Signal, len(r.signals))
	copy(out, r.signals)
	return out
}

// Reason 风险原因（用于对外解释/日志/审计）。
type Reason struct {
	Name     string   `json:"name"`
	Score    int64    `json:"score"`
	Severity Severity `json:"severity"`
	At       int64    `json:"at"` // unix 秒
}

// Reasons 返回按贡献度排序的原因列表。
//
// limit<=0 表示返回全部。
func (r *RiskScore) Reasons(limit int) []Reason {
	if r == nil || len(r.signals) == 0 {
		return nil
	}
	arr := make([]Reason, 0, len(r.signals))
	for _, s := range r.signals {
		arr = append(arr, Reason{
			Name:     s.Name,
			Score:    s.Score,
			Severity: s.Severity,
			At:       s.TS.Unix(),
		})
	}
	// score 降序；同分按时间升序（更早的在前，方便定位首个触发点）
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].Score != arr[j].Score {
			return arr[i].Score > arr[j].Score
		}
		return arr[i].At < arr[j].At
	})
	if limit > 0 && len(arr) > limit {
		return arr[:limit]
	}
	return arr
}

// Thresholds 风险分阈值配置。
//
// Light/Medium/Heavy 分别对应：
//   - 轻度：限流
//   - 中度：要求验证码/滑块
//   - 重度：封禁
//
// BlockTTL 可用于上层策略决定封禁时长（可选）。
type Thresholds struct {
	Light    int64
	Medium   int64
	Heavy    int64
	BlockTTL time.Duration
}

// Decision 风控决策动作。
type Decision string

const (
	DecisionAllow           Decision = "allow"
	DecisionRateLimited     Decision = "rate_limited"
	DecisionCaptchaRequired Decision = "captcha_required"
	DecisionBlocked         Decision = "blocked"
)

// Decide 根据累计分与阈值做决策。
//
// 规则：
//   - total >= Heavy  => blocked
//   - total >= Medium => captcha_required
//   - total >= Light  => rate_limited
//   - 否则 allow
func Decide(total int64, th Thresholds) Decision {
	if th.Heavy > 0 && total >= th.Heavy {
		return DecisionBlocked
	}
	if th.Medium > 0 && total >= th.Medium {
		return DecisionCaptchaRequired
	}
	if th.Light > 0 && total >= th.Light {
		return DecisionRateLimited
	}
	return DecisionAllow
}

func severityRank(s Severity) int {
	s = Severity(strings.ToLower(strings.TrimSpace(string(s))))
	switch s {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		fallthrough
	default:
		return 1
	}
}
