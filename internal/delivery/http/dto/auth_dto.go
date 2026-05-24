package dto

type LoginResponse struct {
	MFAToken     string `json:"mfa_token,omitempty"`
	Channel      string `json:"channel,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	BaseResponse
}

type MFAVerifyResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BaseResponse
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BaseResponse
}

type ChangePasswordResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BaseResponse
}

type PasswordResetVerifyResponse struct {
	ResetToken string `json:"reset_token"`
	BaseResponse
}

type ResetPasswordResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BaseResponse
}
