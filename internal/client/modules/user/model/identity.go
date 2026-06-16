package model

import "gorm.io/datatypes"

// UserIdentity corresponds to table `user_identities`.
type UserIdentity struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键" json:"id"`
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_identity_user_id,priority:1;uniqueIndex:uk_identity_provider_uid,priority:1;comment:租户ID" json:"tenant_id"`
	UserID   uint64 `gorm:"column:user_id;not null;index:idx_identity_user_id,priority:2;comment:关联users.id" json:"user_id"`

	Provider    string `gorm:"column:provider;type:varchar(32);not null;index:idx_identity_provider,priority:1;uniqueIndex:uk_identity_provider_uid,priority:2;comment:提供方:password/google/apple/github/anonymous" json:"provider"`
	ProviderUID string `gorm:"column:provider_uid;type:varchar(255);not null;index:idx_identity_provider,priority:2;uniqueIndex:uk_identity_provider_uid,priority:3;comment:提供方用户唯一ID（password可使用规范化邮箱）" json:"provider_uid"`

	Email      *string        `gorm:"column:email;type:varchar(255);comment:提供方邮箱" json:"email,omitempty"`
	RawProfile datatypes.JSON `gorm:"column:raw_profile;type:json;comment:原始profile" json:"-"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0;comment:创建时间戳秒" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;comment:更新时间戳秒" json:"updated_at"`
}

func (UserIdentity) TableName() string { return "user_identities" }

const (
	IdentityProviderPassword  = "password"
	IdentityProviderGoogle    = "google"
	IdentityProviderApple     = "apple"
	IdentityProviderGitHub    = "github"
	IdentityProviderAnonymous = "anonymous"
)
