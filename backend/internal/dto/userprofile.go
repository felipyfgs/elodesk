package dto

type UpdateProfileReq struct {
	Name            *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Email           *string `json:"email,omitempty" validate:"omitempty,email"`
	AvatarURL       *string `json:"avatar_url,omitempty"`
	CurrentPassword *string `json:"current_password,omitempty"`
	NewPassword     *string `json:"new_password,omitempty" validate:"omitempty,min=8"`
}
