package test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/usecase"
)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCancelReferral_Ownership_Success(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	doctorID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:                refID,
		ReferringDoctorID: doctorID,
		Status:            entity.StatusDraft,
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)
	rRepo.On("UpdateReferralTransaction", mock.Anything, mock.Anything).Return(nil)
	rRepo.On("CreateStatusHistory", mock.Anything, mock.Anything).Return(nil)

	err := uc.CancelReferral(context.Background(), refID, doctorID, "No longer needed")
	assert.NoError(t, err)
	assert.Equal(t, entity.StatusCancelled, existing.Status)
}

func TestCancelReferral_Forbidden_NotOwner(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	doctorID := uuid.New()
	otherDoctorID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:                refID,
		ReferringDoctorID: otherDoctorID,
		Status:            entity.StatusDraft,
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)

	err := uc.CancelReferral(context.Background(), refID, doctorID, "Stealing cancellation")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestCancelReferral_Forbidden_ActivePipeline(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	doctorID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:                refID,
		ReferringDoctorID: doctorID,
		Status:            entity.StatusUnderLiaisonReview,
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)

	err := uc.CancelReferral(context.Background(), refID, doctorID, "Trying to cancel while under review")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestDeleteAttachments_Ownership_Success(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	doctorID := uuid.New()
	refID := uuid.New()
	attID := uuid.New()
	existing := &entity.Referral{
		ID:                refID,
		ReferringDoctorID: doctorID,
		Status:            entity.StatusDraft,
		Attachments: []entity.Attachment{
			{ID: attID},
		},
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)
	aUC.On("DeleteAttachment", mock.Anything, attID).Return(nil)

	err := uc.DeleteAttachmentsByReferralID(context.Background(), refID, doctorID)
	assert.NoError(t, err)
	aUC.AssertCalled(t, "DeleteAttachment", mock.Anything, attID)
}

func TestDeleteAttachments_Forbidden_WrongStatus(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	doctorID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:                refID,
		ReferringDoctorID: doctorID,
		Status:            entity.StatusSubmitted, // Already submitted
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)

	err := uc.DeleteAttachmentsByReferralID(context.Background(), refID, doctorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestSpecialistAccept_Ownership_DeniedOtherSpecialist(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	hospID := uuid.New()
	specA := uuid.New()
	specB := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		TargetHospitalID: hospID,
		Status:           entity.StatusUnderSpecialistReview,
		SpecialistID:     &specA,
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)

	err := uc.SpecialistAccept(context.Background(), refID, specB, hospID, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claimed by another specialist")
}

func TestLiaisonForward_BlockedByPending(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	liaisonID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		LiaisonOfficerID: &liaisonID,
		SenderHospitalID: hospID,
		Status:           entity.StatusUnderLiaisonReview,
		Attachments: []entity.Attachment{
			{ID: uuid.New(), VerificationStatus: entity.VerificationPending},
		},
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)

	err := uc.LiaisonForward(context.Background(), refID, liaisonID, hospID, "Forwarding...")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "need to be confirmed")
}

func TestLiaisonForward_BlockedByRejected(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	liaisonID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		LiaisonOfficerID: &liaisonID,
		SenderHospitalID: hospID,
		Status:           entity.StatusUnderLiaisonReview,
		Attachments: []entity.Attachment{
			{ID: uuid.New(), VerificationStatus: entity.VerificationRejected},
		},
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)

	err := uc.LiaisonForward(context.Background(), refID, liaisonID, hospID, "Forwarding...")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires revision")
}

func TestLiaisonForward_Success_Verified(t *testing.T) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, aUC, notifUC, nil)

	liaisonID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		LiaisonOfficerID: &liaisonID,
		SenderHospitalID: hospID,
		Status:           entity.StatusUnderLiaisonReview,
		Attachments: []entity.Attachment{
			{ID: uuid.New(), VerificationStatus: entity.VerificationVerified},
		},
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)
	rRepo.On("UpdateReferralTransaction", mock.Anything, mock.Anything).Return(nil)
	rRepo.On("CreateStatusHistory", mock.Anything, mock.Anything).Return(nil)

	err := uc.LiaisonForward(context.Background(), refID, liaisonID, hospID, "Forwarding...")
	assert.NoError(t, err)
}
