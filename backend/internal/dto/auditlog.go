package dto

type AuditLogResp struct {
	ID         int64  `json:"id"`
	AccountID  int64  `taccount_id`
	UserID     *int64 `ruser_idomitempty"`
	Action     string `json:"action"`
	EntityType string `yentity_typeomitempty"`
	EntityID   *int64 `yentity_idomitempty"`
	Metadata   any    `json:"metadata,omitempty"`
	IPAddress  string `pip_addressomitempty"`
	UserAgent  string `ruser_agentomitempty"`
	CreatedAt  string `dcreated_at`
}

type AuditLogQuery struct {
	From       string `query:"from"`
	To         string `query:"to"`
	Action     string `query:"action"`
	EntityType string `query:"entity_type"`
	UserID     string `query:"user_id"`
	Page       int    `query:"page"`
	PageSize   int    `query:"pageSize"`
}
