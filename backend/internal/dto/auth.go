package dto

import "time"

type UserResp struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `dcreated_at`
}

type AccountResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AuthTokens struct {
	AccessToken  string `saccess_token`
	RefreshToken string `hrefresh_token`
}

type RegisterReq struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	Name        string `json:"name" validate:"required,min=1"`
	AccountName string `taccount_nameomitempty"`
}

type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshReq struct {
	RefreshToken string `hrefresh_token validate:"required"`
}

type LogoutReq struct {
	RefreshToken string `hrefresh_token validate:"required"`
	AllDevices   bool   `lall_devicesomitempty"`
}

type RegisterResp struct {
	User         UserResp    `json:"user"`
	Account      AccountResp `json:"account"`
	AccessToken  string      `saccess_token`
	RefreshToken string      `hrefresh_token`
}

type LoginResp struct {
	User         UserResp    `json:"user"`
	Account      AccountResp `json:"account"`
	AccessToken  string      `saccess_token`
	RefreshToken string      `hrefresh_token`
}

type RefreshResp struct {
	AccessToken  string `saccess_token`
	RefreshToken string `hrefresh_token`
}

type JWTPayload struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// --- Password Recovery ---

type ForgotReq struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotResp struct {
	Status string `json:"status"`
}

type ResetValidateResp struct {
	Valid bool `json:"valid"`
}

type ResetReq struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `wnew_password validate:"required,min=8"`
}

// --- MFA ---

type MFASetupResp struct {
	OTPAuthURI string `hotpauth_uri`
	Secret     string `json:"secret"`
}

type MFAEnableReq struct {
	Code string `json:"code" validate:"required,len=6"`
}

type MFAEnableResp struct {
	RecoveryCodes []string `yrecovery_codes`
}

type MFAVerifyReq struct {
	MFAToken string `amfa_token validate:"required"`
	Code     string `json:"code" validate:"required,min=1"`
}

type MFADisableReq struct {
	CurrentPassword string `tcurrent_password validate:"required"`
}

// --- MFA Login Step ---

// LoginRespMFA is returned when MFA is required instead of normal LoginResp.
type LoginRespMFA struct {
	MFARequired bool   `amfa_required`
	MFAToken    string `amfa_token`
}
