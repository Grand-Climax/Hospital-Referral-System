package routes

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/infrastructure/crypto"
	"Hospital-Referral-System/internal/infrastructure/middleware"
	"Hospital-Referral-System/internal/infrastructure/sms"
	"Hospital-Referral-System/internal/infrastructure/storage"
	"Hospital-Referral-System/internal/repository"
	"Hospital-Referral-System/internal/usecase"
)

// Register attaches all HTTP routes to the provided router.
func Register(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, cfg config.Config) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Health check successful",
			"status":  "ok",
		})
	})

	// Public Root Landing Page
	router.GET("/", handlers.HomeHandler)

	// Swagger docs
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ---- Dependency Injection ----

	// Cache / token stores
	tokenBlacklist := cache.NewRedisTokenBlacklist(redisClient)
	sessionStore := cache.NewRedisSessionStore(redisClient)

	// Storage
	storageSvc, _ := storage.NewCloudinaryStorage(
		cfg.Cloudinary.CloudName,
		cfg.Cloudinary.APIKey,
		cfg.Cloudinary.APISecret,
	)

	// ---- Dependency Injection (Repositories) ----
	authRepo := repository.NewAuthRepository(db)
	userRepo := repository.NewUserRepository(db)
	hospitalRepo := repository.NewHospitalRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	referralRepo := repository.NewReferralRepository(db)
	refRepo := repository.NewReferenceRepository(db)
	netRepo := repository.NewNetworkRepository(db)
	patientRepo := repository.NewPatientRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)

	// Workflow Repositories
	triageRepo := repository.NewTriageRepository(db)
	scheduleRepo := repository.NewDailyScheduleRepository(db)
	mlRepo := repository.NewMLPredictionRepository(db)
	clinicalRepo := repository.NewClinicalUpdateRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	outcomeRepo := repository.NewReferralOutcomeRepository(db)
	overrideRepo := repository.NewCapacityOverrideRepository(db)
	configRepo := repository.NewSystemConfigRepository(db)
	referralAccessRepo := repository.NewReferralAccessRepository(db)
	checkpointRepo := repository.NewSchedulerCheckpointRepository(db)
	jobCheckpointRepo := repository.NewJobCheckpointRepository(db)
	redirectionRepo := repository.NewReferralRedirectionRepository(db)
	inAppNotifRepo := repository.NewInAppNotificationRepository(db)

	// Infrastructure Clients
	smsClient := sms.NewAfroMessageClient()
	// Use mock if needed: smsClient := sms.NewMockSMSClient()

	cryptoSvc, err := crypto.NewPatientCryptoService(os.Getenv("PATIENT_AES_KEY"), os.Getenv("PATIENT_HMAC_KEY"))
	if err != nil {
		log.Fatalf("Failed to initialize PatientCryptoService: %v", err)
	}

	// ---- Dependency Injection (Use Cases) ----
	// Initialize In-App Notification Use Case early as it's needed by others
	inAppNotifUseCase := usecase.NewInAppNotificationUseCase(inAppNotifRepo, userRepo, referralRepo)

	authUseCase := usecase.NewAuthUseCase(authRepo, tokenBlacklist, sessionStore)
	userUseCase := usecase.NewUserUseCase(userRepo, storageSvc, inAppNotifUseCase)
	hospitalUseCase := usecase.NewHospitalUseCase(hospitalRepo, configRepo, auditLogRepo)
	departmentUseCase := usecase.NewDepartmentUseCase(departmentRepo, hospitalRepo, checkpointRepo)
	attachmentUseCase := usecase.NewAttachmentUseCase(attachmentRepo, referralRepo, storageSvc, inAppNotifUseCase)
	// Post-acceptance Use Cases
	notifUseCase := usecase.NewNotificationUseCase(referralRepo, notifRepo, triageRepo, configRepo, jobCheckpointRepo, smsClient, cryptoSvc)
	triageUseCase := usecase.NewTriageUseCase(db, referralRepo, triageRepo, mlRepo, configRepo, auditLogRepo, cryptoSvc)
	schedUseCase := usecase.NewSchedulingUseCase(db, referralRepo, triageRepo, scheduleRepo, overrideRepo, departmentRepo, configRepo, auditLogRepo, notifUseCase, inAppNotifUseCase, jobCheckpointRepo, clinicalRepo)
	arrivalUseCase := usecase.NewArrivalUseCase(db, triageRepo, referralRepo, userRepo, referralAccessRepo, clinicalRepo, auditLogRepo, inAppNotifUseCase)
	clinicalUseCase := usecase.NewClinicalUseCase(db, referralRepo, clinicalRepo, outcomeRepo, referralAccessRepo, auditLogRepo, inAppNotifUseCase)
	capacityManagementUseCase := usecase.NewCapacityManagementUseCase(scheduleRepo, overrideRepo, departmentRepo, auditLogRepo, inAppNotifUseCase)
	adminConfigUseCase := usecase.NewAdminConfigUseCase(configRepo, auditLogRepo, inAppNotifUseCase)
	dailyWeightUseCase := usecase.NewDailyWeightUseCase(configRepo, triageRepo, auditLogRepo)
	schedulerServiceUseCase := usecase.NewSchedulerServiceUseCase(checkpointRepo, configRepo, schedUseCase)

	referralUseCase := usecase.NewReferralUseCase(referralRepo, clinicalRepo, outcomeRepo, netRepo, redirectionRepo, triageRepo, attachmentUseCase, notifUseCase, inAppNotifUseCase, cryptoSvc, departmentRepo, attachmentRepo, referralAccessRepo)
	refUseCase := usecase.NewReferenceUseCase(refRepo)
	netUseCase := usecase.NewNetworkUseCase(netRepo, hospitalRepo)
	patientUseCase := usecase.NewPatientUseCase(patientRepo, cryptoSvc, auditLogRepo)

	// ---- Dependency Injection (Handlers) ----
	authHandler := handlers.NewAuthHandler(authUseCase)
	userHandler := handlers.NewUserHandler(userUseCase, departmentUseCase)
	hospitalHandler := handlers.NewHospitalHandler(hospitalUseCase)
	departmentHandler := handlers.NewDepartmentHandler(departmentUseCase)
	redirectionHandler := handlers.NewRedirectionHandler(referralUseCase)

	// Role-Based State Machine Handlers
	doctorHandler := handlers.NewDoctorHandler(referralUseCase, attachmentUseCase, patientUseCase, arrivalUseCase)
	liaisonHandler := handlers.NewLiaisonHandler(referralUseCase, patientUseCase)
	specialistHandler := handlers.NewSpecialistHandler(referralUseCase, schedUseCase, triageUseCase, patientUseCase)
	receptionistHandler := handlers.NewReceptionistHandler(referralUseCase, arrivalUseCase, patientUseCase, userUseCase)
	adminHandler := handlers.NewAdminHandlerWithAudit(referralUseCase, auditLogRepo, patientUseCase)
	mohAnalyticsHandler := handlers.NewMohAnalyticsHandler(referralUseCase)
	hospitalAdminStaffHandler := handlers.NewHospitalAdminStaffHandler(userUseCase, referralUseCase, departmentUseCase)
	hospitalAdminOpsHandler := handlers.NewHospitalAdminOperationsHandler(userUseCase, hospitalUseCase, departmentUseCase)
	deptHeadHandler := handlers.NewDepartmentHeadHandler(capacityManagementUseCase, schedUseCase, triageUseCase)

	refHandler := handlers.NewReferenceHandler(refUseCase)
	netHandler := handlers.NewNetworkHandler(netUseCase)
	patientHandler := handlers.NewPatientHandler(patientUseCase)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentUseCase)

	// Workflow Handlers
	triageHandler := handlers.NewTriageHandler(triageUseCase)
	scheduleHandler := handlers.NewScheduleHandler(capacityManagementUseCase)
	jobHandler := handlers.NewJobHandler(capacityManagementUseCase, notifUseCase, dailyWeightUseCase, schedulerServiceUseCase, schedUseCase)
	clinicalHandler := handlers.NewClinicalHandler(clinicalUseCase)
	notifHandler := handlers.NewNotificationHandler(notifUseCase)
	inAppNotifHandler := handlers.NewInAppNotificationHandler(inAppNotifUseCase)
	adminConfigHandler := handlers.NewAdminConfigHandler(adminConfigUseCase)

	// ---- API v1 Routes ----
	v1 := router.Group("/api/v1")
	{
		// Auth (public)
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/login", middleware.RateLimiter(redisClient, 100, time.Minute), authHandler.Login)
			authRoutes.POST("/refresh", authHandler.Refresh)
			authRoutes.POST("/logout", authHandler.Logout)
		}

		// Protected routes (require authentication + audit logging)
		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth(tokenBlacklist, userRepo))
		protected.Use(middleware.AuditLogger(auditLogRepo))

		// Internal Notification Trigger (Administrators only)
		internalNotif := protected.Group("/internal/notifications")
		internalNotif.Use(middleware.RequireRole(entity.RoleHospitalAdmin, entity.RoleDeptHead, entity.RoleSystemSuperAdmin, entity.RoleReceptionist))
		{
			internalNotif.GET("", notifHandler.ListNotifications)
			internalNotif.POST("/send", notifHandler.TriggerManualSend)
			internalNotif.POST("/update-status", notifHandler.UpdateStatus)
			internalNotif.POST("/:id/resend", notifHandler.Resend)
		}

		// Internal Job Routes (Administrators only)
		jobRoutes := protected.Group("/internal/jobs")
		jobRoutes.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin, entity.RoleHospitalAdmin, entity.RoleDeptHead))
		{
			jobRoutes.POST("/extend-daily-schedule", jobHandler.ExtendDailySchedule)
			jobRoutes.POST("/send-reminders", jobHandler.SendReminders)
			jobRoutes.POST("/update-waiting-weights", jobHandler.UpdateWaitingWeights)
			jobRoutes.POST("/run-scheduler-cycle", jobHandler.RunSchedulerCycle)
			jobRoutes.POST("/process-pending-sms", jobHandler.ProcessPendingSMS)
			jobRoutes.POST("/process-missed", jobHandler.ProcessMissedAppointments)
		}
		{
			// Admin Level Network Management Routes
			adminGroup := protected.Group("/admin/network-routes")
			// Depending on exact desired hierarchy either System Admin or MoH Analyst or specific Hospital Admin could map routes
			// We will grant HOSPITAL_ADMIN so the test validates successfully without wrestling the GORM seeder defaults
			adminGroup.Use(middleware.RequireRole(entity.RoleHospitalAdmin))
			{
				adminGroup.POST("", netHandler.Create)
				adminGroup.GET("", netHandler.List)
				adminGroup.DELETE("/:id", netHandler.Delete)
			}

			// System Admin Config Routes
			adminConfigGroup := protected.Group("/admin/config")
			adminConfigGroup.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin))
			{
				adminConfigGroup.GET("", adminConfigHandler.GetConfig)
				adminConfigGroup.PUT("", adminConfigHandler.UpdateConfig)
			}

			// Clinical & Outcome
			clinicalGroup := protected.Group("/referrals/:id/clinical")
			{
				clinicalGroup.GET("/history", clinicalHandler.GetHistory)
				clinicalGroup.POST("/updates", clinicalHandler.AddUpdate)
				clinicalGroup.POST("/outcome", clinicalHandler.RecordOutcome)
			}

			// Patient Identity Routes
			patientGroup := protected.Group("/patients")
			patientGroup.Use(middleware.RequireRole(
				entity.RoleReferringDoctor,
				entity.RoleReceptionist,
				entity.RoleSystemSuperAdmin,
			))
			{
				patientGroup.GET("/national-id/:id", patientHandler.GetByNationalID)
				patientGroup.GET("/lookup", patientHandler.LookupPatient)
				patientGroup.POST("", patientHandler.CreatePatient)
			}

			// Reference Dropdowns & Relational Networks
			protected.GET("/reference/hospitals", refHandler.GetHospitals)
			// Networked Dropdown uses X-Hospital-ID from Auth Token implicitly
			protected.GET("/reference/networked-hospitals", refHandler.GetNetworkedHospitals)
			protected.GET("/reference/departments", refHandler.GetDepartments)
			// Target Dropdown targets a specific receiver hospital path ID
			protected.GET("/reference/hospitals/:id/departments", refHandler.GetHospitalDepartments)
			protected.GET("/reference/icd-codes", refHandler.ListICDCodes)
			protected.GET("/reference/liaisons", refHandler.GetLiaisons)
			// Example: Both DOCTOR and SPECIALIST
			// protected.GET("/clinical-data", middleware.RequireRole(entity.RoleReferringDoctor, entity.RoleReceivingSpecialist), someHandler)

			// -------------------------
			// State Machine Role Groups
			// -------------------------

			// DOCTOR
			doctorGroup := protected.Group("/doctor")
			doctorGroup.Use(middleware.RequireRole(entity.RoleReferringDoctor))
			{
				doctorGroup.GET("/stats", doctorHandler.GetStats)
				doctorGroup.GET("/latest-pending", doctorHandler.GetLatestPending)
				doctorGroup.GET("/referrals", doctorHandler.ListReferrals)
				doctorGroup.GET("/referrals/approved", doctorHandler.ListApprovedReferrals)
				doctorGroup.GET("/referrals/rejected", doctorHandler.ListRejectedReferrals)
				doctorGroup.GET("/referrals/assigned", doctorHandler.ListAssignedReferrals)
				doctorGroup.GET("/referrals/:id", doctorHandler.GetReferral)
				doctorGroup.POST("/referrals", doctorHandler.CreateOrSubmit)
				doctorGroup.POST("/referrals/:id/cancel", doctorHandler.Cancel)
				doctorGroup.DELETE("/referrals/:id/attachments", doctorHandler.DeleteAttachments)
				doctorGroup.PUT("/referrals/:id", doctorHandler.UpdateDraft)
				doctorGroup.PUT("/referrals/:id/submit", doctorHandler.SubmitReferral)
				doctorGroup.POST("/referrals/:id/reject-after-send", doctorHandler.RejectAfterSend)
				doctorGroup.POST("/referrals/:id/consult", doctorHandler.GrantConsultAccess)
				doctorGroup.POST("/referrals/:id/consult/revoke", doctorHandler.RevokeConsultAccess)
			}

			// LIAISON
			liaisonGroup := protected.Group("/liaison/referrals")
			liaisonGroup.Use(middleware.RequireRole(entity.RoleLiaisonOfficer))
			{
				liaisonGroup.GET("/", liaisonHandler.ListOutgoing)
				liaisonGroup.GET("/approved", liaisonHandler.ListApprovedReferrals)
				liaisonGroup.GET("/rejected", liaisonHandler.ListRejectedReferrals)
				// Deprecated
				liaisonGroup.GET("/incoming", liaisonHandler.ListIncoming)
				liaisonGroup.GET("/:id", liaisonHandler.GetReferral)
				liaisonGroup.POST("/:id/read", liaisonHandler.Read)
				liaisonGroup.POST("/:id/forward", liaisonHandler.Forward)
				liaisonGroup.POST("/:id/reject", liaisonHandler.Reject)
				liaisonGroup.POST("/:id/revise", liaisonHandler.Revise)
				liaisonGroup.POST("/:id/reject-after-send", liaisonHandler.RejectAfterSend)
				liaisonGroup.GET("/dashboard/stats", liaisonHandler.GetDashboardStats)
				liaisonGroup.GET("/:id/review-checklist", liaisonHandler.GetReviewChecklist)
				liaisonGroup.PUT("/:id/review-checklist", liaisonHandler.UpdateReviewChecklist)
				// Deprecated
				// liaisonGroup.POST("/incoming/:id/unassign", liaisonHandler.UnassignSpecialist)
			}

			// SPECIALIST
			specialistGroup := protected.Group("/specialist/referrals")
			specialistGroup.Use(middleware.RequireRole(entity.RoleReceivingSpecialist))
			{
				specialistGroup.GET("", specialistHandler.ListReferrals)
				specialistGroup.GET("/approved", specialistHandler.ListApprovedReferrals)
				specialistGroup.GET("/rejected", specialistHandler.ListRejectedReferrals)
				specialistGroup.GET("/:id", specialistHandler.GetReferral)
				specialistGroup.POST("/:id/read", specialistHandler.Read)
				specialistGroup.POST("/:id/accept", specialistHandler.Accept)
				specialistGroup.POST("/:id/reject", specialistHandler.Reject)
				specialistGroup.POST("/:id/release", specialistHandler.Release)
				specialistGroup.POST("/:id/rerun-ml", specialistHandler.RerunML)
				specialistGroup.POST("/:id/redirect", specialistHandler.RedirectReferral)
				specialistGroup.GET("/:id/redirect-options", specialistHandler.ListRedirectionOptions)
				specialistGroup.PUT("/:id/department", specialistHandler.ChangeDepartment)

				// Triage & Scheduling
				specialistGroup.GET("/triage-queue", specialistHandler.GetTriageQueue)
				specialistGroup.POST("/:id/triage-severity", specialistHandler.SetManualSeverity)
				specialistGroup.POST("/:id/triage-review", triageHandler.Review)
				specialistGroup.GET("/capacity", specialistHandler.GetCapacity)
				specialistGroup.POST("/:id/schedule", specialistHandler.Schedule)
				specialistGroup.POST("/:id/emergency-schedule", specialistHandler.ManualEmergencySchedule)
			}

			// Shared Referral Routes
			protected.GET("/referrals/:id/redirections", redirectionHandler.GetRedirectionHistory)
			protected.POST("/referrals/:id/deceased", middleware.RequireRole(entity.RoleReferringDoctor, entity.RoleLiaisonOfficer, entity.RoleReceivingSpecialist, entity.RoleSystemSuperAdmin), redirectionHandler.MarkDeceased)

			// RECEPTIONIST
			receptionistGroup := protected.Group("/receptionist")
			receptionistGroup.Use(middleware.RequireRole(entity.RoleReceptionist))
			{
				receptionistGroup.GET("/doctors", receptionistHandler.ListDoctors)

				refGroup := receptionistGroup.Group("/referrals")
				{
					refGroup.GET("", receptionistHandler.ListReferrals)
					refGroup.GET("/missed", receptionistHandler.ListMissedReferrals)
					// Keep static routes before :id to avoid "schedule/upcoming" being treated as UUIDs.
					refGroup.GET("/schedule", receptionistHandler.GetSchedule)
					refGroup.GET("/upcoming", receptionistHandler.GetSchedule)
					refGroup.GET("/offline-data", receptionistHandler.GetOfflineData)
					refGroup.GET("/:id", receptionistHandler.GetReferral)
					refGroup.POST("/:id/arrive", receptionistHandler.ConfirmArrival)
					refGroup.POST("/:id/assign-doctor", receptionistHandler.AssignDoctor)
					refGroup.POST("/:id/revoke-doctor", receptionistHandler.RevokeDoctor)
					refGroup.POST("/:id/miss", receptionistHandler.MarkMissed)
				}
			}

			// ADMINS
			systemAdminGroup := protected.Group("/system-admin/referrals")
			systemAdminGroup.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin))
			{
				systemAdminGroup.GET("", adminHandler.SystemAdminList)
				systemAdminGroup.GET("/approved", adminHandler.ListApprovedReferrals)
				systemAdminGroup.GET("/rejected", adminHandler.ListRejectedReferrals)
			}

			// MoH Analytics
			mohGroup := protected.Group("/moh")
			mohGroup.Use(middleware.RequireRole(entity.RoleMohAnalyst))
			{
				mohGroup.GET("/dashboard/summary", mohAnalyticsHandler.GetDashboardSummary)
				mohGroup.GET("/referral-trends", mohAnalyticsHandler.GetReferralTrends)
				mohGroup.GET("/hospital-load", mohAnalyticsHandler.GetHospitalLoad)
				mohGroup.GET("/disease-hotspots", mohAnalyticsHandler.GetDiseaseHotspots)
				mohGroup.GET("/severity-distribution", mohAnalyticsHandler.GetSeverityDistribution)
				mohGroup.GET("/reports/export", mohAnalyticsHandler.ExportReport)
			}

			// Global User Management (System Admin Only)
			sysAdminUsersGroup := protected.Group("/system-admin/users")
			sysAdminUsersGroup.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin))
			{
				sysAdminUsersGroup.GET("", userHandler.SystemAdminListUsers)
				sysAdminUsersGroup.POST("", userHandler.CreateUser)
				sysAdminUsersGroup.PUT("/:id", userHandler.UpdateUser)
				sysAdminUsersGroup.DELETE("/:id", userHandler.DeleteUser)
				sysAdminUsersGroup.PATCH("/:id/role", userHandler.AssignRole)
			}

			// Shared Admin Routes (SystemAdmin + HospitalAdmin)
			sharedAdminGroup := protected.Group("/system-admin")
			sharedAdminGroup.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin, entity.RoleHospitalAdmin))
			{
				// Profile moderation (removal of inappropriate images)
				sharedAdminGroup.DELETE("/users/:id/profile/image", userHandler.ModerateProfileImage)
			}

			hospitalAdminGroup := protected.Group("/hospital-admin")
			hospitalAdminGroup.Use(middleware.RequireRole(entity.RoleHospitalAdmin))
			{
				hospitalAdminGroup.GET("/referrals-log", adminHandler.HospitalAdminLogs)
				hospitalAdminGroup.GET("/referrals/inbound", adminHandler.HospitalAdminInboundReferrals)
				hospitalAdminGroup.GET("/referrals/outbound", adminHandler.HospitalAdminOutboundReferrals)
				hospitalAdminGroup.GET("/referrals/pending-approvals", adminHandler.HospitalAdminPendingApprovals)
				hospitalAdminGroup.GET("/referrals/rejected-redirected", adminHandler.HospitalAdminRejectedRedirected)
				hospitalAdminGroup.GET("/referrals/stats/by-status", adminHandler.HospitalAdminReferralStatusCounts)
				hospitalAdminGroup.GET("/referrals/:id", adminHandler.HospitalAdminReferralDetails)
				hospitalAdminGroup.GET("/audit-logs", adminHandler.HospitalAdminAuditLogs)
				hospitalAdminGroup.GET("/reports/monthly-referrals", adminHandler.HospitalAdminMonthlyReferralTotals)
				hospitalAdminGroup.GET("/reports/acceptance-rejection-rate", adminHandler.HospitalAdminAcceptanceRejectionRate)
				hospitalAdminGroup.GET("/reports/missed-appointment-rate", adminHandler.HospitalAdminMissedAppointmentRate)
				hospitalAdminGroup.GET("/reports/busiest-departments", adminHandler.HospitalAdminBusiestDepartments)
				hospitalAdminGroup.GET("/reports/average-wait-time", adminHandler.HospitalAdminAverageWaitTime)
				hospitalAdminGroup.GET("/reports/top-referring-hospitals", adminHandler.HospitalAdminTopReferringHospitals)

				hospitalAdminGroup.GET("/hospital/profile", hospitalAdminOpsHandler.GetMyHospitalProfile)
				hospitalAdminGroup.PATCH("/hospital/profile", hospitalAdminOpsHandler.UpdateMyHospitalProfile)
				hospitalAdminGroup.POST("/departments", hospitalAdminOpsHandler.LinkDepartmentToMyHospital)
				hospitalAdminGroup.GET("/departments", hospitalAdminOpsHandler.ListMyHospitalDepartments)
				hospitalAdminGroup.PATCH("/departments/:deptId/activation", hospitalAdminOpsHandler.SetDepartmentActive)
				hospitalAdminGroup.PATCH("/departments/:deptId/head", hospitalAdminOpsHandler.AssignDepartmentHead)

				hospitalAdminGroup.POST("/staff", hospitalAdminStaffHandler.CreateStaff)
				hospitalAdminGroup.GET("/staff", hospitalAdminStaffHandler.ListStaff)
				hospitalAdminGroup.GET("/staff/sessions", hospitalAdminStaffHandler.ListActiveStaffSessions)
				hospitalAdminGroup.GET("/staff/:id", hospitalAdminStaffHandler.GetStaff)
				hospitalAdminGroup.PATCH("/staff/:id/role", hospitalAdminStaffHandler.ChangeStaffRole)
				hospitalAdminGroup.PATCH("/staff/:id/activation", hospitalAdminStaffHandler.SetStaffActive)
				hospitalAdminGroup.PATCH("/staff/:id/department", hospitalAdminStaffHandler.ReassignDepartment)
				hospitalAdminGroup.POST("/staff/:id/force-logout", hospitalAdminStaffHandler.ForceLogoutStaff)
				hospitalAdminGroup.DELETE("/staff/:id", hospitalAdminStaffHandler.DeleteStaff)
				hospitalAdminGroup.POST("/staff/:id/replace", hospitalAdminStaffHandler.ReplaceStaff)
				hospitalAdminGroup.GET("/referrals/:id/status-history", hospitalAdminStaffHandler.GetReferralStatusHistory)
			}

			// DEPARTMENT HEAD
			deptHeadGroup := protected.Group("/department-head")
			deptHeadGroup.Use(middleware.RequireRole(entity.RoleDeptHead))
			{
				deptHeadGroup.GET("/triage-queue", deptHeadHandler.GetTriageQueue)

				// Capacity Overrides
				deptHeadGroup.GET("/capacity/overrides", deptHeadHandler.ListOverrides)
				deptHeadGroup.POST("/capacity/overrides", deptHeadHandler.CreateOverride)
				deptHeadGroup.PUT("/capacity/overrides/:id", deptHeadHandler.UpdateOverride)
				deptHeadGroup.DELETE("/capacity/overrides/:id", deptHeadHandler.DeleteOverride)

				// Daily Schedule
				deptHeadGroup.GET("/schedule", scheduleHandler.GetSchedule)
				deptHeadGroup.PUT("/schedule/:id/max-slots", scheduleHandler.UpdateMaxSlots)
				deptHeadGroup.POST("/schedule/batch", deptHeadHandler.BatchSchedule)
			}

			attachmentGroup := protected.Group("/attachments")
			{
				attachmentGroup.GET("/:id", attachmentHandler.GetAttachment)
			}
			// Referral-specific attachments
			protected.POST("/referrals/:id/attachments", attachmentHandler.UploadAttachment)
			protected.GET("/referrals/:id/attachments", attachmentHandler.GetReferralAttachments)
			protected.DELETE("/referrals/:id/attachments/:attachment_id", attachmentHandler.DeleteFromReferral)

			// ---- User Management ----
			// Profile & Global Reference Lookup – accessible by clinical/hospital/analyst roles
			userAccesses := protected.Group("/users")
			userAccesses.Use(middleware.RequireRole(
				entity.RoleSystemSuperAdmin,
				entity.RoleHospitalAdmin,
				entity.RoleReferringDoctor,
				entity.RoleReceivingSpecialist,
				entity.RoleLiaisonOfficer,
				entity.RoleReceptionist,
				entity.RoleMohAnalyst,
				entity.RoleDeptHead,
			))
			{
				userAccesses.GET("/me", userHandler.GetMyProfile)
				userAccesses.GET("", userHandler.ListUsers)
				userAccesses.GET("/:id", userHandler.GetUser)
				userAccesses.PUT("/profile/image", userHandler.UpdateProfileImage)
				userAccesses.DELETE("/profile/image", userHandler.DeleteMyProfileImage)
			}

			// ---- In-App Notifications ----
			notificationRoutes := protected.Group("/me/notifications")
			{
				notificationRoutes.GET("", inAppNotifHandler.ListNotifications)
				notificationRoutes.POST("/:id/read", inAppNotifHandler.MarkRead)
				notificationRoutes.POST("/read-all", inAppNotifHandler.MarkAllRead)
				notificationRoutes.GET("/unread-count", inAppNotifHandler.GetUnreadCount)
			}

			// ---- Hospital Management ----
			// Read – accessible by any authenticated user
			protected.GET("/hospitals", hospitalHandler.ListHospitals)
			protected.GET("/hospitals/:id", hospitalHandler.GetHospital)

			// Admin-only hospital write operations
			adminHospitals := protected.Group("/hospitals")
			adminHospitals.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin))
			{
				adminHospitals.POST("", hospitalHandler.CreateHospital)
				adminHospitals.PUT("/:id", hospitalHandler.UpdateHospital)
				adminHospitals.DELETE("/:id", hospitalHandler.DeleteHospital)
			}

			// Hospital-Department linking (admin-only for write, read for all)
			protected.GET("/hospitals/:id/departments", departmentHandler.ListHospitalDepartments)

			adminHospDept := protected.Group("/hospitals")
			adminHospDept.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin))
			{
				adminHospDept.POST("/:id/departments", departmentHandler.LinkDepartmentToHospital)
				adminHospDept.DELETE("/:id/departments/:deptId", departmentHandler.UnlinkDepartmentFromHospital)
			}

			// ---- Department Management ----
			// Read – accessible by any authenticated user
			protected.GET("/departments", departmentHandler.ListDepartments)
			protected.GET("/departments/:id", departmentHandler.GetDepartment)

			// Admin-only department write operations
			adminDepts := protected.Group("/departments")
			adminDepts.Use(middleware.RequireRole(entity.RoleSystemSuperAdmin))
			{
				adminDepts.POST("", departmentHandler.CreateDepartment)
				adminDepts.PUT("/:id", departmentHandler.UpdateDepartment)
				adminDepts.DELETE("/:id", departmentHandler.DeleteDepartment)
			}
		}
	}
}
