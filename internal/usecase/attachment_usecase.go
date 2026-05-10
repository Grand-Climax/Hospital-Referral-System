package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/pkg/utils"
)

const AttachmentLimit = 10

type attachmentUseCase struct {
	repo         irepository.AttachmentRepository
	referralRepo irepository.ReferralRepository
	storage      iinfra.StorageService
	inAppNotifUC iusecase.InAppNotificationUseCase
}

func NewAttachmentUseCase(repo irepository.AttachmentRepository, referralRepo irepository.ReferralRepository, storage iinfra.StorageService, inAppNotifUC iusecase.InAppNotificationUseCase) iusecase.AttachmentUseCase {
	return &attachmentUseCase{repo: repo, referralRepo: referralRepo, storage: storage, inAppNotifUC: inAppNotifUC}
}


func (u *attachmentUseCase) PrepareAttachmentEntity(referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) *entity.Attachment {
	if category == "" {
		category = "GENERAL_CLINICAL"
	}

	return &entity.Attachment{
		ReferralID:  referralID,
		FileName:    fileName,
		FileType:    fileType,
		FileSize:    fileSize,
		Category:    category,
		StoragePath: storagePath,
		PublicID:    publicID,
		Metadata:    make(map[string]interface{}),
	}
}

func (u *attachmentUseCase) GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error) {
	return u.repo.FindByID(ctx, id)
}

func (u *attachmentUseCase) GetAttachmentsByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error) {
	return u.repo.GetByReferralID(ctx, referralID)
}

func (u *attachmentUseCase) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	attachment, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from Cloudinary if PublicID exists
	if attachment.PublicID != "" {
		_ = u.storage.DeleteFile(ctx, attachment.PublicID) // Log error but proceed with DB delete
	}

	return u.repo.HardDelete(ctx, id)
}


func (u *attachmentUseCase) validateManagementAccess(ctx context.Context, referralID, doctorID uuid.UUID, newCount int) (*entity.Referral, error) {
	// 1. Validate Referral
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return nil, err
	}

	// 2. Enforce Status & Ownership
	if referral.Status != entity.StatusDraft && referral.Status != entity.StatusNeedRevision {
		return nil, errors.New("attachments can only be managed for referrals in DRAFT or NEED_REVISION status")
	}
	if referral.ReferringDoctorID != doctorID {
		return nil, errors.New("unauthorized: only the referring doctor can manage attachments")
	}

	// 3. Enforce Limit
	existing, err := u.repo.GetByReferralID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if len(existing)+newCount > AttachmentLimit {
		return nil, fmt.Errorf("attachment limit exceeded: maximum allowed is %d", AttachmentLimit)
	}

	return referral, nil
}


func (u *attachmentUseCase) VerifyAttachmentBytes(att *entity.Attachment, data []byte) (status string, metadata map[string]interface{}, reason string) {
	return u.verifyAttachmentBytes(att, data)
}

func (u *attachmentUseCase) Storage() iinfra.StorageService {
	return u.storage
}


func (u *attachmentUseCase) verifyAttachmentBytes(att *entity.Attachment, fileBytes []byte) (status string, metadata map[string]interface{}, rejectionReason string) {
	// Check if file is DICOM, PDF or Image
	isDicom := strings.Contains(strings.ToLower(att.FileType), "dicom") || strings.HasSuffix(strings.ToLower(att.FileName), ".dcm")
	isPdf := strings.Contains(strings.ToLower(att.FileType), "pdf") || strings.HasSuffix(strings.ToLower(att.FileName), ".pdf")
	isImage := strings.Contains(strings.ToLower(att.FileType), "image/") || 
		strings.HasSuffix(strings.ToLower(att.FileName), ".jpg") ||
		strings.HasSuffix(strings.ToLower(att.FileName), ".jpeg")

	metadata = make(map[string]interface{})
	status = entity.VerificationVerified

	if isDicom || isPdf || isImage {
		reader := bytes.NewReader(fileBytes)
		limitReader := io.LimitReader(reader, 256*1024)
		medicalData, err := utils.ExtractMetadata(limitReader, att.FileName, att.FileType, att.FileSize)
		if err != nil {
			status = entity.VerificationRejected
			rejectionReason = fmt.Sprintf("Metadata extraction failed: %v", err)
		} else {
			for k, v := range medicalData {
				metadata[k] = v
			}
		}
	}
	return
}

func (u *attachmentUseCase) UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, fileBytes []byte, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error) {
	_, err := u.validateManagementAccess(ctx, referralID, doctorID, 1)
	if err != nil {
		return nil, err
	}

	// 1. Upload to permanent folder
	folder := fmt.Sprintf("referrals/%s/attachments", referralID.String())
	reader := bytes.NewReader(fileBytes)
	url, publicID, err := u.storage.UploadFile(ctx, reader, folder)
	if err != nil {
		return nil, fmt.Errorf("upload to storage failed: %w", err)
	}

	// 2. Prepare attachment entity
	att := u.PrepareAttachmentEntity(referralID, fileName, fileType, url, publicID, category, fileSize)

	// 3. Immediate verification
	status, metadata, rejectionReason := u.verifyAttachmentBytes(att, fileBytes)
	att.VerificationStatus = status
	att.Metadata = metadata
	att.RejectionReason = rejectionReason

	// 4. If rejected and status is SUBMITTED, move referral to NEED_REVISION
	if status == entity.VerificationRejected {
		ref, err := u.referralRepo.GetReferralByID(ctx, referralID)
		if err != nil {
			return nil, err
		}

		// Only move to NEED_REVISION if it's currently SUBMITTED.
		// If it's DRAFT, we leave it as DRAFT.
		if ref.Status == entity.StatusSubmitted {
			ref.Status = entity.StatusNeedRevision
			revisionMessage := fmt.Sprintf("Attachment '%s' was rejected: %s", fileName, rejectionReason)
			ref.RevisionReason = &revisionMessage
			if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
				return nil, fmt.Errorf("failed to update referral on attachment rejection: %w", err)
			}
			// Send notification for rejection
			_ = u.inAppNotifUC.CreateForEvent(ctx, "REFERRAL_NEEDS_REVISION", referralID, doctorID)
		}

		now := time.Now()
		att.RejectedAt = &now
	}

	// 5. Save attachment
	if err := u.repo.Create(ctx, att); err != nil {
		_ = u.storage.DeleteFile(ctx, publicID)
		return nil, err
	}

	return att, nil
}


func (u *attachmentUseCase) DeleteAttachmentFromReferral(ctx context.Context, referralID, attachmentID, doctorID uuid.UUID) error {
	// 1. Validate Referral
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return err
	}

	// 2. Enforce Status & Ownership
	if referral.Status != entity.StatusDraft && referral.Status != entity.StatusNeedRevision {
		return errors.New("attachments can only be deleted from referrals in DRAFT or NEED_REVISION status")
	}
	if referral.ReferringDoctorID != doctorID {
		return errors.New("unauthorized: only the referring doctor can manage attachments")
	}

	// 3. Fetch Attachment to verify it belongs to this referral
	att, err := u.repo.FindByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if att.ReferralID != referralID {
		return errors.New("attachment does not belong to the specified referral")
	}

	// 4. Trigger logic-rich DeleteAttachment (handles Cloudinary)
	return u.DeleteAttachment(ctx, attachmentID)
}

