package dto

import "Hospital-Referral-System/internal/domain/entity"

type AttachmentResponse struct {
	*entity.Attachment
	BaseResponse
}
