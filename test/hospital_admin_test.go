package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
)

func setupHospitalAdminTestRouter() (*gin.Engine, *MockUserUseCase, *MockReferralUseCase, *MockHospitalUseCase, *MockDepartmentUseCase) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUserUC := new(MockUserUseCase)
	mockReferralUC := new(MockReferralUseCase)
	mockHospitalUC := new(MockHospitalUseCase)
	mockDeptUC := new(MockDepartmentUseCase)

	staffHandler := handlers.NewHospitalAdminStaffHandler(mockUserUC, mockReferralUC, mockDeptUC)
	opsHandler := handlers.NewHospitalAdminOperationsHandler(mockUserUC, mockHospitalUC, mockDeptUC)

	// Mock Hospital Admin Context
	authMiddleware := func(c *gin.Context) {
		adminID := uuid.New()
		hospID := uuid.New()
		c.Set("userID", adminID)
		c.Set("role", entity.RoleHospitalAdmin)
		c.Set("hospID", &hospID)
		c.Next()
	}

	api := r.Group("/api/v1/hospital-admin")
	api.Use(authMiddleware)
	{
		// Staff Management
		api.POST("/staff", staffHandler.CreateStaff)
		api.GET("/staff", staffHandler.ListStaff)
		api.GET("/staff/:id", staffHandler.GetStaff)
		api.PATCH("/staff/:id/role", staffHandler.ChangeStaffRole)
		api.PATCH("/staff/:id/department", staffHandler.ReassignDepartment)
		api.DELETE("/staff/:id", staffHandler.DeleteStaff)
		api.POST("/staff/:id/force-logout", staffHandler.ForceLogoutStaff)

		// Operations
		api.GET("/hospital/profile", opsHandler.GetMyHospitalProfile)
		api.PATCH("/hospital/profile", opsHandler.UpdateMyHospitalProfile)
		api.POST("/departments", opsHandler.LinkDepartmentToMyHospital)
		api.GET("/departments", opsHandler.ListMyHospitalDepartments)
		api.PATCH("/departments/:deptId/head", opsHandler.AssignDepartmentHead)
	}

	return r, mockUserUC, mockReferralUC, mockHospitalUC, mockDeptUC
}

