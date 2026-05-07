package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
)

func setupPostAcceptanceTestRouter() (*gin.Engine, *MockReferralUseCase, *MockTriageUseCase, *MockSchedulingUseCase, *MockArrivalUseCase, *MockClinicalUseCase, *MockCapacityManagementUseCase, *MockDailyWeightUseCase, *MockSchedulerServiceUseCase, *MockInAppNotificationUseCase) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockReferralUC := new(MockReferralUseCase)
	mockTriageUC := new(MockTriageUseCase)
	mockSchedulingUC := new(MockSchedulingUseCase)
	mockArrivalUC := new(MockArrivalUseCase)
	mockClinicalUC := new(MockClinicalUseCase)
	mockCapacityUC := new(MockCapacityManagementUseCase)
	mockDailyWeightUC := new(MockDailyWeightUseCase)
	mockSchedulerUC := new(MockSchedulerServiceUseCase)
	mockInAppNotifUC := new(MockInAppNotificationUseCase)

	specialistHandler := handlers.NewSpecialistHandler(mockReferralUC, mockSchedulingUC, mockTriageUC)
	scheduleHandler := handlers.NewScheduleHandler(mockCapacityUC)
	deptHeadHandler := handlers.NewDepartmentHeadHandler(mockCapacityUC, mockSchedulingUC, mockTriageUC)
	receptionistHandler := handlers.NewReceptionistHandler(mockReferralUC, mockArrivalUC)
	clinicalHandler := handlers.NewClinicalHandler(mockClinicalUC)
	jobHandler := handlers.NewJobHandler(mockCapacityUC, nil, mockDailyWeightUC, mockSchedulerUC)
	inAppNotifHandler := handlers.NewInAppNotificationHandler(mockInAppNotifUC)

	// Mock JWT Middleware equivalent
	authMiddleware := func(c *gin.Context) {
		userID := uuid.New()
		hospID := uuid.New()
		deptID := uuid.New()
		c.Set("userID", userID)
		c.Set("role", entity.RoleReceivingSpecialist)
		c.Set("hospID", &hospID)
		c.Set("deptID", &deptID)
		c.Next()
	}

	api := r.Group("/api/v1")
	api.Use(authMiddleware)
	{
		// Specialist
		spec := api.Group("/specialist/referrals")
		{
			spec.GET("/triage-queue", specialistHandler.GetTriageQueue)
			spec.POST("/:id/triage-severity", specialistHandler.SetManualSeverity)
			spec.GET("/capacity", specialistHandler.GetCapacity)
			spec.POST("/:id/emergency-schedule", specialistHandler.ManualEmergencySchedule)
		}

		// Dept Head
		dh := api.Group("/department-head")
		{
			dh.GET("/capacity/overrides", deptHeadHandler.ListOverrides)
			dh.POST("/capacity/overrides", deptHeadHandler.CreateOverride)
			dh.PUT("/capacity/overrides/:id", deptHeadHandler.UpdateOverride)
			dh.DELETE("/capacity/overrides/:id", deptHeadHandler.DeleteOverride)
			dh.GET("/schedule", scheduleHandler.GetSchedule)
			dh.PUT("/schedule/:id/max-slots", scheduleHandler.UpdateMaxSlots)
			dh.POST("/schedule/batch", deptHeadHandler.BatchSchedule)
		}

		// Receptionist
		rec := api.Group("/receptionist/referrals")
		{
			rec.GET("/schedule", receptionistHandler.GetSchedule)
			rec.POST("/:id/arrive", receptionistHandler.ConfirmArrival)
			rec.POST("/:id/assign-doctor", receptionistHandler.AssignDoctor)
			rec.POST("/walk-in", receptionistHandler.RegisterWalkIn)
			rec.POST("/:id/miss", receptionistHandler.MarkMissed)
		}

		// Clinical
		clin := api.Group("/referrals/:id/clinical")
		{
			clin.GET("/history", clinicalHandler.GetHistory)
			clin.POST("/updates", clinicalHandler.AddUpdate)
			clin.POST("/outcome", clinicalHandler.RecordOutcome)
		}

		// Internal Jobs
		internalJobs := api.Group("/internal/jobs")
		{
			internalJobs.POST("/update-waiting-weights", jobHandler.UpdateWaitingWeights)
			internalJobs.POST("/run-scheduler-cycle", jobHandler.RunSchedulerCycle)
		}

		// In-App Notifications
		notif := api.Group("/me/notifications")
		{
			notif.GET("", inAppNotifHandler.ListNotifications)
			notif.POST("/:id/read", inAppNotifHandler.MarkRead)
			notif.POST("/read-all", inAppNotifHandler.MarkAllRead)
			notif.GET("/unread-count", inAppNotifHandler.GetUnreadCount)
		}
	}

	return r, mockReferralUC, mockTriageUC, mockSchedulingUC, mockArrivalUC, mockClinicalUC, mockCapacityUC, mockDailyWeightUC, mockSchedulerUC, mockInAppNotifUC
}

