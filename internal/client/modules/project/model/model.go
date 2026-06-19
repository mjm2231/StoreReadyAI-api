package model

const (
	ProjectStatusDraft     = "draft"
	ProjectStatusActive    = "active"
	ProjectStatusArchived  = "archived"
	ProjectPlatformIOS     = "ios"
	ProjectPlatformAndroid = "android"
)

type Project struct {
	ID       uint64 `gorm:"column:id;primaryKey;comment:项目ID" json:"id"`
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_projects_user,priority:1;comment:租户ID" json:"tenant_id"`
	UserID   uint64 `gorm:"column:user_id;not null;default:0;index:idx_projects_user,priority:2;comment:用户ID" json:"user_id"`

	Name        string `gorm:"column:name;type:varchar(100);not null;default:'';comment:项目名称" json:"name"`
	Description string `gorm:"column:description;type:text;comment:项目描述，用于补充说明应用功能、目标用户、核心卖点等" json:"description"`
	Platform    string `gorm:"column:platform;type:varchar(32);not null;default:'';comment:平台:ios/android" json:"platform"`
	Status      string `gorm:"column:status;type:varchar(32);not null;default:'draft';index:idx_projects_status,priority:1;comment:状态:draft/active/archived" json:"status"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0;index:idx_projects_user,priority:4;comment:创建时间戳秒" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;comment:更新时间戳秒" json:"updated_at"`
	DeletedAt uint64 `gorm:"column:deleted_at;not null;default:0;index:idx_projects_user,priority:3;index:idx_projects_status,priority:2;comment:软删除时间戳秒，0表示未删除" json:"deleted_at"`
}

func (Project) TableName() string { return "projects" }
