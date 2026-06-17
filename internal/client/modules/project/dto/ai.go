package dto

// GenerateProjectStoreInfoReq AI 生成上架资料请求。
//
// MVP：只需要 project_id，后端会根据当前项目和已有上架资料组织 Prompt。
// 注意：project_id 必须是数字，前端不要传字符串。
type GenerateProjectStoreInfoReq struct {
	ProjectID uint64 `json:"project_id" binding:"required,min=1"`
}

// GenerateProjectStoreInfoResp AI 生成上架资料响应。
//
// MVP 只生成 5 个文本字段，前端收到后回填表单，用户确认后再手动保存。
type GenerateProjectStoreInfoResp struct {
	AppName          string `json:"app_name"`
	Subtitle         string `json:"subtitle"`
	ShortDescription string `json:"short_description"`
	FullDescription  string `json:"full_description"`
	Keywords         string `json:"keywords"`
}
