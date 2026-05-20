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

func newTestUC() (*MockReferralRepo, *MockNetworkRepo, *MockReferralRedirectionRepo, *MockDepartmentRepo, usecase.ReferralUseCaseFacade) {
	rRepo := new(MockReferralRepo)
	cRepo := new(MockClinicalUpdateRepo)
	oRepo := new(MockReferralOutcomeRepo)
	nRepo := new(MockNetworkRepo)
	redirRepo := new(MockReferralRedirectionRepo)
	deptRepo := new(MockDepartmentRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	inAppNotifUC := new(MockInAppNotificationUseCase)
	triageRepo := new(MockTriageQueueRepo)
	attRepo := new(MockAttachmentRepo)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, redirRepo, triageRepo, aUC, notifUC, inAppNotifUC, nil, deptRepo, attRepo, new(MockReferralAccessRepo), nil, nil)
	
	// Default expectations for inAppNotifUC to avoid panics on unexpected calls
	inAppNotifUC.On("CreateForEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	
	return rRepo, nRepo, redirRepo, deptRepo, uc
}

func TestCancelReferral_Ownership_Success(t *testing.T) {
	rRepo, _, _, _, uc := newTestUC()

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
	rRepo, _, _, _, uc := newTestUC()

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
	rRepo, _, _, _, uc := newTestUC()

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
	redirRepo := new(MockReferralRedirectionRepo)
	deptRepo := new(MockDepartmentRepo)
	aUC := new(MockAttachmentUseCase)
	notifUC := new(MockNotificationUseCase)
	inAppNotifUC := new(MockInAppNotificationUseCase)
	triageRepo := new(MockTriageQueueRepo)
	attRepo := new(MockAttachmentRepo)
	uc := usecase.NewReferralUseCase(rRepo, cRepo, oRepo, nRepo, redirRepo, triageRepo, aUC, notifUC, inAppNotifUC, nil, deptRepo, attRepo, new(MockReferralAccessRepo), nil, nil)
	inAppNotifUC.On("CreateForEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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
	rRepo, _, _, _, uc := newTestUC()

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
	rRepo, _, _, _, uc := newTestUC()

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
	rRepo, _, _, _, uc := newTestUC()

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
	rRepo, _, _, _, uc := newTestUC()

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
	rRepo, _, _, _, uc := newTestUC()

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

func TestSpecialistRead_Redirected_Success(t *testing.T) {
	rRepo, _, _, _, uc := newTestUC()

	specID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		TargetHospitalID: hospID,
		Status:           entity.StatusRedirected,
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)
	rRepo.On("UpdateReferralTransaction", mock.Anything, mock.Anything).Return(nil)
	rRepo.On("CreateStatusHistory", mock.Anything, mock.Anything).Return(nil)

	err := uc.SpecialistRead(context.Background(), refID, specID, hospID)
	assert.NoError(t, err)
	assert.Equal(t, entity.StatusUnderSpecialistReview, existing.Status)
	assert.Equal(t, &specID, existing.SpecialistID)
}

func TestSpecialistRelease_RevertsToRedirected(t *testing.T) {
	rRepo, _, redirRepo, _, uc := newTestUC()

	specID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		TargetHospitalID: hospID,
		Status:           entity.StatusUnderSpecialistReview,
		SpecialistID:     &specID,
	}

	redirections := []entity.ReferralRedirection{
		{ReferralID: refID},
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)
	redirRepo.On("ListByReferralID", mock.Anything, refID).Return(redirections, nil)
	rRepo.On("UpdateReferralTransaction", mock.Anything, mock.Anything).Return(nil)
	rRepo.On("CreateStatusHistory", mock.Anything, mock.Anything).Return(nil)

	err := uc.SpecialistRelease(context.Background(), refID, specID, hospID, "Releasing...")
	assert.NoError(t, err)
	assert.Equal(t, entity.StatusRedirected, existing.Status)
	assert.Nil(t, existing.SpecialistID)
}

func TestLiaisonUnassignSpecialist_RevertsToRedirected(t *testing.T) {
	rRepo, _, redirRepo, _, uc := newTestUC()

	liaisonID := uuid.New()
	specID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()
	existing := &entity.Referral{
		ID:               refID,
		TargetHospitalID: hospID,
		Status:           entity.StatusUnderSpecialistReview,
		SpecialistID:     &specID,
	}

	redirections := []entity.ReferralRedirection{
		{ReferralID: refID},
	}

	rRepo.On("GetReferralByID", mock.Anything, refID).Return(existing, nil)
	redirRepo.On("ListByReferralID", mock.Anything, refID).Return(redirections, nil)
	rRepo.On("UpdateReferralTransaction", mock.Anything, mock.Anything).Return(nil)
	rRepo.On("CreateStatusHistory", mock.Anything, mock.Anything).Return(nil)

	err := uc.LiaisonUnassignSpecialist(context.Background(), refID, liaisonID, hospID, "Unassigning...")
	assert.NoError(t, err)
	assert.Equal(t, entity.StatusRedirected, existing.Status)
	assert.Nil(t, existing.SpecialistID)
}

