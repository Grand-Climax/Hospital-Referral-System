package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/pkg/utils"
	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
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

func (u *attachmentUseCase) GenerateSignature(hospitalID uuid.UUID) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"folder": "hospitals/" + hospitalID.String() + "/referrals",
	}
	return u.storage.GenerateUploadSignature(params)
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

func (u *attachmentUseCase) AddAttachmentsToReferral(ctx context.Context, referralID, doctorID uuid.UUID, reqs []dto.CreateAttachmentRequest) ([]entity.Attachment, error) {
	if _, err := u.validateManagementAccess(ctx, referralID, doctorID, len(reqs)); err != nil {
		return nil, err
	}

	// Save Bulk
	var results []entity.Attachment
	for _, req := range reqs {
		// --- DE-DUPLICATION CHECK ---
		existing, _ := u.repo.FindByPublicID(ctx, req.PublicID)
		if existing != nil {
			results = append(results, *existing)
			continue
		}

		att := u.PrepareAttachmentEntity(referralID, req.FileName, req.FileType, req.FileURL, req.PublicID, req.Category, req.FileSize)
		
		// If it's a DICOM or PDF file with no metadata yet, try to fetch headers/trailers
		isDicom := strings.Contains(strings.ToLower(req.FileType), "dicom") || strings.HasSuffix(strings.ToLower(req.FileName), ".dcm")
		isPdf := strings.Contains(strings.ToLower(req.FileType), "pdf") || strings.HasSuffix(strings.ToLower(req.FileName), ".pdf")
		
		if isDicom || isPdf {
			resp, err := http.Get(req.FileURL)
			if err == nil {
				defer resp.Body.Close()
				// For DICOM/PDF, we limit the read to avoid huge bandwidth usage
				limitReader := io.LimitReader(resp.Body, 256*1024)
				medicalData, _ := utils.ExtractMetadata(limitReader, req.FileName, req.FileType, req.FileSize)
				for k, v := range medicalData {
					att.Metadata[k] = v
				}
			}
		}

		if err := u.repo.Create(ctx, att); err != nil {
			return nil, err
		}
		results = append(results, *att)
	}

	return results, nil
}

func (u *attachmentUseCase) UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, file interface{}, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error) {
	referral, err := u.validateManagementAccess(ctx, referralID, doctorID, 1)
	if err != nil {
		return nil, err
	}

	// 1. Extract Metadata from stream before upload
	metadata := make(map[string]interface{})
	if seeker, ok := file.(io.ReadSeeker); ok {
		extracted, _ := utils.ExtractMetadata(seeker, fileName, fileType, fileSize)
		if extracted != nil {
			metadata = extracted
		}
		// Reset stream for Cloudinary upload
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	// 2. Upload to Cloudinary
	folder := fmt.Sprintf("hospitals/%s/referrals", referral.SenderHospitalID.String())
	url, publicID, err := u.storage.UploadFile(ctx, file, folder)
	if err != nil {
		return nil, err
	}

	// 3. Create and save
	att := u.PrepareAttachmentEntity(referralID, fileName, fileType, url, publicID, category, fileSize)
	att.Metadata = metadata

	if err := u.repo.Create(ctx, att); err != nil {
		// Cleanup Cloudinary if DB save fails
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

func (u *attachmentUseCase) ProcessWebhookAttachment(ctx context.Context, payload map[string]interface{}) error {
	// 1. Extract context (Referral ID)
	// Cloudinary context is usually in 'context' or 'custom_headers' depending on configuration
	// We expect 'context' with 'referral_id' and 'category'
	contextData, ok := payload["context"].(map[string]interface{})
	if !ok {
		return errors.New("missing context in webhook payload")
	}

	referralIDStr, _ := contextData["custom"].(map[string]interface{})["referral_id"].(string)
	if referralIDStr == "" {
		// This is likely a non-referral upload (e.g. Profile Picture)
		// We return nil to tell Cloudinary we received the notification, but we're not interested in it.
		return nil
	}

	referralID, err := uuid.Parse(referralIDStr)
	if err != nil {
		// Log error but return nil to stop Cloudinary retries for invalid context
		return nil
	}

	// 2. Extract basic file info
	publicID, _ := payload["public_id"].(string)
	fileURL, _ := payload["secure_url"].(string)
	fileType, _ := payload["format"].(string)
	fileSize := int64(payload["bytes"].(float64))
	fileName, _ := payload["original_filename"].(string)
	category, _ := contextData["category"].(string)

	if fileName == "" {
		fileName = publicID // Fallback
	}

	// --- DE-DUPLICATION CHECK ---
	existing, _ := u.repo.FindByPublicID(ctx, publicID)
	if existing != nil {
		// Log that we've already handled this file
		return nil 
	}

	// 3. Create Attachment
	att := u.PrepareAttachmentEntity(referralID, fileName, fileType, fileURL, publicID, category, fileSize)
	
	// 4. Handle Metadata
	att.Metadata["cloudinary_raw"] = payload
	
	resourceType, _ := payload["resource_type"].(string)
	
	// Mode A: Standard Image Dimensions from Payload
	if resourceType == "image" || strings.HasPrefix(fileType, "image/") {
		if w, ok := payload["width"].(float64); ok {
			att.Metadata["width"] = int(w)
		}
		if h, ok := payload["height"].(float64); ok {
			att.Metadata["height"] = int(h)
		}
	}

	// Mode B: Partial Download for DICOM/PDF Medical Metadata
	isDicom := strings.Contains(strings.ToLower(fileType), "dicom") || strings.HasSuffix(strings.ToLower(fileName), ".dcm")
	isPdf := strings.Contains(strings.ToLower(fileType), "pdf") || strings.HasSuffix(strings.ToLower(fileName), ".pdf")

	if isDicom || isPdf {
		// Fetch only the first 256KB to extract dataset/trailer info
		resp, err := http.Get(fileURL)
		if err == nil {
			defer resp.Body.Close()
			limitReader := io.LimitReader(resp.Body, 256*1024)
			medicalData, _ := utils.ExtractMetadata(limitReader, fileName, fileType, fileSize)
			for k, v := range medicalData {
				att.Metadata[k] = v
			}
		}
	}

	return u.repo.Create(ctx, att)
}
