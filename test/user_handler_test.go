package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/usecase"
)

func setupUserHandlerTestRouter() (*gin.Engine, *MockUserUseCase, *MockDepartmentUseCase) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUserUC := new(MockUserUseCase)
	mockDeptUC := new(MockDepartmentUseCase)

	userHandler := handlers.NewUserHandler(mockUserUC, mockDeptUC)

	// Mock System Admin context middleware
	authMiddleware := func(c *gin.Context) {
		adminID := uuid.New()
		c.Set("userID", adminID)
		c.Set("role", entity.RoleSystemSuperAdmin)
		c.Next()
	}

	api := r.Group("/api/v1/system-admin/users")
	api.Use(authMiddleware)
	{
		api.POST("", userHandler.CreateUser)
		api.PUT("/:id", userHandler.UpdateUser)
		api.PATCH("/:id/role", userHandler.AssignRole)
	}

	return r, mockUserUC, mockDeptUC
}

func TestUserHandler_CreateUser_ValidationFailures(t *testing.T) {
	hospID := uuid.New().String()
	deptID := uuid.New().String()

	tests := []struct {
		name          string
		role          entity.UserRole
		hospID        *string
		deptID        *string
		expectedMsg   string
		mockValidate  func(*MockDepartmentUseCase)
	}{
		{
			name:        "MOH analyst with hospital",
			role:        entity.RoleMohAnalyst,
			hospID:      &hospID,
			deptID:      nil,
			expectedMsg: "does not accept a hospital",
		},
		{
			name:        "Liaison officer without hospital",
			role:        entity.RoleLiaisonOfficer,
			hospID:      nil,
			deptID:      nil,
			expectedMsg: "requires a hospital",
		},
		{
			name:        "Liaison officer with department",
			role:        entity.RoleLiaisonOfficer,
			hospID:      &hospID,
			deptID:      &deptID,
			expectedMsg: "does not accept a department",
		},
		{
			name:        "Referring doctor without department",
			role:        entity.RoleReferringDoctor,
			hospID:      &hospID,
			deptID:      nil,
			expectedMsg: "requires a department",
		},
		{
			name:        "Referring doctor with unconnected department",
			role:        entity.RoleReferringDoctor,
			hospID:      &hospID,
			deptID:      &deptID,
			expectedMsg: "department not connected",
			mockValidate: func(m *MockDepartmentUseCase) {
				m.On("ValidateDepartmentForHospital", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("department not connected"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, mockDept := setupUserHandlerTestRouter()
			if tt.mockValidate != nil {
				tt.mockValidate(mockDept)
			}

			reqBody := map[string]interface{}{
				"email":       "test@hospital.com",
				"password":    "securePass123",
				"first_name":  "John",
				"middle_name": "Middle",
				"last_name":   "Doe",
				"role":        tt.role,
			}
			if tt.hospID != nil {
				reqBody["hospital_id"] = *tt.hospID
			}
			if tt.deptID != nil {
				reqBody["department_id"] = *tt.deptID
			}

			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", "/api/v1/system-admin/users", bytes.NewBuffer(body))
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
			assert.Contains(t, resp.Body.String(), tt.expectedMsg)
		})
	}
}

func TestUserHandler_UpdateUser_PartialUpdatesAndValidationFailures(t *testing.T) {
	uID := uuid.New()
	hospID := uuid.New()
	deptID := uuid.New()

	t.Run("Update role to Liaison Officer without clearing existing department succeeds and auto-clears department", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		// Existing user is a Referring Doctor with a hospital and department
		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Role == entity.RoleLiaisonOfficer && u.DepartmentID == nil
		})).Return(nil)

		// Change role to Liaison Officer (which forbids department) without explicitly clearing department in the request
		reqBody := map[string]interface{}{
			"role": entity.RoleLiaisonOfficer,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Update hospital and department to unconnected pair", func(t *testing.T) {
		r, mockUser, mockDept := setupUserHandlerTestRouter()

		// Existing user is a Referring Doctor with a hospital and department
		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)

		newHospID := uuid.New()
		newDeptID := uuid.New()

		mockDept.On("ValidateDepartmentForHospital", mock.Anything, newHospID, newDeptID).
			Return(errors.New("department not connected to new hospital"))

		// Try to update both to unconnected ones
		newHospStr := newHospID.String()
		newDeptStr := newDeptID.String()
		reqBody := map[string]interface{}{
			"hospital_id":   newHospStr,
			"department_id": newDeptStr,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "department not connected to new hospital")
	})

	t.Run("Clean hospital/department update succeeds", func(t *testing.T) {
		r, mockUser, mockDept := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)

		newHospID := uuid.New()
		newDeptID := uuid.New()

		mockDept.On("ValidateDepartmentForHospital", mock.Anything, newHospID, newDeptID).Return(nil)
		mockUser.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

		newHospStr := newHospID.String()
		newDeptStr := newDeptID.String()
		reqBody := map[string]interface{}{
			"hospital_id":   newHospStr,
			"department_id": newDeptStr,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
		mockDept.AssertExpectations(t)
	})

	t.Run("Clean partial update to Liaison Officer succeeds if department is explicitly cleared", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Role == entity.RoleLiaisonOfficer && u.HospitalID != nil && *u.HospitalID == hospID && u.DepartmentID == nil
		})).Return(nil)

		reqBody := map[string]interface{}{
			"role":          entity.RoleLiaisonOfficer,
			"department_id": "", // explicitly clear department
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("Clean partial update to Liaison Officer succeeds and auto-clears department if department is omitted from request", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Role == entity.RoleLiaisonOfficer && u.HospitalID != nil && *u.HospitalID == hospID && u.DepartmentID == nil
		})).Return(nil)

		reqBody := map[string]interface{}{
			"role": entity.RoleLiaisonOfficer,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("Fails if updating to Liaison Officer and explicitly sending a department_id in request", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)

		reqBody := map[string]interface{}{
			"role":          entity.RoleLiaisonOfficer,
			"department_id": deptID.String(), // explicitly passing a department
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "does not accept a department")
	})
}

