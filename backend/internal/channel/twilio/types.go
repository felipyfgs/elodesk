package twilio

const (
	APIBase     = "https://api.twilio.com/2010-04-01"
	ContentBase = "https://content.twilio.com/v1"

	HeaderSignature = "X-Twilio-Signature"
	WhatsappPrefix  = "whatsapp:"
)

var APIBaseOverride string

type SendResponse struct {
	SID          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    any    `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ContentTemplate struct {
	SID            string         `json:"sid"`
	FriendlyName   string         `json:"friendly_name"`
	Language       string         `json:"language"`
	DateCreated    string         `json:"date_created"`
	DateUpdated    string         `json:"date_updated"`
	Variables      map[string]any `json:"variables,omitempty"`
	Types          map[string]any `json:"types,omitempty"`
	ApprovalStatus string         `json:"approval_status,omitempty"`
}

type contentListResponse struct {
	Contents []ContentTemplate `json:"contents"`
	Meta     struct {
		NextPageURL string `json:"next_page_url"`
	} `json:"meta"`
}

type accountInfoResponse struct {
	SID    string `json:"sid"`
	Status string `json:"status"`
}