func TestHospitalAdminStaffManagement(t *testing.T) {
	staffID := uuid.New()

	t.Run("Create Staff", func(t *testing.T) {
		r, mockUser, _, _, _ := setupHospitalAdminTestRouter()
		reqBody := dto.HospitalAdminCreateStaffRequest{
			FirstName:  "John",
			MiddleName: "Middle",
			LastName:   "Doe",
			Email:      "john.doe@hospital.com",
			Password:   "securePass123",
			Role:       entity.RoleLiaisonOfficer,
			NationalID: "NAT123456",
		}
		mockUser.On("HospitalAdminCreateStaff", mock.Anything, mock.Anything, mock.Anything, "securePass123").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/hospital-admin/staff", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("List Staff", func(t *testing.T) {
		r, mockUser, _, _, _ := setupHospitalAdminTestRouter()
		mockUser.On("HospitalAdminListStaff", mock.Anything, mock.Anything, mock.Anything).Return([]entity.User{}, int64(0), nil)

		req, _ := http.NewRequest("GET", "/api/v1/hospital-admin/staff", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("Force Logout Staff", func(t *testing.T) {
		r, mockUser, _, _, _ := setupHospitalAdminTestRouter()
		mockUser.On("HospitalAdminForceLogoutStaff", mock.Anything, mock.Anything, staffID).Return(int64(1), nil)

		req, _ := http.NewRequest("POST", "/api/v1/hospital-admin/staff/"+staffID.String()+"/force-logout", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})
}

func TestHospitalAdminOperations(t *testing.T) {
	deptID := uuid.New()

	t.Run("Get Hospital Profile", func(t *testing.T) {
		r, _, _, mockHosp, _ := setupHospitalAdminTestRouter()
		mockHosp.On("GetHospitalByID", mock.Anything, mock.Anything).Return(&entity.Hospital{Name: "General Hospital"}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/hospital-admin/hospital/profile", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockHosp.AssertExpectations(t)
	})

	t.Run("Link Department", func(t *testing.T) {
		r, _, _, _, mockDept := setupHospitalAdminTestRouter()
		reqBody := dto.HospitalAdminLinkDepartmentRequest{
			DepartmentID: deptID.String(),
			DailyLimit:   50,
		}
		mockDept.On("LinkDepartmentToHospital", mock.Anything, mock.Anything, deptID, 50).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/hospital-admin/departments", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockDept.AssertExpectations(t)
	})

	t.Run("Assign Department Head", func(t *testing.T) {
		r, mockUser, _, _, mockDept := setupHospitalAdminTestRouter()
		staffID := uuid.New()
		reqBody := dto.HospitalAdminAssignDepartmentHeadRequest{
			StaffID: staffID.String(),
		}
		
		// AssignDepartmentHead calls ListHospitalDepartments first for validation
		mockDept.On("ListHospitalDepartments", mock.Anything, mock.Anything).Return([]entity.HospitalDepartment{{DepartmentID: deptID}}, nil)
		mockUser.On("HospitalAdminReassignStaffDepartment", mock.Anything, mock.Anything, staffID, &deptID).Return(nil)
		mockUser.On("HospitalAdminChangeStaffRole", mock.Anything, mock.Anything, staffID, entity.RoleDeptHead).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/api/v1/hospital-admin/departments/"+deptID.String()+"/head", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockDept.AssertExpectations(t)
		mockUser.AssertExpectations(t)
	})
}

func TestHospitalAdminStaffManagement_ValidationFailures(t *testing.T) {
	staffID := uuid.New()
	deptID := uuid.New()

	t.Run("Create staff without department for Referring Doctor", func(t *testing.T) {
		r, _, _, _, _ := setupHospitalAdminTestRouter()
		reqBody := dto.HospitalAdminCreateStaffRequest{
			FirstName:  "John",
			MiddleName: "Middle",
			LastName:   "Doe",
			Email:      "john.doe2@hospital.com",
			Password:   "securePass123",
			Role:       entity.RoleReferringDoctor,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/hospital-admin/staff", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "requires a department")
	})

	t.Run("Change role to Liaison Officer fails if department exists", func(t *testing.T) {
		r, mockUser, _, _, _ := setupHospitalAdminTestRouter()

		existingStaff := &entity.User{
			ID:           staffID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &deptID, // mock hospital id
			DepartmentID: &deptID,
		}
		mockUser.On("HospitalAdminGetStaffByID", mock.Anything, mock.Anything, staffID).Return(existingStaff, nil)

		reqBody := dto.HospitalAdminChangeRoleRequest{
			Role: entity.RoleLiaisonOfficer,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/api/v1/hospital-admin/staff/"+staffID.String()+"/role", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "does not accept a department")
	})

	t.Run("Reassign department fails if role forbids it", func(t *testing.T) {
		r, mockUser, _, _, mockDept := setupHospitalAdminTestRouter()

		existingStaff := &entity.User{
			ID:           staffID,
			Role:         entity.RoleLiaisonOfficer,
			HospitalID:   &deptID,
			DepartmentID: nil,
		}
		mockUser.On("HospitalAdminGetStaffByID", mock.Anything, mock.Anything, staffID).Return(existingStaff, nil)
		mockDept.On("ValidateDepartmentForHospital", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		reqBody := dto.HospitalAdminReassignDepartmentRequest{
			DepartmentID: pointerToString(deptID.String()),
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/api/v1/hospital-admin/staff/"+staffID.String()+"/department", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "does not accept a department")
	})
}

func TestHospitalAdmin_CreateStaff_Region(t *testing.T) {
	t.Run("Create staff with invalid region returns 400", func(t *testing.T) {
		r, _, _, _, _ := setupHospitalAdminTestRouter()

		reqBody := dto.HospitalAdminCreateStaffRequest{
			Email:      "staff@test.com",
			Password:   "password123",
			FirstName:  "Abebe",
			MiddleName: "Kebede",
			LastName:   "Molla",
			Role:       entity.RoleLiaisonOfficer,
			Region:     pointerToString("Invalid Region Name"),
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/hospital-admin/staff", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid region")
	})

	t.Run("Create staff with valid region succeeds", func(t *testing.T) {
		r, mockUser, _, _, _ := setupHospitalAdminTestRouter()

		mockUser.On("HospitalAdminCreateStaff", mock.Anything, mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Region != nil && string(*u.Region) == "Amhara"
		}), "password123").Return(nil)

		reqBody := dto.HospitalAdminCreateStaffRequest{
			Email:      "staff@test.com",
			Password:   "password123",
			FirstName:  "Abebe",
			MiddleName: "Kebede",
			LastName:   "Molla",
			Role:       entity.RoleLiaisonOfficer,
			Region:     pointerToString("Amhara"),
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/hospital-admin/staff", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockUser.AssertExpectations(t)
	})
}

func pointerToString(s string) *string {
	return &s
}


