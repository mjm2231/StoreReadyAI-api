package db

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"storeready_ai/internal/config"
)

func NewGorm(cfg *config.Config) (*gorm.DB, func() error, error) {
	var dial gorm.Dialector

	switch cfg.DB.Driver {
	case "sqlite":
		dial = sqlite.Open(cfg.DB.DSN)
	// case "mysql": dial = mysql.Open(cfg.DB.DSN)
	// case "postgres": dial = postgres.Open(cfg.DB.DSN)
	default:
		dial = sqlite.Open(cfg.DB.DSN)
	}

	gdb, err := gorm.Open(dial, &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Silent), // dev 可调到 Info
		PrepareStmt: true,                                  // 性能：预编译缓存
	})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DB.ConnMaxLifetime) * time.Second)

	stop := func() error { return sqlDB.Close() }
	return gdb, stop, nil
}
