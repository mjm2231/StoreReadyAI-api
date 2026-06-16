package model

// AuditLog 对应表：audit_logs
//
// 说明：
// 1. 用于后台关键操作审计，不用于客户端普通事件记录；
// 2. created_at 使用 unix 秒级时间戳，与当前项目 admin 模块保持一致；
// 3. query/body/resp/risk_reasons 为摘要或 JSON 字符串，写入前应先做脱敏与截断。
type AuditLog struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	RID       string `gorm:"column:rid;type:varchar(64);not null;default:''" json:"rid"`
	TraceID   string `gorm:"column:trace_id;type:varchar(64);not null;default:''" json:"trace_id"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0" json:"created_at"`

	UID      string `gorm:"column:uid;type:varchar(32);not null;default:''" json:"uid"`
	TenantID string `gorm:"column:tenant_id;type:varchar(32);not null;default:''" json:"tenant_id"`
	Role     string `gorm:"column:role;type:varchar(64);not null;default:''" json:"role"`
	Scopes   string `gorm:"column:scopes;type:varchar(512);not null;default:''" json:"scopes"`

	Action       string `gorm:"column:action;type:varchar(128);not null;default:''" json:"action"`
	ResourceType string `gorm:"column:resource_type;type:varchar(64);not null;default:''" json:"resource_type"`
	ResourceID   string `gorm:"column:resource_id;type:varchar(64);not null;default:''" json:"resource_id"`

	IP     string `gorm:"column:ip;type:varchar(64);not null;default:''" json:"ip"`
	UA     string `gorm:"column:ua;type:varchar(512);not null;default:''" json:"ua"`
	Device string `gorm:"column:device;type:varchar(128);not null;default:''" json:"device"`
	Refer  string `gorm:"column:refer;type:varchar(512);not null;default:''" json:"refer"`

	Success    uint8  `gorm:"column:success;type:tinyint(1);not null;default:0" json:"success"`
	HTTPStatus int32  `gorm:"column:http_status;type:int;not null;default:0" json:"http_status"`
	ErrCode    string `gorm:"column:err_code;type:varchar(64);not null;default:''" json:"err_code"`
	LatencyMS  int64  `gorm:"column:latency_ms;type:bigint;not null;default:0" json:"latency_ms"`

	Method string `gorm:"column:method;type:varchar(16);not null;default:''" json:"method"`
	Path   string `gorm:"column:path;type:varchar(256);not null;default:''" json:"path"`

	QuerySummary string `gorm:"column:query_summary;type:text" json:"query_summary"`
	BodySummary  string `gorm:"column:body_summary;type:text" json:"body_summary"`
	RespSummary  string `gorm:"column:resp_summary;type:text" json:"resp_summary"`

	RequestSizeB  int64 `gorm:"column:request_size_b;type:bigint;not null;default:0" json:"request_size_b"`
	ResponseSizeB int64 `gorm:"column:response_size_b;type:bigint;not null;default:0" json:"response_size_b"`

	RiskScore   int64  `gorm:"column:risk_score;type:bigint;not null;default:0" json:"risk_score"`
	RiskAction  string `gorm:"column:risk_action;type:varchar(32);not null;default:''" json:"risk_action"`
	RiskReasons string `gorm:"column:risk_reasons;type:text" json:"risk_reasons"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

func (l AuditLog) IsSuccess() bool {
	return l.Success == 1
}

func (l AuditLog) HasError() bool {
	return l.ErrCode != "" || l.HTTPStatus >= 400
}

func (l AuditLog) HasRisk() bool {
	return l.RiskScore > 0 || l.RiskAction != "" || l.RiskReasons != ""
}
