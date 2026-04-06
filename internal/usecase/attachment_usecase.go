package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
)

const AttachmentLimit = 10

type attachmentUseCase struct {
	repo         irepository.AttachmentRepository
	referralRepo irepository.ReferralRepository
	storage      iinfra.StorageService
}

func NewAttachmentUseCase(repo irepository.AttachmentRepository, referralRepo irepository.ReferralRepository, storage iinfra.StorageService) iusecase.AttachmentUseCase {
	return &attachmentUseCase{repo: repo, referralRepo: referralRepo, storage: storage}
}

func (u *attachmentUseCase) SaveAttachment(ctx context.Context, referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) (*entity.Attachment, error) {
	attachment := u.PrepareAttachmentEntity(referralID, fileName, fileType, storagePath, publicID, category, fileSize)

	if err := u.repo.Create(ctx, attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

func (u *attachmentUseCase) PrepareAttachmentEntity(referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) *entity.Attachment {
	if category == "" {
		category = "GENERAL_CLINICAL"
	}

	attachment := &entity.Attachment{
		ReferralID:  referralID,
		FileName:    fileName,
		FileType:    fileType,
		FileSize:    fileSize,
		Category:    category,
		StoragePath: storagePath,
		PublicID:    publicID,
		Metadata:    make(map[string]interface{}),
	}

	// DICOM Metadata Extraction
	if strings.Contains(strings.ToLower(fileType), "dicom") || strings.HasSuffix(strings.ToLower(fileName), ".dcm") {
		u.extractDicomMetadata(storagePath, attachment)
	}

	return attachment
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

func (u *attachmentUseCase) GenerateSignature(hospitalID uuid.UUID) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"folder": "hospitals/" + hospitalID.String() + "/referrals",
	}
	return u.storage.GenerateUploadSignature(params)
}

func (u *attachmentUseCase) extractDicomMetadata(path string, attachment *entity.Attachment) {
	// Simple extraction for now. In a real Cloudinary-only setup, we'd need to stream this or use a Cloudinary function.
	// Since we currently have local storagePath, we can parse it if it exists.
	dataset, err := dicom.ParseFile(path, nil)
	if err != nil {
		return
	}

	if elem, err := dataset.FindElementByTag(tag.Modality); err == nil {
		attachment.Metadata["modality"] = elem.Value.String()
	}
	if elem, err := dataset.FindElementByTag(tag.StudyDate); err == nil {
		attachment.Metadata["study_date"] = elem.Value.String()
	}
	if elem, err := dataset.FindElementByTag(tag.PatientName); err == nil {
		attachment.Metadata["patient_name"] = elem.Value.String()
	}
}

func (u *attachmentUseCase) AddAttachmentsToReferral(ctx context.Context, referralID, doctorID uuid.UUID, reqs []dto.CreateAttachmentRequest) ([]entity.Attachment, error) {
	// 1. Validate Referral
	referral, err := u.referralRepo.GetReferralByID(ctx, referralID)
	if err != nil {
		return nil, err
	}

	// 2. Enforce Status & Ownership
	if referral.Status != entity.StatusDraft && referral.Status != entity.StatusNeedRevision {
		return nil, errors.New("attachments can only be added to referrals in DRAFT or NEED_REVISION status")
	}
	if referral.ReferringDoctorID != doctorID {
		return nil, errors.New("unauthorized: only the referring doctor can manage attachments")
	}

	// 3. Enforce Limit
	existing, err := u.repo.GetByReferralID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if len(existing)+len(reqs) > AttachmentLimit {
		return nil, fmt.Errorf("attachment limit exceeded: maximum allowed is %d", AttachmentLimit)
	}

	// 4. Save Bulk
	var results []entity.Attachment
	for _, req := range reqs {
		att := u.PrepareAttachmentEntity(referralID, req.FileName, req.FileType, req.FileURL, req.PublicID, req.Category, req.FileSize)
		if err := u.repo.Create(ctx, att); err != nil {
			return nil, err
		}
		results = append(results, *att)
	}

	return results, nil
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
