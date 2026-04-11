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
	ID                 string                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReferralID         string                 `json:"referral_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FileName           string                 `json:"file_name" example:"xray_chest.dcm"`
	FileType           string                 `json:"file_type" example:"application/dicom"`
	FileSize           int64                  `json:"file_size" example:"4587210"`
	Category           string                 `json:"category" example:"RADIOLOGY"`
	StoragePath        string                 `json:"file_url" example:"https://res.cloudinary.com/..."`
	VerificationStatus string                 `json:"verification_status" example:"VERIFIED"`
	RejectionReason    string                 `json:"rejection_message,omitempty" example:"Metadata extraction failed"`
	RejectedAt         *time.Time             `json:"rejected_at,omitempty" example:"2026-04-11T19:55:00Z"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	UploadedAt         time.Time              `json:"uploaded_at"`
	BaseResponse
}

type AttachmentListResponse struct {
	Data []AttachmentResponse `json:"data"`
	BaseResponse
}

type UploadSignatureResponse struct {
	BaseResponse
	ReferralID string `json:"referral_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Signature  string `json:"signature" example:"a92425642...`
	Timestamp  int64  `json:"timestamp" example:"1649684700"`
	APIKey     string `json:"api_key" example:"123456789"`
	CloudName  string `json:"cloud_name" example:"hospital-system"`
	Folder     string `json:"folder" example:"temp/550e8400-e29b-41d4-a716-446655440000"`
}
