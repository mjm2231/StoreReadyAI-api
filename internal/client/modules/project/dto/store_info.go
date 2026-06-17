package dto

// GetProjectDetailReq 项目详情请求。
type GetProjectDetailReq struct {
	ProjectID uint64 `json:"project_id" binding:"required,min=1"`
}

// SaveProjectStoreInfoReq 保存项目上架资料请求。
type SaveProjectStoreInfoReq struct {
	ProjectID uint64 `json:"project_id" binding:"required,min=1"`

	AppName          string `json:"app_name" binding:"omitempty,max=100"`
	Subtitle         string `json:"subtitle" binding:"omitempty,max=255"`
	Keywords         string `json:"keywords" binding:"omitempty,max=500"`
	ShortDescription string `json:"short_description" binding:"omitempty,max=500"`
	FullDescription  string `json:"full_description"`
	Category         string `json:"category" binding:"omitempty,max=100"`
	ContentRating    string `json:"content_rating" binding:"omitempty,max=100"`

	PrivacyPolicyURL string `json:"privacy_policy_url" binding:"omitempty,max=512"`
	SupportURL       string `json:"support_url" binding:"omitempty,max=512"`
	MarketingURL     string `json:"marketing_url" binding:"omitempty,max=512"`
	Copyright        string `json:"copyright" binding:"omitempty,max=255"`
	ContactEmail     string `json:"contact_email" binding:"omitempty,max=255"`

	Status string `json:"status" binding:"omitempty,oneof=draft ready"`
}

// ProjectStoreInfoItem 项目上架资料响应对象。
type ProjectStoreInfoItem struct {
	ID        uint64 `json:"id"`
	ProjectID uint64 `json:"project_id"`

	AppName          string `json:"app_name"`
	Subtitle         string `json:"subtitle"`
	Keywords         string `json:"keywords"`
	ShortDescription string `json:"short_description"`
	FullDescription  string `json:"full_description"`
	Category         string `json:"category"`
	ContentRating    string `json:"content_rating"`

	PrivacyPolicyURL string `json:"privacy_policy_url"`
	SupportURL       string `json:"support_url"`
	MarketingURL     string `json:"marketing_url"`
	Copyright        string `json:"copyright"`
	ContactEmail     string `json:"contact_email"`

	Status    string `json:"status"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
}

// ProjectDetailResp 项目详情响应。
type ProjectDetailResp struct {
	Project   ProjectItem           `json:"project"`
	StoreInfo *ProjectStoreInfoItem `json:"store_info,omitempty"`
}

// SaveProjectStoreInfoResp 保存项目上架资料响应。
type SaveProjectStoreInfoResp struct {
	StoreInfo ProjectStoreInfoItem `json:"store_info"`
}
