package dto

import (
	"Hospital-Referral-System/internal/domain/entity"
)

func referralDepartmentName(r entity.Referral) string {
	if r.TargetDepartment != nil && r.TargetDepartment.Name != "" {
		return r.TargetDepartment.Name
	}
	return r.TargetDeptID.String()
}

// MapListReferralResponse maps a referral entity to the shared list DTO.
// Patient plain-text fields must already be decrypted when Patient is loaded.
func MapListReferralResponse(r entity.Referral) ListReferralResponse {
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
			patientRegion = string(*r.Patient.HomeRegion)
		}
	}

	condition := ""
	if r.ReferralForm != nil {
		condition = r.ReferralForm.ConditionAtReferral
	}

	return ListReferralResponse{
		ID:                  r.ID,
		PatientFirstName:    patientNameFirst,
		PatientMiddleName:   patientNameMiddle,
		PatientLastName:     patientNameLast,
		PatientRegion:       patientRegion,
		DepartmentID:        r.TargetDeptID,
		Department:          referralDepartmentName(r),
		Status:              string(r.Status),
		ICDCode:             icd,
		Diagnosis:           diag,
		ConditionAtReferral: condition,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
		MLStatus:            r.MLStatus,
		MLSeverityScore:     r.MLSeverityScore,
		MLSeverityTier:      r.MLSeverityTier,
	}
}

func MapListReferralResponseSlice(referrals []entity.Referral) []ListReferralResponse {
	out := make([]ListReferralResponse, 0, len(referrals))
	for _, r := range referrals {
		out = append(out, MapListReferralResponse(r))
	}
	return out
}
