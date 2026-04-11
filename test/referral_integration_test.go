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

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

// --- DOCTOR TESTS ---
func TestDoctorOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewDoctorHandler(mockUC)
	doctorID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/doctor/referrals", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.CreateOrSubmit)

	router.POST("/api/v1/doctor/referrals/:id/cancel", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Next()
	}, handler.Cancel)

	router.PUT("/api/v1/doctor/referrals/:id", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Next()
	}, handler.UpdateDraft)

	router.PUT("/api/v1/doctor/referrals/:id/submit", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Next()
	}, handler.SubmitReferral)

	t.Run("Create Draft", func(t *testing.T) {
		liaisonID := uuid.New()
		reqPayload := dto.CreateReferralRequest{
			Status: "DRAFT", 
			PatientID: uuid.New(),
			TargetHospitalID: uuid.New(),
			TargetDeptID: uuid.New(),
			LiaisonOfficerID: &liaisonID,
		}
		mockUC.On("CreateDraftOrSubmit", mock.Anything, doctorID, hospID, reqPayload).Return(&dto.ReferralCreationResponse{
			Referral: &entity.Referral{ID: uuid.New()},
		}, nil)

		body, _ := json.Marshal(reqPayload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/doctor/referrals", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("Cancel Referral", func(t *testing.T) {
		refID := uuid.New()
		reason := "Patient decided not to proceed"
		mockUC.On("CancelReferral", mock.Anything, refID, doctorID, reason).Return(nil)

		payload := dto.CancelReferralRequest{Reason: reason}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/doctor/referrals/"+refID.String()+"/cancel", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("Cancel Already Cancelled", func(t *testing.T) {
		refID := uuid.New()
		reason := "Patient decided not to proceed"
		mockUC.On("CancelReferral", mock.Anything, refID, doctorID, reason).Return(errors.New("referral is already cancelled"))

		payload := dto.CancelReferralRequest{Reason: reason}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/doctor/referrals/"+refID.String()+"/cancel", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "referral is already cancelled", resp["error"])
		mockUC.AssertExpectations(t)
	})

	t.Run("Create Default Submitted", func(t *testing.T) {
		liaisonID := uuid.New()
		reqPayload := dto.CreateReferralRequest{
			// Status is missing
			PatientID: uuid.New(),
			TargetHospitalID: uuid.New(),
			TargetDeptID: uuid.New(),
			LiaisonOfficerID: &liaisonID,
		}
		// Expectation should have Status: "SUBMITTED"
		expectedReq := reqPayload
		expectedReq.Status = "SUBMITTED"
		mockUC.On("CreateDraftOrSubmit", mock.Anything, doctorID, hospID, expectedReq).Return(&dto.ReferralCreationResponse{
			Referral: &entity.Referral{ID: uuid.New()},
		}, nil)

		body, _ := json.Marshal(reqPayload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/doctor/referrals", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("Update Draft Only", func(t *testing.T) {
		refID := uuid.New()
		liaisonID := uuid.New()
		reqPayload := dto.UpdateReferralRequest{
			PatientID: uuid.New(),
			TargetHospitalID: uuid.New(),
			TargetDeptID: uuid.New(),
			LiaisonOfficerID: &liaisonID,
			ClinicalSummary: "Updating draft details",
		}
		// In Draft mode, submit flag is false
		mockUC.On("UpdateAndResubmit", mock.Anything, refID, doctorID, reqPayload, false).Return(&dto.ReferralCreationResponse{
			Referral: &entity.Referral{ID: refID},
		}, nil)

		body, _ := json.Marshal(reqPayload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/doctor/referrals/"+refID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("Submit Update", func(t *testing.T) {
		refID := uuid.New()
		liaisonID := uuid.New()
		reqPayload := dto.UpdateReferralRequest{
			PatientID: uuid.New(),
			TargetHospitalID: uuid.New(),
			TargetDeptID: uuid.New(),
			LiaisonOfficerID: &liaisonID,
			ClinicalSummary: "Final submission",
		}
		// In Submit mode, submit flag is true
		mockUC.On("UpdateAndResubmit", mock.Anything, refID, doctorID, reqPayload, true).Return(&dto.ReferralCreationResponse{
			Referral: &entity.Referral{ID: refID},
		}, nil)

		body, _ := json.Marshal(reqPayload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/doctor/referrals/"+refID.String()+"/submit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("Redundant Submission", func(t *testing.T) {
		refID := uuid.New()
		liaisonID := uuid.New()
		reqPayload := dto.UpdateReferralRequest{
			PatientID: uuid.New(),
			TargetHospitalID: uuid.New(),
			TargetDeptID: uuid.New(),
			LiaisonOfficerID: &liaisonID,
		}
		
		mockUC.On("UpdateAndResubmit", mock.Anything, refID, doctorID, reqPayload, true).Return(nil, errors.New("referral is already submitted; please wait for review"))

		body, _ := json.Marshal(reqPayload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/doctor/referrals/"+refID.String()+"/submit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "referral is already submitted; please wait for review", resp["error"])
	})
}

// --- LIAISON TESTS ---
func TestLiaisonOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewLiaisonHandler(mockUC)
	liaisonID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/liaison/referrals/:id/forward", func(c *gin.Context) {
		c.Set("userID", liaisonID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.Forward)

	t.Run("Forward Referral", func(t *testing.T) {
		refID := uuid.New()
		mockUC.On("LiaisonForward", mock.Anything, refID, liaisonID, hospID, "").Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/liaison/referrals/"+refID.String()+"/forward", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- SPECIALIST TESTS ---
func TestSpecialistOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewSpecialistHandler(mockUC)
	specialistID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/specialist/referrals/:id/accept", func(c *gin.Context) {
		c.Set("userID", specialistID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.Accept)

	t.Run("Accept Referral", func(t *testing.T) {
		refID := uuid.New()
		mockUC.On("SpecialistAccept", mock.Anything, refID, specialistID, hospID, (*float64)(nil)).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/specialist/referrals/"+refID.String()+"/accept", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- RECEPTIONIST TESTS ---
func TestReceptionistOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReceptionistHandler(mockUC)
	receptionistID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/receptionist/referrals/:id/confirm-attendance", func(c *gin.Context) {
		c.Set("userID", receptionistID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.ConfirmAttendance)

	t.Run("Confirm Attendance", func(t *testing.T) {
		refID := uuid.New()
		mockUC.On("ConfirmAttendance", mock.Anything, refID, receptionistID, hospID, "ASSIGNED").Return(nil)

		payload := map[string]string{"status": "ASSIGNED"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/receptionist/referrals/"+refID.String()+"/confirm-attendance", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- ADMIN TESTS ---
func TestAdminOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewAdminHandler(mockUC)
	hospID := uuid.New()

	router := gin.Default()
	router.GET("/api/v1/system-admin/referrals", handler.SystemAdminList)
	router.GET("/api/v1/hospital-admin/referrals-log", func(c *gin.Context) {
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.HospitalAdminLogs)

	t.Run("System Admin List", func(t *testing.T) {
		mockUC.On("ListForSystemAdmin", mock.Anything, irepository.ReferralFilter{Limit: 20, Page: 1}).Return([]entity.Referral{}, int64(0), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/system-admin/referrals", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Hospital Admin Logs", func(t *testing.T) {
		mockUC.On("GetHospitalLogsForAdmin", mock.Anything, hospID, 20, 1).Return([]entity.ReferralStatusHistory{}, int64(0), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/hospital-admin/referrals-log", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