func TestUserHandler_AssignRole_ValidationFailures(t *testing.T) {
	uID := uuid.New()
	hospID := uuid.New()
	deptID := uuid.New()

	t.Run("Succeeds and auto-clears forbidden department on role assignment", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleReferringDoctor,
			HospitalID:   &hospID,
			DepartmentID: &deptID,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("AssignRole", mock.Anything, uID, entity.RoleLiaisonOfficer).Return(nil)

		reqBody := map[string]interface{}{
			"role": entity.RoleLiaisonOfficer,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/api/v1/system-admin/users/"+uID.String()+"/role", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("Succeeds if current state has no department", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:           uID,
			Role:         entity.RoleLiaisonOfficer,
			HospitalID:   &hospID,
			DepartmentID: nil,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("AssignRole", mock.Anything, uID, entity.RoleHospitalAdmin).Return(nil)

		reqBody := map[string]interface{}{
			"role": entity.RoleHospitalAdmin,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/api/v1/system-admin/users/"+uID.String()+"/role", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})
}

func TestUserHandler_UpdateUser_UniquenessFailures(t *testing.T) {
	uID := uuid.New()
	hospID := uuid.New()

	t.Run("Update user with taken email returns 400", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:         uID,
			Role:       entity.RoleLiaisonOfficer,
			HospitalID: &hospID,
			Email:      "old@test.com",
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("UpdateUser", mock.Anything, mock.Anything).Return(usecase.ErrEmailExists)

		reqBody := map[string]interface{}{
			"email": "taken@test.com",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), usecase.ErrEmailExists.Error())
	})
}

func TestUserHandler_CreateAndUpdate_Region(t *testing.T) {
	uID := uuid.New()

	t.Run("Create user with invalid region returns 400", func(t *testing.T) {
		r, _, _ := setupUserHandlerTestRouter()

		reqBody := map[string]interface{}{
			"email":       "new@test.com",
			"password":    "password123",
			"first_name":  "Abebe",
			"middle_name": "Kebede",
			"last_name":   "Molla",
			"role":        entity.RoleSystemSuperAdmin,
			"region":      "Invalid Region Name",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/system-admin/users", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid region")
	})

	t.Run("Create user with valid region succeeds", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		mockUser.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Region != nil && string(*u.Region) == "Addis Ababa"
		}), "password123").Return(nil)

		reqBody := map[string]interface{}{
			"email":       "new@test.com",
			"password":    "password123",
			"first_name":  "Abebe",
			"middle_name": "Kebede",
			"last_name":   "Molla",
			"role":        entity.RoleSystemSuperAdmin,
			"region":      "Addis Ababa",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/system-admin/users", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("Update user with invalid region returns 400", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:         uID,
			Role:       entity.RoleSystemSuperAdmin,
			Email:      "admin@test.com",
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)

		reqBody := map[string]interface{}{
			"region": "Invalid Region Name",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid region")
	})

	t.Run("Update user with valid region succeeds", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		existingUser := &entity.User{
			ID:         uID,
			Role:       entity.RoleSystemSuperAdmin,
			Email:      "admin@test.com",
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Region != nil && string(*u.Region) == "Amhara"
		})).Return(nil)

		reqBody := map[string]interface{}{
			"region": "Amhara",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})

	t.Run("Update user to clear region succeeds", func(t *testing.T) {
		r, mockUser, _ := setupUserHandlerTestRouter()

		reg := entity.RegionAddisAbaba
		existingUser := &entity.User{
			ID:         uID,
			Role:       entity.RoleSystemSuperAdmin,
			Email:      "admin@test.com",
			Region:     &reg,
		}
		mockUser.On("GetUserByID", mock.Anything, uID, mock.Anything).Return(existingUser, nil)
		mockUser.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
			return u.Region == nil
		})).Return(nil)

		reqBody := map[string]interface{}{
			"region": "",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/system-admin/users/"+uID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUser.AssertExpectations(t)
	})
}



