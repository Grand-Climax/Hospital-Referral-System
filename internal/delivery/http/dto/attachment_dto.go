package dto

import "time"

type CreateAttachmentRequest struct {
	FileName   string `json:"file_name" binding:"required" example:"xray_chest.dcm"`
	FileType   string `json:"file_type" binding:"required" example:"application/dicom"`
	PublicID   string `json:"public_id" binding:"required" example:"referrals/hosp_123/ref_456/abc123"`
	FileURL    string `json:"file_url" binding:"required" example:"https://res.cloudinary.com/..."`
	Category   string `json:"category" example:"RADIOLOGY"`
	FileSize   int64  `json:"file_size" example:"4587210"`
}

type BulkAttachmentRequest struct {
	Attachments []CreateAttachmentRequest `json:"attachments" binding:"required,dive"`
}

type AttachmentResponse struct {
	ID          string                 `json:"id"`
	ReferralID  string                 `json:"referral_id"`
	FileName    string                 `json:"file_name"`
	FileType    string                 `json:"file_type"`
	FileSize    int64                  `json:"file_size"`
	Category    string                 `json:"category"`
	StoragePath string                 `json:"file_url"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	UploadedAt  time.Time              `json:"uploaded_at"`
	BaseResponse
}

type AttachmentListResponse struct {
	Data []AttachmentResponse `json:"data"`
	BaseResponse
}

type UploadSignatureResponse struct {
	Signature string `json:"signature"`
	Timestamp int64  `json:"timestamp"`
	APIKey    string `json:"api_key"`
	CloudName string `json:"cloud_name"`
	Folder    string `json:"folder"`
	BaseResponse
}
