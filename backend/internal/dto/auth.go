package dto

import "time"

type UserResp struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type AccountResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type RegisterReq struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	Name        string `json:"name" validate:"required,min=1"`
	AccountName string `json:"accountName,omitempty"`
}

type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutReq struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
	AllDevices   bool   `json:"allDevices,omitempty"`
}

type RegisterResp struct {
	User         UserResp    `json:"user"`
	Account      AccountResp `json:"account"`
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
}

type LoginResp struct {
	User         UserResp    `json:"user"`
	Account      AccountResp `json:"account"`
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
}

type RefreshResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
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
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// --- MFA ---

type MfaSetupResp struct {
	OTPAuthURI string `json:"otpauthUri"`
	Secret     string `json:"secret"`
}

type MfaEnableReq struct {
	Code string `json:"code" validate:"required,len=6"`
}

type MfaEnableResp struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type MfaVerifyReq struct {
	MfaToken string `json:"mfaToken" validate:"required"`
	Code     string `json:"code" validate:"required,min=1"`
}

type MfaDisableReq struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
}

// --- MFA Login Step ---

// LoginRespMfa is returned when MFA is required instead of normal LoginResp.
type LoginRespMfa struct {
	MfaRequired bool   `json:"mfaRequired"`
	MfaToken    string `json:"mfaToken"`
}
