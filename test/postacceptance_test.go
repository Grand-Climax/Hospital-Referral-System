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
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
)

func setupPostAcceptanceTestRouter() (*gin.Engine, *MockReferralUseCase, *MockTriageUseCase, *MockSchedulingUseCase, *MockArrivalUseCase, *MockClinicalUseCase, *MockCapacityManagementUseCase, *MockDailyWeightUseCase, *MockSchedulerServiceUseCase, *MockInAppNotificationUseCase, *MockUserUseCase, *MockNotificationUseCase, *MockMLUseCase, *MockDepartmentHeadDashboardUseCase) {
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
	mockPatientUC := new(MockPatientUseCase)
	mockNotificationUC := new(MockNotificationUseCase)
	mockMLUC := new(MockMLUseCase)
	mockDeptHeadDash := new(MockDepartmentHeadDashboardUseCase)

	specialistHandler := handlers.NewSpecialistHandler(mockReferralUC, mockSchedulingUC, mockTriageUC, mockPatientUC, mockMLUC, mockArrivalUC)
	scheduleHandler := handlers.NewScheduleHandler(mockCapacityUC)
	deptHeadHandler := handlers.NewDepartmentHeadHandler(mockCapacityUC, mockSchedulingUC, mockTriageUC)
	deptHeadDashboardHandler := handlers.NewDepartmentHeadDashboardHandler(mockDeptHeadDash)
	mockUserUC := new(MockUserUseCase)
	receptionistHandler := handlers.NewReceptionistHandler(mockReferralUC, mockArrivalUC, mockPatientUC, mockUserUC)
	clinicalHandler := handlers.NewClinicalHandler(mockClinicalUC)
	jobHandler := handlers.NewJobHandler(mockNotificationUC, mockDailyWeightUC, mockSchedulerUC, mockSchedulingUC)
	inAppNotifHandler := handlers.NewInAppNotificationHandler(mockInAppNotifUC)
	doctorHandler := handlers.NewDoctorHandler(mockReferralUC, nil, mockPatientUC, mockArrivalUC)
	liaisonHandler := handlers.NewLiaisonHandler(mockReferralUC, mockPatientUC)

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
			spec.POST("/:id/ml-severity-override", specialistHandler.MLSeverityOverride)
			spec.GET("/capacity", specialistHandler.GetCapacity)
			spec.POST("/:id/emergency-schedule", specialistHandler.ManualEmergencySchedule)
			spec.POST("/:id/return-to-triage", specialistHandler.ReturnToTriage)
		}

		// Dept Head
		dh := api.Group("/department-head")
		{
			dh.GET("/capacity/overrides", deptHeadHandler.ListOverrides)
			dh.GET("/capacity/overrides/by-month", deptHeadHandler.ListOverridesByMonth)
			dh.GET("/capacity/overrides/:id", deptHeadHandler.GetOverride)
			dh.POST("/capacity/overrides", deptHeadHandler.CreateOverride)
			dh.DELETE("/capacity/overrides/:id", deptHeadHandler.DeleteOverride)
			dh.GET("/schedule", scheduleHandler.GetSchedule)
			dh.GET("/schedule/patients", deptHeadHandler.GetScheduledPatients)
			dh.POST("/schedule/batch", deptHeadHandler.BatchSchedule)
			dh.GET("/capacity/detail", deptHeadHandler.GetCapacityDetail)
			dh.GET("/capacity/calendar", deptHeadHandler.GetCapacityCalendar)
			dh.PUT("/staff-capacity", deptHeadHandler.UpdateStaffCapacity)
			dh.GET("/daily-capacity", deptHeadHandler.GetDailyCapacity)
			dh.PUT("/daily-capacity", deptHeadHandler.UpdateDailyCapacity)

			dh.GET("/dashboard/stats", deptHeadDashboardHandler.GetDashboardStats)
			dh.GET("/dashboard/trends", deptHeadDashboardHandler.GetTrends)
			dh.GET("/triage-queue/buckets", deptHeadDashboardHandler.GetPriorityBuckets)
			dh.GET("/staff/summary", deptHeadDashboardHandler.GetStaffSummary)
			dh.GET("/activity", deptHeadDashboardHandler.GetActivity)
		}

		// Specialist Schedule Options
		api.GET("/specialist/referrals/:id/schedule-options", specialistHandler.ScheduleOptions)

		// Department Staff
		userHandlerForTest := handlers.NewUserHandler(mockUserUC, nil)
		api.GET("/departments/staff", userHandlerForTest.ListDepartmentStaff)

		// Receptionist
		recBase := api.Group("/receptionist")
		{
			recBase.GET("/doctors", receptionistHandler.ListDoctors)
			rec := recBase.Group("/referrals")
			{
				rec.GET("/upcoming", receptionistHandler.GetSchedule)
				rec.POST("/:id/arrive", receptionistHandler.ConfirmArrival)
				rec.POST("/:id/assign-doctor", receptionistHandler.AssignDoctor)
				rec.POST("/:id/miss", receptionistHandler.MarkMissed)
				rec.POST("/:id/return-to-triage", receptionistHandler.ReturnToTriage)
			}
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

		// Rejection After Send
		api.POST("/doctor/referrals/:id/reject-after-send", doctorHandler.RejectAfterSend)
		api.POST("/liaison/referrals/:id/reject-after-send", liaisonHandler.RejectAfterSend)
	}

	return r, mockReferralUC, mockTriageUC, mockSchedulingUC, mockArrivalUC, mockClinicalUC, mockCapacityUC, mockDailyWeightUC, mockSchedulerUC, mockInAppNotifUC, mockUserUC, mockNotificationUC, mockMLUC, mockDeptHeadDash
}

