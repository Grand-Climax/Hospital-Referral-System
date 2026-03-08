package auth

import "Hospital-Referral-System/internal/domain/entity"

// RolePermissions defines the hardcoded capability matrix aligning with docs/permission_matrix.md
var RolePermissions = map[entity.UserRole]map[entity.ActionType]bool{
	entity.RoleReferringDoctor: {
		entity.ActionCreateReferral:  true,
		entity.ActionViewPatientData: true,
		entity.ActionLogin:           true,
		entity.ActionLogout:          true,
	},
	entity.RoleLiaisonOfficer: {
		entity.ActionApproveReferral:  true,
		entity.ActionRejectReferral:   true,
		entity.ActionRedirectReferral: true,
		entity.ActionViewPatientData:  true,
		entity.ActionLogin:            true,
		entity.ActionLogout:           true,
	},
	entity.RoleReceivingSpecialist: {
		entity.ActionAcceptReferral:   true,
		entity.ActionRejectReferral:   true,
		entity.ActionRedirectReferral: true,
		entity.ActionOverrideMLScore:  true,
		entity.ActionViewPatientData:  true,
		entity.ActionLogin:            true,
		entity.ActionLogout:           true,
	},
	entity.RoleReceptionist: {
		entity.ActionViewPatientData:     true,
		entity.ActionUpdatePatientStatus: true,
		entity.ActionLogin:               true,
		entity.ActionLogout:              true,
	},
	entity.RoleMohAnalyst: {
		entity.ActionViewPatientData: true,
		entity.ActionExportData:      true,
		entity.ActionGenerateReports: true,
		entity.ActionLogin:           true,
		entity.ActionLogout:          true,
	},
	entity.RoleDeptHead: {
		entity.ActionAcceptReferral:   true,
		entity.ActionRejectReferral:   true,
		entity.ActionRedirectReferral: true,
		entity.ActionOverrideQueue:    true,
		entity.ActionManageCapacity:   true,
		entity.ActionViewPatientData:  true,
		entity.ActionLogin:            true,
		entity.ActionLogout:           true,
	},
	entity.RoleHospitalAdmin: {
		entity.ActionManageUsers:     true,
		entity.ActionAssignRoles:     true,
		entity.ActionManageCapacity:  true,
		entity.ActionViewAuditLog:    true,
		entity.ActionResetMFA:        true,
		entity.ActionViewPatientData: true, // Only for their hospital
		entity.ActionLogin:           true,
		entity.ActionLogout:          true,
	},
	entity.RoleSystemSuperAdmin: {
		entity.ActionManageHospitals: true,
		entity.ActionManageUsers:     true,
		entity.ActionViewAuditLog:    true,
		entity.ActionLogin:           true,
		entity.ActionLogout:          true,
	},
}

// HasPermission checks if a given role is allowed to perform a specific action
func HasPermission(role entity.UserRole, action entity.ActionType) bool {
	if perms, ok := RolePermissions[role]; ok {
		return perms[action]
	}
	return false
}
