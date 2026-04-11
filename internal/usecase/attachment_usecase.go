package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/pkg/utils"
	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"golang.org/x/sync/errgroup"
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

func (u *attachmentUseCase) GenerateSignature(referralID *uuid.UUID) (map[string]interface{}, uuid.UUID, error) {
	targetID := uuid.New()
	if referralID != nil && *referralID != uuid.Nil {
		targetID = *referralID
	}

	params := map[string]interface{}{
		"folder": "temp/" + targetID.String(),
	}
	sig, err := u.storage.GenerateUploadSignature(params)
	return sig, targetID, err
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
		
		// Set verification status to PENDING initially. Metadata is extracted later by a Cron Job.
		att.VerificationStatus = entity.VerificationPending

		if err := u.repo.Create(ctx, att); err != nil {
			return nil, err
		}
		results = append(results, *att)
	}

	return results, nil
}

func (u *attachmentUseCase) UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, file interface{}, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error) {
	_, err := u.validateManagementAccess(ctx, referralID, doctorID, 1)
	if err != nil {
		return nil, err
	}

	// 1. Upload to Cloudinary's temp folder immediately
	folder := fmt.Sprintf("temp/%s", referralID.String())
	url, publicID, err := u.storage.UploadFile(ctx, file, folder)
	if err != nil {
		return nil, err
	}

	// 2. Create and save with PENDING status
	att := u.PrepareAttachmentEntity(referralID, fileName, fileType, url, publicID, category, fileSize)
	att.VerificationStatus = entity.VerificationPending

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

func (u *attachmentUseCase) VerifyPendingAttachments(ctx context.Context) error {
	// Robustness: Process in manageable batches to avoid cron timeouts
	const BatchSize = 20
	pending, err := u.repo.GetPendingAttachmentsBatch(ctx, BatchSize)
	if err != nil {
		return err
	}

	// Group by Referral for better batching and potential collective promotion
	groups := make(map[uuid.UUID][]entity.Attachment)
	for _, att := range pending {
		groups[att.ReferralID] = append(groups[att.ReferralID], att)
	}

	for _, attachments := range groups {
		// Parallelize within a single referral for faster processing
		g, gCtx := errgroup.WithContext(ctx)
		for _, att := range attachments {
			a := att // capture loop var
			g.Go(func() error {
				_, err := u.VerifyAttachment(gCtx, a.ID)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			// Log and continue to next referral to ensure one failure doesn't block the whole job
			continue 
		}
	}

	return nil
}

func (u *attachmentUseCase) VerifyAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error) {
	att, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if att.VerificationStatus != entity.VerificationPending {
		return att, nil // Already processed
	}

	// 1. Get referral to ensure we can build the permanent hospital path
	ref, err := u.referralRepo.GetReferralByID(ctx, att.ReferralID)
	if err != nil {
		return nil, fmt.Errorf("referral not found: %w", err)
	}

	isDicom := strings.Contains(strings.ToLower(att.FileType), "dicom") || strings.HasSuffix(strings.ToLower(att.FileName), ".dcm")
	isPdf := strings.Contains(strings.ToLower(att.FileType), "pdf") || strings.HasSuffix(strings.ToLower(att.FileName), ".pdf")

	metadata := make(map[string]interface{})
	status := entity.VerificationVerified
	var rejectionReason string

	// 2. Verification constraint for DICOM/PDF headers
	if isDicom || isPdf {
		resp, err := http.Get(att.StoragePath)
		if err != nil || resp.StatusCode != http.StatusOK {
			status = entity.VerificationRejected
			rejectionReason = "Failed to fetch file from storage"
			if err != nil {
				rejectionReason = fmt.Sprintf("Failed to fetch file: %v", err)
			}
			if resp != nil {
				resp.Body.Close()
			}
		} else {
			defer resp.Body.Close()
			limitReader := io.LimitReader(resp.Body, 256*1024)
			medicalData, err := utils.ExtractMetadata(limitReader, att.FileName, att.FileType, att.FileSize)

			// Basic validation constraint: if it claims to be DICOM but extraction fails
			if err != nil {
				status = entity.VerificationRejected
				rejectionReason = fmt.Sprintf("Metadata extraction failed: %v", err)
			} else {
				for k, v := range medicalData {
					metadata[k] = v
				}
			}
		}
	}

	// 3. Promote (Rename) if Verified
	var newStoragePath string
	var newPublicID string
	if status == entity.VerificationVerified {
		// New path: hospitals/<sender_hosp>/referrals/<referral_id>/filename
		newPublicIDRaw := fmt.Sprintf("hospitals/%s/referrals/%s/%s", ref.SenderHospitalID.String(), ref.ID.String(), att.FileName)
		
		newUrl, newId, renameErr := u.storage.RenameFile(ctx, att.PublicID, newPublicIDRaw)
		if renameErr != nil {
			return nil, fmt.Errorf("failed to promote file in Cloudinary: %w", renameErr)
		}
		newStoragePath = newUrl
		newPublicID = newId
	}

	// 4. Update DB
	var rejectedAt *time.Time
	if status == entity.VerificationRejected {
		now := time.Now()
		rejectedAt = &now

		// Update parent referral status to NEED_REVISION
		revisionMessage := fmt.Sprintf("Attachment '%s' was rejected: %s", att.FileName, rejectionReason)
		ref.Status = entity.StatusNeedRevision
		ref.RevisionReason = &revisionMessage
		
		if err := u.referralRepo.UpdateReferralTransaction(ctx, ref); err != nil {
			return nil, fmt.Errorf("failed to update referral status on attachment rejection: %w", err)
		}
	}

	if err := u.repo.UpdateVerificationStatus(ctx, att.ID, status, metadata, newStoragePath, newPublicID, rejectionReason, rejectedAt); err != nil {
		return nil, err
	}

	// Return updated entity
	return u.repo.FindByID(ctx, id)
}

func (u *attachmentUseCase) VerifyReferralAttachments(ctx context.Context, referralID uuid.UUID) error {
	attachments, err := u.repo.GetByReferralID(ctx, referralID)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)
	for _, att := range attachments {
		if att.VerificationStatus != entity.VerificationPending {
			continue
		}
		a := att
		g.Go(func() error {
			_, err := u.VerifyAttachment(gCtx, a.ID)
			return err
		})
	}
	return g.Wait()
}

func (u *attachmentUseCase) CleanupTempAttachments(ctx context.Context) error {
	// Folder-Centric Cleanup logic
	folders, err := u.storage.ListFolders(ctx, "temp")
	if err != nil {
		return fmt.Errorf("failed to list temp folders: %w", err)
	}

	for _, folderPath := range folders {
		// folderPath is "temp/<uuid>"
		parts := strings.Split(folderPath, "/")
		if len(parts) < 2 {
			continue
		}
		folderID := parts[1]

		// 1. Safety Check: Is it referenced in DB?
		// We check for both clinical attachment records and existence of referral ID
		count, err := u.repo.CountByPublicIDPrefix(ctx, folderPath)
		if err != nil {
			continue
		}
		
		if count > 0 {
			// Active attachments exist, do not clear
			continue
		}

		// 2. Existence Check: Does the folder ID match a known referral?
		// (Case where signature was generated but create-referral hasn't finished yet)
		refUUID, pErr := uuid.Parse(folderID)
		if pErr == nil {
			_, rErr := u.referralRepo.GetReferralByID(ctx, refUUID)
			if rErr == nil {
				// Referral exists, keep for now
				continue
			}
		}

		// 3. Purge Orphan
		// Piece-by-piece: Handles one orphan folder cleanup per pass if we wanted, 
		// but here we loop through all for simplicity.
		_ = u.storage.DeleteFilesByPrefix(ctx, folderPath+"/")
	}

	return nil
}
