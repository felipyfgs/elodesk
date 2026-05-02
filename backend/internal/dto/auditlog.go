package dto

type AuditLogResp struct {
	ID         int64  `json:"id"`
	AccountID  int64  `json:"account_id"`
	UserID     *int64 `json:"user_id,omitempty"`
	Action     string `json:"action"`
	EntityType string `json:"entity_type,omitempty"`
	EntityID   *int64 `json:"entity_id,omitempty"`
	Metadata   any    `json:"metadata,omitempty"`
	IPAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	CreatedAt  string `json:"created_at"`
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
