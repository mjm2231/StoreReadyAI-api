package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

// MigrateConfig 用于控制迁移行为。
// MigrationsDir 为空时默认使用 "./migrations"。
//
// 说明：这里使用显式迁移（企业级推荐），而不是 GORM AutoMigrate。
// - 变更可重复执行、可审计
// - 支持回滚
// - 更适合 CI/CD 流程
//
// 本地开发可在启动阶段调用 Up()；生产环境更建议在部署流水线中
// 将数据库迁移作为独立步骤执行。
type MigrateConfig struct {
	MigrationsDir string
	// TableName 用于覆盖数据库中的迁移记录表名。
	// 默认值为 "schema_migrations"。
	TableName string
	// 脏状态请通过 Version() 返回值感知。
}

// Up 执行全部向上迁移。
// 若当前没有可执行变更，则返回 nil。
func Up(gdb *gorm.DB, cfg MigrateConfig) error {
	m, closeFn, err := newMigrator(gdb, cfg)
	if err != nil {
		return err
	}
	defer closeFn()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return err
	}
	return nil
}

// DownSteps 回滚 N 个版本（N>=1）。
// 若当前没有可回滚变更，则返回 nil。
func DownSteps(gdb *gorm.DB, cfg MigrateConfig, n int) error {
	if n <= 0 {
		return fmt.Errorf("invalid down steps: %d", n)
	}
	m, closeFn, err := newMigrator(gdb, cfg)
	if err != nil {
		return err
	}
	defer closeFn()

	if err := m.Steps(-n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return err
	}
	return nil
}

// Version 返回当前 schema 版本及脏状态。
func Version(gdb *gorm.DB, cfg MigrateConfig) (version uint, dirty bool, err error) {
	m, closeFn, err := newMigrator(gdb, cfg)
	if err != nil {
		return 0, false, err
	}
	defer closeFn()

	v, d, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return v, d, nil
}

// Force 强制将迁移版本设置为指定值，并清除 dirty 状态。
//
// 适用场景：
// - 某个迁移执行失败后，schema_migrations 被标记为 dirty
// - 已手动修复 SQL 文件或数据库状态，需要把版本重置到一个确定值后重新执行迁移
//
// 注意：
// - Force 只修改迁移记录，不会自动执行 up/down SQL
// - 调用前应先确认数据库表结构与目标版本匹配
func Force(gdb *gorm.DB, cfg MigrateConfig, version int) error {
	if version < 0 {
		return fmt.Errorf("invalid force version: %d", version)
	}

	m, closeFn, err := newMigrator(gdb, cfg)
	if err != nil {
		return err
	}
	defer closeFn()

	if err := m.Force(version); err != nil {
		return err
	}
	return nil
}

// newMigrator 基于现有 *gorm.DB 创建 migrate.Migrate 实例。
// 数据源使用 file://，数据库驱动使用 mysql。
func newMigrator(gdb *gorm.DB, cfg MigrateConfig) (*migrate.Migrate, func(), error) {
	if gdb == nil {
		return nil, func() {}, errors.New("gorm db is nil")
	}

	dir := strings.TrimSpace(cfg.MigrationsDir)
	if dir == "" {
		dir = "./migrations"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, func() {}, fmt.Errorf("abs migrations dir: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, func() {}, fmt.Errorf("get sql db: %w", err)
	}

	dbCfg := &mysql.Config{}
	if cfg.TableName != "" {
		dbCfg.MigrationsTable = cfg.TableName
	}

	drv, err := mysql.WithInstance(sqlDB, dbCfg)
	if err != nil {
		return nil, func() {}, fmt.Errorf("mysql migrate driver: %w", err)
	}

	// 校验迁移目录是否存在且为目录。
	st, err := os.Stat(abs)
	if err != nil {
		return nil, func() {}, fmt.Errorf("stat migrations dir: %w", err)
	}
	if !st.IsDir() {
		return nil, func() {}, fmt.Errorf("migrations path is not a directory: %s", abs)
	}

	srcURL := "file://" + abs
	m, err := migrate.NewWithDatabaseInstance(srcURL, "mysql", drv)
	if err != nil {
		return nil, func() {}, fmt.Errorf("new migrator: %w", err)
	}

	closeFn := func() {
		// 不在这里关闭底层数据库连接，避免影响同一 *gorm.DB 的后续迁移/业务复用。
		// 迁移使用的主数据库连接由应用生命周期统一管理。
	}

	return m, closeFn, nil
}
