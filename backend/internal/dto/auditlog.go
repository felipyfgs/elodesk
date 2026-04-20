package dto

type AuditLogResp struct {
	ID         int64  `json:"id"`
	AccountID  int64  `json:"accountId"`
	UserID     *int64 `json:"userId,omitempty"`
	Action     string `json:"action"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   *int64 `json:"entityId,omitempty"`
	Metadata   any    `json:"metadata,omitempty"`
	IPAddress  string `json:"ipAddress,omitempty"`
	UserAgent  string `json:"userAgent,omitempty"`
	CreatedAt  string `json:"createdAt"`
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
