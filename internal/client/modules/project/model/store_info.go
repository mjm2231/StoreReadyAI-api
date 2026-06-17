package model

const (
	ProjectStoreInfoStatusDraft = "draft"
	ProjectStoreInfoStatusReady = "ready"
)

// ProjectStoreInfo 项目上架资料。
//
// 说明：
//  1. projects 表只保存项目基础信息。
//  2. project_store_infos 单独保存上架资料，避免后续字段变多污染 projects 表。
//  3. 当前 MVP 只做保存/读取，不做发布、不做审核状态流。
type ProjectStoreInfo struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:上架资料ID" json:"id"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;uniqueIndex:uk_project_store_info,priority:1;index:idx_project_store_infos_user,priority:1;index:idx_project_store_infos_project,priority:1;comment:租户ID" json:"tenant_id"`
	UserID    uint64 `gorm:"column:user_id;not null;default:0;uniqueIndex:uk_project_store_info,priority:2;index:idx_project_store_infos_user,priority:2;comment:用户ID" json:"user_id"`
	ProjectID uint64 `gorm:"column:project_id;not null;default:0;uniqueIndex:uk_project_store_info,priority:3;index:idx_project_store_infos_project,priority:2;comment:项目ID" json:"project_id"`

	AppName          string `gorm:"column:app_name;type:varchar(100);not null;default:'';comment:App名称" json:"app_name"`
	Subtitle         string `gorm:"column:subtitle;type:varchar(255);not null;default:'';comment:副标题" json:"subtitle"`
	Keywords         string `gorm:"column:keywords;type:varchar(500);not null;default:'';comment:关键词，逗号分隔" json:"keywords"`
	ShortDescription string `gorm:"column:short_description;type:varchar(500);not null;default:'';comment:短描述" json:"short_description"`
	FullDescription  string `gorm:"column:full_description;type:text;comment:完整描述" json:"full_description"`
	Category         string `gorm:"column:category;type:varchar(100);not null;default:'';comment:应用分类" json:"category"`
	ContentRating    string `gorm:"column:content_rating;type:varchar(100);not null;default:'';comment:内容分级" json:"content_rating"`

	PrivacyPolicyURL string `gorm:"column:privacy_policy_url;type:varchar(512);not null;default:'';comment:隐私政策URL" json:"privacy_policy_url"`
	SupportURL       string `gorm:"column:support_url;type:varchar(512);not null;default:'';comment:支持URL" json:"support_url"`
	MarketingURL     string `gorm:"column:marketing_url;type:varchar(512);not null;default:'';comment:营销URL" json:"marketing_url"`
	Copyright        string `gorm:"column:copyright;type:varchar(255);not null;default:'';comment:版权信息" json:"copyright"`
	ContactEmail     string `gorm:"column:contact_email;type:varchar(255);not null;default:'';comment:联系邮箱" json:"contact_email"`

	Status    string `gorm:"column:status;type:varchar(32);not null;default:'draft';comment:状态:draft/ready" json:"status"`
	CreatedAt uint64 `gorm:"column:created_at;not null;default:0;comment:创建时间戳秒" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;index:idx_project_store_infos_user,priority:4;comment:更新时间戳秒" json:"updated_at"`
	DeletedAt uint64 `gorm:"column:deleted_at;not null;default:0;uniqueIndex:uk_project_store_info,priority:4;index:idx_project_store_infos_user,priority:3;index:idx_project_store_infos_project,priority:3;comment:软删除时间戳秒，0表示未删除" json:"deleted_at"`
}

func (ProjectStoreInfo) TableName() string { return "project_store_infos" }
