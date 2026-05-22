package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
)

// MockReferenceUseCase simulates the DB logic for dropdown endpoints
type MockReferenceUseCase struct {
	mock.Mock
}

func (m *MockReferenceUseCase) GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error) {
	args := m.Called(ctx, tier)
	return args.Get(0).([]entity.Hospital), args.Error(1)
}

func (m *MockReferenceUseCase) GetDepartments(ctx context.Context) ([]entity.Department, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Department), args.Error(1)
}

func (m *MockReferenceUseCase) ListICDCodes(ctx context.Context, search string, category string, page int, pageSize int) ([]entity.ICDCode, int64, error) {
	args := m.Called(ctx, search, category, page, pageSize)
	return args.Get(0).([]entity.ICDCode), int64(args.Int(1)), args.Error(2)
}

func (m *MockReferenceUseCase) ListICDCategories(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockReferenceUseCase) GetNetworkedHospitals(ctx context.Context, senderID uuid.UUID) ([]entity.Hospital, error) {
	args := m.Called(ctx, senderID)
	return args.Get(0).([]entity.Hospital), args.Error(1)
}

func (m *MockReferenceUseCase) GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error) {
	args := m.Called(ctx, hospitalID)
	return args.Get(0).([]entity.Department), args.Error(1)
}

func (m *MockReferenceUseCase) GetLiaisonsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, hospitalID)
	return args.Get(0).([]entity.User), args.Error(1)
}

func TestReferenceEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferenceUseCase)
	handler := handlers.NewReferenceHandler(mockUC)

	router := gin.Default()
	router.GET("/api/v1/reference/hospitals", handler.GetHospitals)
	router.GET("/api/v1/reference/departments", handler.GetDepartments)
	router.GET("/api/v1/reference/icd-codes", handler.ListICDCodes)
	router.GET("/api/v1/reference/icd-categories", handler.ListICDCategories)
	router.GET("/api/v1/reference/networked-hospitals", handler.GetNetworkedHospitals)
	router.GET("/api/v1/reference/hospitals/:id/departments", handler.GetHospitalDepartments)
	router.GET("/api/v1/reference/regions", handler.GetRegions)

	t.Run("Get Global Hospitals List", func(t *testing.T) {
		hosp := entity.Hospital{
			ID:        uuid.New(),
			Name:      "Tikur Anbessa Specialized Hospital",
			TierLevel: entity.HospitalTier("SPECIALIZED"),
			Region:    "Addis Ababa",
		}
		mockUC.On("GetHospitals", mock.Anything, "").Return([]entity.Hospital{hosp}, nil)

		req := httptest.NewRequest("GET", "/api/v1/reference/hospitals", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []dto.HospitalResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		hospitals := resp.Data
		assert.Len(t, hospitals, 1)
		assert.Equal(t, "Tikur Anbessa Specialized Hospital", hospitals[0].Name)
	})

	t.Run("List All ICD Codes", func(t *testing.T) {
		icd := entity.ICDCode{
			Code:        "A00",
			Description: "Cholera",
		}
		mockUC.On("ListICDCodes", mock.Anything, "", "", 1, 30).Return([]entity.ICDCode{icd}, 1, nil)

		req := httptest.NewRequest("GET", "/api/v1/reference/icd-codes", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data  []entity.ICDCode `json:"data"`
			Total int64            `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		codes := resp.Data
		assert.Len(t, codes, 1)
		assert.Equal(t, "A00", codes[0].Code)
		assert.Equal(t, int64(1), resp.Total)
	})

	t.Run("List ICD Categories", func(t *testing.T) {
		categories := []string{"Certain infectious and parasitic diseases", "Diseases of the circulatory system"}
		mockUC.On("ListICDCategories", mock.Anything).Return(categories, nil)

		req := httptest.NewRequest("GET", "/api/v1/reference/icd-categories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp dto.RegionListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data, "Certain infectious and parasitic diseases")
		assert.Contains(t, resp.Data, "Diseases of the circulatory system")
	})

	t.Run("Get Networked Hospitals", func(t *testing.T) {
		hosp := entity.Hospital{
			ID:        uuid.New(),
			Name:      "Addis Ababa General Hospital",
			TierLevel: entity.HospitalTier("GENERAL"),
		}
		senderID := uuid.New()
		mockUC.On("GetNetworkedHospitals", mock.Anything, senderID).Return([]entity.Hospital{hosp}, nil)

		// Create a separate route for this test to inject the context
		testRouter := gin.Default()
		testRouter.Use(func(c *gin.Context) {
			c.Set("hospID", &senderID)
			c.Next()
		})
		testRouter.GET("/api/v1/reference/networked-hospitals", handler.GetNetworkedHospitals)

		req := httptest.NewRequest("GET", "/api/v1/reference/networked-hospitals", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []dto.HospitalResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		hospitals := resp.Data
		assert.Len(t, hospitals, 1)
		assert.Equal(t, "Addis Ababa General Hospital", hospitals[0].Name)
	})

	t.Run("Get Hospital Departments", func(t *testing.T) {
		dept := entity.Department{
			ID:   uuid.New(),
			Name: "Neurology",
		}
		hospitalID := uuid.New()
		mockUC.On("GetHospitalDepartments", mock.Anything, hospitalID).Return([]entity.Department{dept}, nil)

		req := httptest.NewRequest("GET", "/api/v1/reference/hospitals/"+hospitalID.String()+"/departments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data []dto.DepartmentResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		depts := resp.Data
		assert.Len(t, depts, 1)
		assert.Equal(t, "Neurology", depts[0].Name)
	})

	t.Run("Get Ethiopian Regions List", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/reference/regions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp dto.RegionListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data, "Addis Ababa")
		assert.Contains(t, resp.Data, "Oromia")
	})
}
