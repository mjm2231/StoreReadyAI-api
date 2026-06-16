package securityevent

import (
	"context"

	"storeready_ai/internal/security"

	"gorm.io/gorm"
)

// GormSecurityEventWriter 使用 GORM 将安全事件落库。
// 表结构见 sqls/21_security_events.sql
type GormSecurityEventWriter struct {
	DB *gorm.DB
}

type SecurityEventModel struct {
	ID uint64 `gorm:"primaryKey;autoIncrement;column:id"`

	Type     string `gorm:"column:type"`
	Severity string `gorm:"column:severity"`
	RID      string `gorm:"column:rid"`

	CreatedAt int64 `gorm:"column:created_at"`

	UID      string `gorm:"column:uid"`
	TenantID string `gorm:"column:tenant_id"`
	Role     string `gorm:"column:role"`

	IP     string `gorm:"column:ip"`
	UA     string `gorm:"column:ua"`
	Device string `gorm:"column:device"`

	Route  string `gorm:"column:route"`
	Method string `gorm:"column:method"`

	Details string `gorm:"column:details"`
}

func (SecurityEventModel) TableName() string { return "security_events" }

func NewGormSecurityEventWriter(db *gorm.DB) *GormSecurityEventWriter {
	return &GormSecurityEventWriter{DB: db}
}

func (w *GormSecurityEventWriter) Write(ctx context.Context, ev security.SecurityEvent) error {
	if w == nil || w.DB == nil {
		return nil
	}
	m := SecurityEventModel{
		Type:      ev.Type,
		Severity:  string(ev.Severity),
		RID:       ev.RID,
		CreatedAt: ev.CreatedAt,

		UID:      ev.UID,
		TenantID: ev.TenantID,
		Role:     ev.Role,

		IP:     ev.IP,
		UA:     ev.UA,
		Device: ev.Device,

		Route:  ev.Route,
		Method: ev.Method,

		Details: ev.Details,
	}
	return w.DB.WithContext(ctx).Create(&m).Error
}
