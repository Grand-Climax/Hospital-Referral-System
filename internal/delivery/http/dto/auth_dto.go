package dto

type LoginResponse struct {
	MFAToken         string `json:"mfa_token"`
	MFASetupRequired bool   `json:"mfa_setup_required"`
	BaseResponse
}

type MFAVerifyResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BaseResponse
}

type MFASetupResponse struct {
	OTPAuthURL string `json:"otpauth_url"`
	BaseResponse
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BaseResponse
}