func TestSpecialistEndpoints(t *testing.T) {
	r, _, mockTriage, mockSched, mockArrival, _, _, _, _, _, _, _, mockML, _ := setupPostAcceptanceTestRouter()
	referralID := uuid.New()

	t.Run("Override ML Severity", func(t *testing.T) {
		reqBody := dto.MLSeverityOverrideRequest{
			Score:         85.5,
			Justification: "Clinical worsening",
		}
		mockML.On("MLSeverityOverride", mock.Anything, referralID, mock.Anything, 85.5, "Clinical worsening").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/specialist/referrals/"+referralID.String()+"/ml-severity-override", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockML.AssertExpectations(t)
	})

	t.Run("Manual Emergency Schedule", func(t *testing.T) {
		appDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		reqBody := dto.ManualEmergencyScheduleRequest{
			AppointmentDate: appDate,
			Justification:   "Immediate intervention required",
		}
		mockSched.On("ManualEmergencySchedule", mock.Anything, referralID, mock.Anything, "Immediate intervention required", mock.Anything).Return(false, nil)

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

	t.Run("Return To Triage - Success", func(t *testing.T) {
		mockArrival.On("ReturnToTriage", mock.Anything, referralID, mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/specialist/referrals/"+referralID.String()+"/return-to-triage", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})
}

func TestDepartmentHeadEndpoints(t *testing.T) {
	r, _, _, mockSched, _, _, mockCapacity, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
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
		// Pick a date safely past the buffer window so the future-date
		// check in the handler/use case never bites.
		reqBody := dto.CreateOverrideRequest{
			Date:     time.Now().Add(10 * 24 * time.Hour).Format("2006-01-02"),
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

	t.Run("Batch Schedule", func(t *testing.T) {
		mockSched.On("BatchSchedule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&dto.BatchScheduleResult{}, nil)

		req, _ := http.NewRequest("POST", "/api/v1/department-head/schedule/batch", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockSched.AssertExpectations(t)
	})

	t.Run("Batch Schedule - lease already held", func(t *testing.T) {
		// Fresh router + mocks so the assertion below does not collide
		// with the happy-path expectation above.
		r2, _, _, mockSched2, _, _, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()

		mockSched2.On("BatchSchedule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&dto.BatchScheduleResult{Message: "Batch already running for this department; try again in a few minutes"}, nil)

		req, _ := http.NewRequest("POST", "/api/v1/department-head/schedule/batch", nil)
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "already running")
		mockSched2.AssertExpectations(t)
	})

	t.Run("List Overrides By Month", func(t *testing.T) {
		mockCapacity.On("ListOverridesByYearMonth", mock.Anything, mock.Anything, mock.Anything, 2026, 5).Return([]entity.CapacityOverride{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/overrides/by-month?year=2026&month=5", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("List Overrides By Month - missing year", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/overrides/by-month", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("Get Single Override", func(t *testing.T) {
		id := uuid.New()
		mockCapacity.On("GetOverride", mock.Anything, id).Return(&entity.CapacityOverride{ID: id, NewLimit: 7, IsActive: true}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/overrides/"+id.String(), nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Get Capacity Detail", func(t *testing.T) {
		date := time.Now().Add(48 * time.Hour).Format("2006-01-02")
		mockCapacity.On("GetCapacityDetail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&dto.CapacityDetailResponse{Date: date, MaxSlots: 10, BookedSlots: 3, AvailableSlots: 7}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/detail?date="+date, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Get Capacity Detail - missing date", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/detail", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("Get Scheduled Patients", func(t *testing.T) {
		date := time.Now().Add(48 * time.Hour).Format("2006-01-02")
		mockCapacity.On("GetScheduledPatientsForDate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]entity.TriageQueue{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/schedule/patients?date="+date, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Get Capacity Calendar", func(t *testing.T) {
		mockCapacity.On("BuildCapacityCalendar", mock.Anything, mock.Anything, mock.Anything, 2026, 5).
			Return([]dto.CapacityCalendarDay{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/capacity/calendar?year=2026&month=5", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Update Staff Capacity", func(t *testing.T) {
		mockCapacity.On("UpdateStaffCapacity", mock.Anything, mock.Anything, mock.Anything, 12, mock.Anything).Return(nil)

		body, _ := json.Marshal(dto.UpdateStaffCapacityRequest{MaxCapacityOfStaff: 12})
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/staff-capacity", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCapacity.AssertExpectations(t)
	})

	t.Run("Get Daily Capacity - happy path returns current baseline", func(t *testing.T) {
		r2, _, _, _, _, _, mockCap2, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		mockCap2.On("GetDailyCapacity", mock.Anything, mock.Anything, mock.Anything).
			Return(&dto.DeptHeadDailyCapacityResponse{
				Success:            true,
				HospitalID:         uuid.New(),
				DepartmentID:       uuid.New(),
				StandardDailyLimit: 25,
				OverbookLimit:      3,
				UpdatedAt:          time.Now(),
			}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/daily-capacity", nil)
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"standard_daily_limit":25`)
		assert.Contains(t, resp.Body.String(), `"overbook_limit":3`)
		mockCap2.AssertExpectations(t)
	})

	t.Run("Get Daily Capacity - 404 when link missing", func(t *testing.T) {
		r2, _, _, _, _, _, mockCap2, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		mockCap2.On("GetDailyCapacity", mock.Anything, mock.Anything, mock.Anything).
			Return((*dto.DeptHeadDailyCapacityResponse)(nil), gorm.ErrRecordNotFound)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/daily-capacity", nil)
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		assert.Contains(t, resp.Body.String(), "hospital-department link not found")
		mockCap2.AssertExpectations(t)
	})

	t.Run("Update Daily Capacity - happy path echoes new values", func(t *testing.T) {
		r2, _, _, _, _, _, mockCap2, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		mockCap2.On("UpdateDailyCapacity", mock.Anything, mock.Anything, mock.Anything, 35, 5, mock.Anything).Return(nil)

		body, _ := json.Marshal(dto.UpdateDailyCapacityRequest{StandardDailyLimit: 35, OverbookLimit: 5})
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/daily-capacity", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "daily capacity baseline updated")
		assert.Contains(t, resp.Body.String(), `"standard_daily_limit":35`)
		assert.Contains(t, resp.Body.String(), `"overbook_limit":5`)
		mockCap2.AssertExpectations(t)
	})

	t.Run("Update Daily Capacity - zero is a valid value (paused department)", func(t *testing.T) {
		// 0 is legitimate (e.g. dept temporarily paused). The binding
		// tag is `min=0` without `required` so 0 must NOT be rejected.
		r2, _, _, _, _, _, mockCap2, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		mockCap2.On("UpdateDailyCapacity", mock.Anything, mock.Anything, mock.Anything, 0, 0, mock.Anything).Return(nil)

		body, _ := json.Marshal(dto.UpdateDailyCapacityRequest{StandardDailyLimit: 0, OverbookLimit: 0})
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/daily-capacity", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockCap2.AssertExpectations(t)
	})

	t.Run("Update Daily Capacity - negative value rejected by binding", func(t *testing.T) {
		// No mock expectation: the request must be rejected at the
		// binding layer before the use case is ever invoked.
		r2, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()

		// We can't use the struct because binding:"min=0" lives on the
		// DTO; instead build a raw JSON body with a negative value.
		raw := []byte(`{"standard_daily_limit": -1, "overbook_limit": 5}`)
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/daily-capacity", bytes.NewBuffer(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("Update Daily Capacity - 404 when link missing", func(t *testing.T) {
		r2, _, _, _, _, _, mockCap2, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		mockCap2.On("UpdateDailyCapacity", mock.Anything, mock.Anything, mock.Anything, 30, 0, mock.Anything).
			Return(gorm.ErrRecordNotFound)

		body, _ := json.Marshal(dto.UpdateDailyCapacityRequest{StandardDailyLimit: 30, OverbookLimit: 0})
		req, _ := http.NewRequest("PUT", "/api/v1/department-head/daily-capacity", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		assert.Contains(t, resp.Body.String(), "hospital-department link not found")
		mockCap2.AssertExpectations(t)
	})

	t.Run("Get Schedule single-day missing", func(t *testing.T) {
		// Fresh router so the existing GetSchedule expectation does not
		// collide with this same-day variant.
		r2, _, _, _, _, _, mockCap2, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		date := time.Now().Format("2006-01-02")
		mockCap2.On("GetSchedule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]entity.DailySchedule{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/schedule?start_date="+date+"&end_date="+date, nil)
		resp := httptest.NewRecorder()
		r2.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "No schedule log for this date yet")
		mockCap2.AssertExpectations(t)
	})
}

func TestPostV17ExtraEndpoints(t *testing.T) {
	t.Run("List Department Staff", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, mockUserUC, _, _, _ := setupPostAcceptanceTestRouter()
		mockUserUC.On("ListDepartmentStaff", mock.Anything, mock.Anything, mock.Anything).Return([]entity.User{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/departments/staff", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUserUC.AssertExpectations(t)
	})

	t.Run("Specialist Schedule Options", func(t *testing.T) {
		r, _, _, mockSched, _, _, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
		refID := uuid.New()

		mockSched.On("ListScheduleOptions", mock.Anything, refID, mock.AnythingOfType("int")).
			Return([]dto.ScheduleOption{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/specialist/referrals/"+refID.String()+"/schedule-options?days=7", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockSched.AssertExpectations(t)
	})
}

// TestDepartmentHeadDashboardEndpoints exercises the read-only widgets
// that back the dept-head landing page. Each sub-test wires its own
// router so mock expectations do not bleed across cases.
func TestDepartmentHeadDashboardEndpoints(t *testing.T) {
	t.Run("Dashboard Stats", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, mockDash := setupPostAcceptanceTestRouter()
		stats := &dto.DepartmentHeadDashboardStats{
			WaitingQueueSize:    7,
			OldestWaitingDays:   3,
			ScheduledToday:      4,
			ScheduledNext7Days:  11,
			MissedLast7Days:     2,
			PendingReferrals:    5,
			CompletedLast30Days: 18,
			ActiveStaff:         9,
			ActiveOverrides:     1,
			StatusCounts: []dto.DeptReferralStatusItem{
				{Status: "ACCEPTED", Count: 5},
			},
		}
		mockDash.On("GetDashboardStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/dashboard/stats", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"waiting_queue_size":7`)
		assert.Contains(t, resp.Body.String(), `"completed_last_30_days":18`)
		mockDash.AssertExpectations(t)
	})

	t.Run("Trends defaults to 14 days", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, mockDash := setupPostAcceptanceTestRouter()
		mockDash.On("GetTrends", mock.Anything, mock.Anything, mock.Anything, 14).
			Return([]dto.DepartmentHeadTrendPoint{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/dashboard/trends", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"days":14`)
		mockDash.AssertExpectations(t)
	})

	t.Run("Trends rejects invalid days", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()

		req, _ := http.NewRequest("GET", "/api/v1/department-head/dashboard/trends?days=-3", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("Priority Buckets", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, mockDash := setupPostAcceptanceTestRouter()
		mockDash.On("GetPriorityBuckets", mock.Anything, mock.Anything, mock.Anything).
			Return(&dto.PriorityBucketResponse{TotalWaiting: 3}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/triage-queue/buckets", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"total_waiting":3`)
		mockDash.AssertExpectations(t)
	})

	t.Run("Staff Summary", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, mockDash := setupPostAcceptanceTestRouter()
		mockDash.On("GetStaffSummary", mock.Anything, mock.Anything, mock.Anything).
			Return(&dto.StaffSummaryResponse{Department: "Cardiology"}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/staff/summary", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"department":"Cardiology"`)
		mockDash.AssertExpectations(t)
	})

	t.Run("Activity Stream", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, mockDash := setupPostAcceptanceTestRouter()
		mockDash.On("GetActivity", mock.Anything, mock.Anything, mock.Anything,
			mock.AnythingOfType("int"), mock.Anything, mock.Anything).
			Return([]dto.DepartmentHeadActivityItem{
				{ActionType: "BATCH_SCHEDULE_RUN", Summary: "Ran batch scheduling"},
			}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/department-head/activity?limit=5", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"action_type":"BATCH_SCHEDULE_RUN"`)
		mockDash.AssertExpectations(t)
	})

	t.Run("Activity rejects malformed start_date", func(t *testing.T) {
		r, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()

		req, _ := http.NewRequest("GET", "/api/v1/department-head/activity?start_date=not-a-date", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestReceptionistEndpoints(t *testing.T) {
	r, _, _, _, mockArrival, _, _, _, _, _, mockUserUC, _, _, _ := setupPostAcceptanceTestRouter()
	queueID := uuid.New()

	t.Run("List Doctors", func(t *testing.T) {
		mockUserUC.On("ListUsers", mock.Anything, mock.Anything, mock.Anything).Return([]entity.User{}, int64(0), nil)

		req, _ := http.NewRequest("GET", "/api/v1/receptionist/doctors", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUserUC.AssertExpectations(t)
	})

	t.Run("Get Schedule", func(t *testing.T) {
		mockArrival.On("GetTodayAndTomorrowSchedule", mock.Anything, mock.Anything, mock.Anything).Return([]*entity.TriageQueue{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/receptionist/referrals/upcoming", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})

	t.Run("Confirm Arrival", func(t *testing.T) {
		mockArrival.On("GetTriageQueueByReferralID", mock.Anything, queueID).Return(&entity.TriageQueue{ID: queueID, ReferralID: queueID}, nil)
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
		mockArrival.On("GetTriageQueueByReferralID", mock.Anything, queueID).Return(&entity.TriageQueue{ID: queueID, ReferralID: queueID}, nil)
		mockArrival.On("AssignDoctor", mock.Anything, queueID, doctorID, mock.Anything, mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/"+queueID.String()+"/assign-doctor", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})


	t.Run("Mark Missed", func(t *testing.T) {
		reqBody := dto.MarkMissedRequest{MissReason: "PATIENT_NO_SHOW"}
		mockArrival.On("GetTriageQueueByReferralID", mock.Anything, queueID).Return(&entity.TriageQueue{ID: queueID, ReferralID: queueID}, nil)
		mockArrival.On("MarkMissed", mock.Anything, queueID, entity.MissPatientNoShow, mock.Anything).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/"+queueID.String()+"/miss", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})

	t.Run("Return To Triage", func(t *testing.T) {
		mockArrival.On("ReturnToTriage", mock.Anything, queueID, mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/receptionist/referrals/"+queueID.String()+"/return-to-triage", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockArrival.AssertExpectations(t)
	})
}

func TestClinicalEndpoints(t *testing.T) {
	r, _, _, _, _, mockClinical, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
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
	r, _, _, _, _, _, _, mockDailyWeight, mockScheduler, _, _, _, _, _ := setupPostAcceptanceTestRouter()

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
	r, _, _, _, _, _, _, _, _, mockInAppNotif, _, _, _, _ := setupPostAcceptanceTestRouter()
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

func TestRejectionAfterSendEndpoints(t *testing.T) {
	r, mockReferral, _, _, _, _, _, _, _, _, _, _, _, _ := setupPostAcceptanceTestRouter()
	referralID := uuid.New()

	t.Run("Doctor Reject After Send", func(t *testing.T) {
		reqBody := dto.RejectDTO{Reason: "Patient decided to stay home"}
		mockReferral.On("RejectAfterSend", mock.Anything, referralID, mock.Anything, mock.Anything, entity.RoleReferringDoctor, "Patient decided to stay home").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/doctor/referrals/"+referralID.String()+"/reject-after-send", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockReferral.AssertExpectations(t)
	})

	t.Run("Liaison Reject After Send", func(t *testing.T) {
		reqBody := dto.RejectDTO{Reason: "Clerical error in department selection"}
		mockReferral.On("RejectAfterSend", mock.Anything, referralID, mock.Anything, mock.Anything, entity.RoleLiaisonOfficer, "Clerical error in department selection").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/liaison/referrals/"+referralID.String()+"/reject-after-send", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockReferral.AssertExpectations(t)
	})
}
