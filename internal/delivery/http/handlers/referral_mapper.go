package handlers

import (
	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

func toListReferralResponse(r entity.Referral) dto.ListReferralResponse {
	diag := ""
	icd := ""
	if len(r.Diagnoses) > 0 && r.Diagnoses[0].CodeInfo != nil {
		diag = r.Diagnoses[0].CodeInfo.Description
		icd = r.Diagnoses[0].ICDCode
	}

	patientNameFirst := ""
	patientNameMiddle := ""
	patientNameLast := ""
	patientRegion := ""
	if r.Patient != nil {
		patientNameFirst = r.Patient.FirstNamePlain
		patientNameMiddle = r.Patient.MiddleNamePlain
		patientNameLast = r.Patient.LastNamePlain
		if r.Patient.HomeRegion != nil {
			patientRegion = *r.Patient.HomeRegion
		}
	}

	condition := ""
	if r.ReferralForm != nil {
		condition = r.ReferralForm.ConditionAtReferral
	}

	return dto.ListReferralResponse{
		ID:                  r.ID,
		PatientFirstName:    patientNameFirst,
		PatientMiddleName:   patientNameMiddle,
		PatientLastName:     patientNameLast,
		PatientRegion:       patientRegion,
		Department:          r.TargetDeptID.String(),
		Status:              string(r.Status),
		ICDCode:             icd,
		Diagnosis:           diag,
		ConditionAtReferral: condition,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

func toListReferralResponseSlice(referrals []entity.Referral) []dto.ListReferralResponse {
	responseData := make([]dto.ListReferralResponse, 0, len(referrals))
	for _, r := range referrals {
		responseData = append(responseData, toListReferralResponse(r))
	}
	return responseData
}
