package dto

type LoginResponse struct {
	MFAToken string `json:"mfa_token"`
	Channel  string `json:"channel"`
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
