package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	infrolog "storeready_ai/internal/infra/log"

	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLOptions 用于控制企业级场景下“可选但不一定默认开启”的参数。
//
// 说明：
//   - PrepareStmt：用内存换性能（需要 profile 后再决定）；开发环境可开启减少 prepare 开销，生产环境建议谨慎。
//   - DisableDefaultTransaction：写多场景可提升吞吐，但若业务依赖隐式事务需谨慎。
//   - SkipPing：是否跳过启动时 ping（默认不跳过，企业最佳实践是启动时带超时 ping）。

type MySQLOptions struct {
	PrepareStmt               bool
	SlowThreshold             time.Duration
	LogLevel                  logger.LogLevel
	DisableDefaultTransaction bool
	SkipPing                  bool
}

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// OpenMySQL 打开 GORM MySQL 连接，并应用企业级默认优化：
//   - 连接池调优
//   - 可选 PrepareStmt
//   - 慢 SQL 日志阈值
//   - 启动时带超时的 Ping（企业最佳实践）
//
// 返回值：(*gorm.DB, stop)。stop 用于关闭底层 sqlDB（幂等），调用方在退出时调用。
func OpenMySQL(ctx context.Context, dsn string, pool PoolConfig, opt MySQLOptions) (*gorm.DB, func() error, error) {
	if dsn == "" {
		return nil, nil, errors.New("mysql dsn is empty")
	}

	// 默认值
	if opt.SlowThreshold <= 0 {
		opt.SlowThreshold = 200 * time.Millisecond
	}
	if opt.LogLevel == 0 {
		opt.LogLevel = logger.Warn
	}

	// 将 GORM 的日志桥接到企业级日志（zap）。
	// 提示：生产环境建议保持 Warn，避免日志洪泛。
	gormLogger := logger.New(
		newGormWriter(infrolog.Writer(zapcore.WarnLevel)),
		logger.Config{
			SlowThreshold:             opt.SlowThreshold,
			LogLevel:                  opt.LogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	dial := mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
	})

	gdb, err := gorm.Open(dial, &gorm.Config{
		Logger:                 gormLogger,
		PrepareStmt:            opt.PrepareStmt,
		DisableAutomaticPing:   true, // 我们自己带超时做 Ping
		SkipDefaultTransaction: opt.DisableDefaultTransaction,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open mysql: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get sql db: %w", err)
	}

	// stop 用于关闭底层 sql.DB（幂等）。
	var once sync.Once
	stop := func() error {
		var cerr error
		once.Do(func() {
			cerr = sqlDB.Close()
		})
		return cerr
	}

	// 连接池
	if pool.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.MaxIdleConns >= 0 {
		sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(pool.ConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(pool.ConnMaxIdleTime)
	}

	// 启动时带超时 Ping（企业最佳实践）
	if !opt.SkipPing {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			_ = stop()
			return nil, nil, fmt.Errorf("mysql ping: %w", err)
		}
	}

	return gdb, stop, nil
}

// gormWriter 实现 gorm.io/gorm/logger 的 logger.Writer 接口。
// 用于把 GORM 的 printf 风格日志桥接到 zap（通过 io.Writer 输出）。

type gormWriter struct {
	w io.Writer
}

func newGormWriter(w io.Writer) *gormWriter { return &gormWriter{w: w} }

func (gw *gormWriter) Printf(format string, args ...any) {
	// 保持简单：由 zap 的 formatter 负责最终输出格式。
	_, _ = fmt.Fprintf(gw.w, format, args...)
	_, _ = fmt.Fprint(gw.w, "\n")
}
