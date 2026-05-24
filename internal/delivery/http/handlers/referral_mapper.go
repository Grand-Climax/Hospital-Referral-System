package handlers

import (
	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

func toListReferralResponse(r entity.Referral) dto.ListReferralResponse {
	return dto.MapListReferralResponse(r)
}

func toListReferralResponseSlice(referrals []entity.Referral) []dto.ListReferralResponse {
	return dto.MapListReferralResponseSlice(referrals)
}

func toRedirectionResponse(r entity.ReferralRedirection) dto.RedirectionResponse {
	reason := ""
	if r.RedirectionReason != nil {
		reason = *r.RedirectionReason
	}
	fromName := ""
	if r.RedirectedFromHospital != nil {
		fromName = r.RedirectedFromHospital.Name
	}
	toName := ""
	if r.RedirectedToHospital != nil {
		toName = r.RedirectedToHospital.Name
	}

	return dto.RedirectionResponse{
		ID:                         r.ID,
		ReferralID:                 r.ReferralID,
		RedirectedFromHospitalID:   r.RedirectedFromHospitalID,
		RedirectedFromHospitalName: fromName,
		RedirectedToHospitalID:     r.RedirectedToHospitalID,
		RedirectedToHospitalName:   toName,
		RedirectedBySpecialistID:   r.RedirectedBySpecialistID,
		RedirectionReason:          reason,
		CreatedAt:                  r.CreatedAt,
	}
}

func toRedirectionResponseSlice(redirs []entity.ReferralRedirection) []dto.RedirectionResponse {
	resp := make([]dto.RedirectionResponse, 0, len(redirs))
	for _, r := range redirs {
		resp = append(resp, toRedirectionResponse(r))
	}
	return resp
}

