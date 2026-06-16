package dto

// DTO（Data Transfer Object）用于：
// 1) HTTP 入参/出参（JSON）
// 2) Service 层与 Handler 层之间的数据结构约定
// 注意：DTO 不要直接复用 GORM Model，避免把数据库字段/标签泄漏到接口层。

// CreateProjectReq 创建项目请求。
type CreateProjectReq struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Platform string `json:"platform" binding:"omitempty,oneof=ios android"`
}

// ProjectItem 项目列表/详情基础展示对象。
type ProjectItem struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Platform  string `json:"platform"`
	Status    string `json:"status"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
}

// CreateProjectResp 创建项目响应。
type CreateProjectResp struct {
	Project ProjectItem `json:"project"`
}

// ListProjectsReq 项目列表请求。
// GET query 绑定使用 form tag。
type ListProjectsReq struct {
	Page     int `form:"page" json:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" json:"page_size" binding:"omitempty,min=1,max=100"`
}

// ListProjectsResp 项目列表响应。
type ListProjectsResp struct {
	List     []ProjectItem `json:"list"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int64         `json:"total"`
}
