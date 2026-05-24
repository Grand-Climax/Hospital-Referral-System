package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"Hospital-Referral-System/internal/domain/entity"
)

func TestMapListReferralResponse_UsesDepartmentNameWhenPreloaded(t *testing.T) {
	deptID := uuid.MustParse("b3000000-0000-0000-0000-000000000003")
	resp := MapListReferralResponse(entity.Referral{
		ID:           uuid.New(),
		TargetDeptID: deptID,
		Status:       entity.StatusAccepted,
		TargetDepartment: &entity.Department{
			ID:   deptID,
			Name: "Orthopedics",
		},
	})
	assert.Equal(t, deptID, resp.DepartmentID)
	assert.Equal(t, "Orthopedics", resp.Department)
}

func TestMapListReferralResponse_FallsBackToDeptIDWhenNameMissing(t *testing.T) {
	deptID := uuid.MustParse("b3000000-0000-0000-0000-000000000003")
	resp := MapListReferralResponse(entity.Referral{
		ID:           uuid.New(),
		TargetDeptID: deptID,
		Status:       entity.StatusAccepted,
	})
	assert.Equal(t, deptID, resp.DepartmentID)
	assert.Equal(t, deptID.String(), resp.Department)
}
