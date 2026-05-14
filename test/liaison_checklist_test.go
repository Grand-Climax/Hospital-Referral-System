package test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

func TestUpdateReviewChecklist_StatusRestriction(t *testing.T) {
	rRepo, _, _, _, uc := newTestUC()

	refID := uuid.New()
	liaisonID := uuid.New()
	hospID := uuid.New()

	cases := []struct {
		name    string
		status  entity.ReferralStatus
		wantErr bool
	}{
		{"Allowed - Submitted", entity.StatusSubmitted, false},
		{"Allowed - UnderReview", entity.StatusUnderLiaisonReview, false},
		{"Forbidden - Forwarded", entity.StatusForwarded, true},
		{"Forbidden - Accepted", entity.StatusAccepted, true},
		{"Forbidden - Draft", entity.StatusDraft, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref := &entity.Referral{
				ID:               refID,
				Status:           tc.status,
				SenderHospitalID: hospID,
			}
			rRepo.On("GetReferralByID", mock.Anything, refID).Return(ref, nil).Once()
			if !tc.wantErr {
				rRepo.On("UpdateFields", mock.Anything, refID, mock.Anything).Return(nil).Once()
			}

			trueVal := true
			err := uc.UpdateReviewChecklist(context.Background(), refID, liaisonID, hospID, dto.ReviewChecklistRequest{
				PatientIdentityVerified: &trueVal,
			})

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "forbidden")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLiaisonForward_ChecklistEnforcement(t *testing.T) {
	rRepo, _, _, _, uc := newTestUC()

	liaisonID := uuid.New()
	hospID := uuid.New()
	refID := uuid.New()

	cases := []struct {
		name      string
		condition string
		verified  bool
		history   bool
		vitals    bool
		attach    bool
		wantErr   bool
	}{
		{"Stable - All Verified - Success", "stable", true, true, true, true, false},
		{"Stable - Missing History - Fail", "stable", true, false, true, true, true},
		{"Stable - Missing Identity - Fail", "stable", false, true, true, true, true},
		{"Emergency - Identity Only - Success", "emergency", true, false, false, false, false},
		{"Emergency - Missing Identity - Fail", "emergency", false, false, false, false, true},
		{"Critical - Identity Only - Success", "critical", true, false, false, false, false},
		{"Other - Bypasses Checklist - Success", "unstable", false, false, false, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref := &entity.Referral{
				ID:               refID,
				SenderHospitalID: hospID,
				Status:           entity.StatusUnderLiaisonReview,
				ReferralForm: &entity.ReferralForm{
					ConditionAtReferral: tc.condition,
				},
				PatientIdentityVerified: tc.verified,
				ClinicalHistoryAttached: tc.history,
				VitalsIncluded:          tc.vitals,
				AttachmentsIncluded:     tc.attach,
			}

			rRepo.On("GetReferralByID", mock.Anything, refID).Return(ref, nil).Once()
			if !tc.wantErr {
				rRepo.On("UpdateReferralTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				rRepo.On("CreateStatusHistory", mock.Anything, mock.Anything).Return(nil).Once()
			}

			err := uc.LiaisonForward(context.Background(), refID, liaisonID, hospID, "Forwarding...")

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
