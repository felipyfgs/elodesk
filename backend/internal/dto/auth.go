package dto

import "time"

type UserResp struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type AccountResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RegisterReq struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	Name        string `json:"name" validate:"required,min=1"`
	AccountName string `json:"account_name,omitempty"`
}

type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	AllDevices   bool   `json:"all_devices,omitempty"`
}

type RegisterResp struct {
	User         UserResp    `json:"user"`
	Account      AccountResp `json:"account"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

type LoginResp struct {
	User         UserResp    `json:"user"`
	Account      AccountResp `json:"account"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

type RefreshResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTPayload struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

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
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type MFASetupResp struct {
	OTPAuthURI string `json:"otpauth_uri"`
	Secret     string `json:"secret"`
}

type MFAEnableReq struct {
	Code string `json:"code" validate:"required,len=6"`
}

type MFAEnableResp struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

type MFAVerifyReq struct {
	MFAToken string `json:"mfa_token" validate:"required"`
	Code     string `json:"code" validate:"required,min=1"`
}

type MFADisableReq struct {
	CurrentPassword string `json:"current_password" validate:"required"`
}

type LoginRespMFA struct {
	MFARequired bool   `json:"mfa_required"`
	MFAToken    string `json:"mfa_token"`
}