func TestSpecialistEndpoints(t *testing.T) {
	r, _, mockTriage, mockSched, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
	referralID := uuid.New()

	t.Run("Set Manual Severity", func(t *testing.T) {
		reqBody := dto.SetManualSeverityRequest{
			Score:         85.5,
			Justification: "Clinical worsening",
		}
		mockTriage.On("SetManualSeverity", mock.Anything, referralID, mock.Anything, 85.5, "Clinical worsening").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/specialist/referrals/"+referralID.String()+"/triage-severity", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTriage.AssertExpectations(t)
	})

	t.Run("Manual Emergency Schedule", func(t *testing.T) {
		appDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		reqBody := dto.ManualEmergencyScheduleRequest{
			AppointmentDate: appDate,
			Justification:   "Immediate intervention required",
		}
		mockSched.On("ManualEmergencySchedule", mock.Anything, referralID, mock.Anything, "Immediate intervention required", mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/specialist/referrals/"+referralID.String()+"/emergency-schedule", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockSched.AssertExpectations(t)
	})

	t.Run("Get Triage Queue", func(t *testing.T) {
		mockTriage.On("ListForTriage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]dto.TriageListResponse{}, int64(0), nil)

		req, _ := http.NewRequest("GET", "/api/v1/specialist/referrals/triage-queue", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTriage.AssertExpectations(t)
	})

	t.Run("Get Capacity Status", func(t *testing.T) {
		mockSched.On("GetCapacityStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]dto.CapacityStatusResponse{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/specialist/referrals/capacity", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockSched.AssertExpectations(t)
	})
}

func TestDepartmentHeadEndpoints(t *testing.T) {
	r, _, _, mockSched, _, _, mockCapacity, _, _, _ := setupPostAcceptanceTestRouter()
	scheduleID := uuid.New()
	overrideID := uuid.New()

	t.Run("List Overrides", func(t *testing.T) {
		mockCapacity.On("GetOverrides", mock.Anything, mock.Anything, mock.Anything).Return([]entity.CapacityOverride{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/overrides", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Create Override", func(t *testing.T) {
		reqBody := dto.CreateOverrideRequest{
			Date:     time.Now().Add(48 * time.Hour).Format("2006-01-02"),
			NewLimit: 15,
			Reason:   "Staff training",
		}
		mockCapacity.On("CreateOverride", mock.Anything, mock.Anything, mock.Anything, mock.Anything, 15, "Staff training", mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/department-head/capacity/overrides", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Update Override", func(t *testing.T) {
		reqBody := dto.UpdateOverrideRequest{
			NewLimit: 12,
			Reason:   "Staff shortage",
		}
		mockCapacity.On("UpdateOverride", mock.Anything, overrideID, 12, "Staff shortage", mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/capacity/overrides/"+overrideID.String(), bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Delete Override", func(t *testing.T) {
		mockCapacity.On("DeleteOverride", mock.Anything, overrideID, mock.Anything).Return(nil)

		req, _ := http.NewRequest("DELETE", "/api/v1/department-head/capacity/overrides/"+overrideID.String(), nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Get Schedule", func(t *testing.T) {
		mockCapacity.On("GetSchedule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]entity.DailySchedule{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/schedule", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Update Max Slots", func(t *testing.T) {
		reqBody := dto.UpdateMaxSlotsRequest{MaxSlots: 20}
		mockCapacity.On("UpdateMaxSlots", mock.Anything, scheduleID, 20, mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/schedule/"+scheduleID.String()+"/max-slots", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Batch Schedule", func(t *testing.T) {
		mockSched.On("BatchSchedule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&dto.BatchScheduleResult{}, nil)

		req, _ := http.NewRequest("POST", "/api/v1/department-head/schedule/batch", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockSched.AssertExpectations(t)
	})
}

func TestReceptionistEndpoints(t *testing.T) {
	r, _, _, _, mockArrival, _, _, _, _, _ := setupPostAcceptanceTestRouter()
	queueID := uuid.New()
	referralID := uuid.New()

	t.Run("Get Schedule", func(t *testing.T) {
		mockArrival.On("GetTodayAndTomorrowSchedule", mock.Anything, mock.Anything, mock.Anything).Return([]*entity.TriageQueue{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/receptionist/referrals/schedule", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})

	t.Run("Confirm Arrival", func(t *testing.T) {
		mockArrival.On("ConfirmArrival", mock.Anything, queueID, mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/"+queueID.String()+"/arrive", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})

	t.Run("Assign Doctor", func(t *testing.T) {
		doctorID := uuid.New()
		reqBody := dto.AssignDoctorRequest{DoctorID: doctorID}
		mockArrival.On("AssignDoctor", mock.Anything, queueID, doctorID, mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/"+queueID.String()+"/assign-doctor", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})

	t.Run("Register Walk-in", func(t *testing.T) {
		reqBody := dto.WalkInRequest{ReferralID: referralID}
		mockArrival.On("RegisterWalkIn", mock.Anything, referralID, mock.Anything, mock.Anything, mock.Anything).Return(&entity.TriageQueue{}, nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/walk-in", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})

	t.Run("Mark Missed", func(t *testing.T) {
		reqBody := dto.MarkMissedRequest{MissReason: "PATIENT_NO_SHOW"}
		mockArrival.On("MarkMissed", mock.Anything, queueID, entity.MissPatientNoShow, mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/"+queueID.String()+"/miss", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})
}

func TestClinicalEndpoints(t *testing.T) {
	r, _, _, _, _, mockClinical, _, _, _, _ := setupPostAcceptanceTestRouter()
	referralID := uuid.New()

	t.Run("Add Clinical Update", func(t *testing.T) {
		reqBody := dto.AddClinicalUpdateRequest{
			UpdateReason:  "CONDITION_CHANGE",
			ClinicalNotes: "Patient condition deteriorated",
		}
		mockClinical.On("AddClinicalUpdate", mock.Anything, referralID, mock.Anything, reqBody).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/referrals/"+referralID.String()+"/clinical/updates", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockClinical.AssertExpectations(t)
	})

	t.Run("Record Outcome", func(t *testing.T) {
		reqBody := dto.RecordOutcomeRequest{
			Outcome:      "improved",
			OutcomeNotes: "Discharged with medications",
		}
		mockClinical.On("RecordOutcome", mock.Anything, referralID, mock.Anything, reqBody).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/referrals/"+referralID.String()+"/clinical/outcome", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockClinical.AssertExpectations(t)
	})

	t.Run("Get Clinical History", func(t *testing.T) {
		mockClinical.On("GetClinicalHistory", mock.Anything, referralID, mock.Anything).Return([]entity.ClinicalUpdate{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/referrals/"+referralID.String()+"/clinical/history", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockClinical.AssertExpectations(t)
	})
}

func TestInternalJobEndpoints(t *testing.T) {
	r, _, _, _, _, _, _, mockDailyWeight, mockScheduler, _ := setupPostAcceptanceTestRouter()

	t.Run("Update Waiting Weights - Success", func(t *testing.T) {
		mockDailyWeight.On("Execute", mock.Anything, mock.Anything).Return("Successfully updated 5 records", nil).Once()

		req, _ := http.NewRequest("POST", "/api/v1/internal/jobs/update-waiting-weights", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &body)
		assert.Equal(t, "Successfully updated 5 records", body["message"])
		mockDailyWeight.AssertExpectations(t)
	})

	t.Run("Update Waiting Weights - Idempotent", func(t *testing.T) {
		mockDailyWeight.On("Execute", mock.Anything, mock.Anything).Return("Already updated today", nil).Once()

		req, _ := http.NewRequest("POST", "/api/v1/internal/jobs/update-waiting-weights", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &body)
		assert.Equal(t, "Already updated today", body["message"])
		mockDailyWeight.AssertExpectations(t)
	})

	t.Run("Run Scheduler Cycle - Success", func(t *testing.T) {
		mockScheduler.On("RunSchedulerCycle", mock.Anything, mock.Anything).Return(&dto.BatchScheduleResult{
			ScheduledCount: 3,
			WaitingCount:   10,
		}, nil).Once()

		req, _ := http.NewRequest("POST", "/api/v1/internal/jobs/run-scheduler-cycle", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &body)
		assert.True(t, body["success"].(bool))
		data := body["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["scheduled_count"])
		mockScheduler.AssertExpectations(t)
	})
}

func TestInAppNotificationEndpoints(t *testing.T) {
	r, _, _, _, _, _, _, _, _, mockInAppNotif := setupPostAcceptanceTestRouter()
	notifID := uuid.New()

	t.Run("List Notifications - Success", func(t *testing.T) {
		mockInAppNotif.On("ListForUser", mock.Anything, mock.Anything, mock.Anything, 20, 1).
			Return([]entity.InAppNotification{}, int64(0), int64(0), nil)

		req, _ := http.NewRequest("GET", "/api/v1/me/notifications?limit=20&page=1", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockInAppNotif.AssertExpectations(t)
	})

	t.Run("Mark Read", func(t *testing.T) {
		mockInAppNotif.On("MarkRead", mock.Anything, notifID, mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/me/notifications/"+notifID.String()+"/read", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockInAppNotif.AssertExpectations(t)
	})

	t.Run("Mark All Read", func(t *testing.T) {
		mockInAppNotif.On("MarkAllRead", mock.Anything, mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/me/notifications/read-all", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockInAppNotif.AssertExpectations(t)
	})

	t.Run("Get Unread Count", func(t *testing.T) {
		mockInAppNotif.On("GetUnreadCount", mock.Anything, mock.Anything).Return(int64(5), nil)

		req, _ := http.NewRequest("GET", "/api/v1/me/notifications/unread-count", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &body)
		assert.Equal(t, float64(5), body["unread_count"])
		mockInAppNotif.AssertExpectations(t)
	})
}
