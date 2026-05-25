package test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) FindByID(ctx context.Context, id interface{}) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepo) FindAll(ctx context.Context) ([]entity.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.User), args.Error(1)
}

func (m *MockUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepo) FindByNationalID(ctx context.Context, nationalID string) (*entity.User, error) {
	args := m.Called(ctx, nationalID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepo) Update(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepo) ListUsers(ctx context.Context, filter irepository.UserListFilter) ([]entity.User, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepo) CreateStaffReplacementLog(ctx context.Context, log *entity.StaffReplacementLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockUserRepo) CountHospitalStaffByStatus(ctx context.Context, hospitalID uuid.UUID) (total int64, active int64, inactive int64, err error) {
	args := m.Called(ctx, hospitalID)
	return args.Get(0).(int64), args.Get(1).(int64), args.Get(2).(int64), args.Error(3)
}

func (m *MockUserRepo) FindDepartmentHeadsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, hospitalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.User), args.Error(1)
}


// ---------------------------------------------------------------------------
// Mock: ReferralRepository
// ---------------------------------------------------------------------------

type MockReferralRepo struct {
	mock.Mock
}

func (m *MockReferralRepo) CreateReferralTransaction(ctx context.Context, referral *entity.Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *MockReferralRepo) UpdateReferralTransaction(ctx context.Context, referral *entity.Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *MockReferralRepo) DeleteReferral(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReferralRepo) GetReferralByID(ctx context.Context, id uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReferralRepo) ListReferrals(ctx context.Context, filter map[string]interface{}) ([]entity.Referral, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Referral), args.Error(1)
}

func (m *MockReferralRepo) Create(ctx context.Context, referral *entity.Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *MockReferralRepo) Update(ctx context.Context, referral *entity.Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *MockReferralRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReferralRepo) FindByID(ctx context.Context, id interface{}) (*entity.Referral, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralRepo) FindAll(ctx context.Context) ([]entity.Referral, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Referral), args.Error(1)
}

func (m *MockReferralRepo) CreateStatusHistory(ctx context.Context, history *entity.ReferralStatusHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockReferralRepo) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListInboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListOutboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListByStatusesForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter, statuses []entity.ReferralStatus) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter, statuses)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) GetDetailsForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, hospID, referralID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockReferralRepo) CountByStatusForHospitalAdmin(ctx context.Context, hospID uuid.UUID) ([]irepository.ReferralStatusCount, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).([]irepository.ReferralStatusCount), args.Error(1)
}

func (m *MockReferralRepo) GetReferralStatusCounts(ctx context.Context, hospID uuid.UUID) ([]irepository.ReferralStatusCount, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).([]irepository.ReferralStatusCount), args.Error(1)
}

func (m *MockReferralRepo) UpdateTargetDeptAndStatus(ctx context.Context, referralID, targetHospID, targetDeptID uuid.UUID, status entity.ReferralStatus) error {
	args := m.Called(ctx, referralID, targetHospID, targetDeptID, status)
	return args.Error(0)
}

func (m *MockReferralRepo) GetMonthlyReferralTotalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, months int) ([]irepository.MonthlyReferralTotal, error) {
	args := m.Called(ctx, hospID, months)
	return args.Get(0).([]irepository.MonthlyReferralTotal), args.Error(1)
}

func (m *MockReferralRepo) GetAcceptanceRejectionRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, float64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *MockReferralRepo) GetMissedAppointmentRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockReferralRepo) GetBusiestDepartmentsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.DepartmentReferralLoad, error) {
	args := m.Called(ctx, hospID, limit)
	return args.Get(0).([]irepository.DepartmentReferralLoad), args.Error(1)
}

func (m *MockReferralRepo) GetAverageWaitTimeForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockReferralRepo) GetTopReferringHospitalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.ReferringHospitalCount, error) {
	args := m.Called(ctx, hospID, limit)
	return args.Get(0).([]irepository.ReferringHospitalCount), args.Error(1)
}

func (m *MockReferralRepo) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, limit, page)
	return args.Get(0).([]entity.ReferralStatusHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) GetReferralStatusHistoryForHospital(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, referralID, limit, page)
	return args.Get(0).([]entity.ReferralStatusHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, doctorID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) ListForSpecialist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) CountBySenderHospitalAndStatuses(ctx context.Context, hospID uuid.UUID, statuses []entity.ReferralStatus, excludeDraft bool, startDate, endDate *time.Time) (int64, error) {
	args := m.Called(ctx, hospID, statuses, excludeDraft, startDate, endDate)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReferralRepo) CountAcceptedOrCompletedToday(ctx context.Context, hospID uuid.UUID) (int64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReferralRepo) CountByTargetDeptAndStatuses(ctx context.Context, hospID, deptID uuid.UUID, statuses []entity.ReferralStatus, startDate, endDate *time.Time) ([]irepository.ReferralStatusCount, error) {
	args := m.Called(ctx, hospID, deptID, statuses, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.ReferralStatusCount), args.Error(1)
}

func (m *MockReferralRepo) GetMohDashboardSummary(ctx context.Context, filter irepository.MohAnalyticsFilter) (*irepository.MohDashboardSummary, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*irepository.MohDashboardSummary), args.Error(1)
}

func (m *MockReferralRepo) GetMohReferralTrends(ctx context.Context, filter irepository.MohAnalyticsFilter, granularity string) ([]irepository.MohReferralTrendPoint, error) {
	args := m.Called(ctx, filter, granularity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohReferralTrendPoint), args.Error(1)
}

func (m *MockReferralRepo) GetMohHospitalLoad(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohHospitalLoadMetric, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohHospitalLoadMetric), args.Error(1)
}

func (m *MockReferralRepo) GetMohDiseaseHotspots(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohDiseaseHotspot, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohDiseaseHotspot), args.Error(1)
}

func (m *MockReferralRepo) GetMohSeverityDistribution(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohSeverityDistribution, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohSeverityDistribution), args.Error(1)
}

func (m *MockReferralRepo) ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) GetDoctorStats(ctx context.Context, doctorID uuid.UUID) (total, pending, accepted, critical int64, err error) {
	args := m.Called(ctx, doctorID)
	return args.Get(0).(int64), args.Get(1).(int64), args.Get(2).(int64), args.Get(3).(int64), args.Error(4)
}

func (m *MockReferralRepo) GetLatestPendingForDoctor(ctx context.Context, doctorID uuid.UUID, limit int) ([]entity.Referral, error) {
	args := m.Called(ctx, doctorID, limit)
	return args.Get(0).([]entity.Referral), args.Error(1)
}

func (m *MockReferralRepo) UpdateFields(ctx context.Context, referralID uuid.UUID, updates irepository.ReferralUpdateFields) error {
	args := m.Called(ctx, referralID, updates)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Mock: ReferralUseCase
// ---------------------------------------------------------------------------

type MockReferralUseCase struct {
	mock.Mock
}

func (m *MockReferralUseCase) CreateDraftOrSubmit(ctx context.Context, doctorID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*dto.ReferralCreationResponse, error) {
	args := m.Called(ctx, doctorID, senderHospitalID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ReferralCreationResponse), args.Error(1)
}

func (m *MockReferralUseCase) CreateReferralWithAttachments(ctx context.Context, doctorID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest, refID uuid.UUID, uploads []iusecase.UploadedFileData) (*dto.ReferralCreationResponse, error) {
	args := m.Called(ctx, doctorID, senderHospitalID, req, refID, uploads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ReferralCreationResponse), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: NetworkRepository
// ---------------------------------------------------------------------------

type MockNetworkRepo struct {
	mock.Mock
}

func (m *MockNetworkRepo) CreateNetworkRoute(ctx context.Context, route *entity.ReferralNetwork) error {
	args := m.Called(ctx, route)
	return args.Error(0)
}

func (m *MockNetworkRepo) ListNetworkRoutes(ctx context.Context, senderID *uuid.UUID) ([]entity.ReferralNetwork, error) {
	args := m.Called(ctx, senderID)
	return args.Get(0).([]entity.ReferralNetwork), args.Error(1)
}

func (m *MockNetworkRepo) VerifyNetworkPathway(ctx context.Context, senderID, targetID uuid.UUID) (bool, error) {
	args := m.Called(ctx, senderID, targetID)
	return args.Bool(0), args.Error(1)
}

func (m *MockNetworkRepo) DeleteNetworkRoute(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNetworkRepo) GetOutgoingNetworkHospitals(ctx context.Context, senderID uuid.UUID) ([]entity.Hospital, error) {
	args := m.Called(ctx, senderID)
	return args.Get(0).([]entity.Hospital), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: ReferralRedirectionRepository
// ---------------------------------------------------------------------------

type MockReferralRedirectionRepo struct {
	mock.Mock
}

func (m *MockReferralRedirectionRepo) Create(ctx context.Context, r *entity.ReferralRedirection) error {
	return m.Called(ctx, r).Error(0)
}

func (m *MockReferralRedirectionRepo) FindByID(ctx context.Context, id interface{}) (*entity.ReferralRedirection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ReferralRedirection), args.Error(1)
}

func (m *MockReferralRedirectionRepo) FindAll(ctx context.Context) ([]entity.ReferralRedirection, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.ReferralRedirection), args.Error(1)
}

func (m *MockReferralRedirectionRepo) Update(ctx context.Context, r *entity.ReferralRedirection) error {
	return m.Called(ctx, r).Error(0)
}

func (m *MockReferralRedirectionRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockReferralRedirectionRepo) ListByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.ReferralRedirection, error) {
	args := m.Called(ctx, referralID)
	return args.Get(0).([]entity.ReferralRedirection), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: DepartmentRepository (minimal - for referral usecase injection)
// ---------------------------------------------------------------------------

type MockDepartmentRepo struct {
	mock.Mock
}

func (m *MockDepartmentRepo) Create(ctx context.Context, d *entity.Department) error {
	return m.Called(ctx, d).Error(0)
}

func (m *MockDepartmentRepo) FindByID(ctx context.Context, id interface{}) (*entity.Department, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Department), args.Error(1)
}

func (m *MockDepartmentRepo) FindAll(ctx context.Context) ([]entity.Department, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Department), args.Error(1)
}

func (m *MockDepartmentRepo) Update(ctx context.Context, d *entity.Department) error {
	return m.Called(ctx, d).Error(0)
}

func (m *MockDepartmentRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockDepartmentRepo) ListDepartments(ctx context.Context, filter irepository.DepartmentListFilter) ([]entity.Department, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Department), args.Get(1).(int64), args.Error(2)
}

func (m *MockDepartmentRepo) LinkToHospital(ctx context.Context, link *entity.HospitalDepartment) error {
	return m.Called(ctx, link).Error(0)
}

func (m *MockDepartmentRepo) UnlinkFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error {
	return m.Called(ctx, hospitalID, departmentID).Error(0)
}

func (m *MockDepartmentRepo) ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error) {
	args := m.Called(ctx, hospitalID)
	return args.Get(0).([]entity.HospitalDepartment), args.Error(1)
}

func (m *MockDepartmentRepo) FindHospitalDepartment(ctx context.Context, hospitalID, departmentID uuid.UUID) (*entity.HospitalDepartment, error) {
	args := m.Called(ctx, hospitalID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.HospitalDepartment), args.Error(1)
}

func (m *MockDepartmentRepo) FindHospitalDepartmentForHospital(ctx context.Context, hospitalID, departmentOrLinkID uuid.UUID) (*entity.HospitalDepartment, error) {
	args := m.Called(ctx, hospitalID, departmentOrLinkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.HospitalDepartment), args.Error(1)
}

func (m *MockDepartmentRepo) UpdateHospitalDepartment(ctx context.Context, link *entity.HospitalDepartment) error {
	return m.Called(ctx, link).Error(0)
}

func (m *MockDepartmentRepo) UpdateStaffCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, value int) error {
	return m.Called(ctx, hospitalID, deptID, value).Error(0)
}

func (m *MockDepartmentRepo) UpdateDailyCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, standardDailyLimit, overbookLimit int) error {
	return m.Called(ctx, hospitalID, deptID, standardDailyLimit, overbookLimit).Error(0)
}

func (m *MockReferralUseCase) UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.UpdateReferralRequest, submit bool) (*dto.ReferralCreationResponse, error) {
	args := m.Called(ctx, id, doctorID, req, submit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ReferralCreationResponse), args.Error(1)
}

func (m *MockReferralUseCase) ListForDoctor(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, doctorID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, doctorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) CancelReferral(ctx context.Context, id, doctorID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, doctorID, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) GetDoctorDashboardStats(ctx context.Context, doctorID uuid.UUID) (*dto.DoctorDashboardStats, error) {
	args := m.Called(ctx, doctorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.DoctorDashboardStats), args.Error(1)
}

func (m *MockReferralUseCase) GetLatestPendingReferrals(ctx context.Context, doctorID uuid.UUID, limit int) ([]dto.ListReferralResponse, error) {
	args := m.Called(ctx, doctorID, limit)
	return args.Get(0).([]dto.ListReferralResponse), args.Error(1)
}

func (m *MockReferralUseCase) ListAssignedReferrals(ctx context.Context, doctorID uuid.UUID, filter irepository.ReferralFilter, accessType string, includeRevoked bool) ([]entity.Referral, []entity.ReferralAccess, int64, error) {
	args := m.Called(ctx, doctorID, filter, accessType, includeRevoked)
	return args.Get(0).([]entity.Referral), args.Get(1).([]entity.ReferralAccess), args.Get(2).(int64), args.Error(3)
}

func (m *MockReferralUseCase) DeleteAttachmentsByReferralID(ctx context.Context, id, doctorID uuid.UUID) error {
	args := m.Called(ctx, id, doctorID)
	return args.Error(0)
}

func (m *MockReferralUseCase) ListOutgoingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) ListIncomingForLiaison(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetDetailsForLiaison(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, hospID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) LiaisonRead(ctx context.Context, id, liaisonID, hospID uuid.UUID) error {
	args := m.Called(ctx, id, liaisonID, hospID)
	return args.Error(0)
}

func (m *MockReferralUseCase) LiaisonForward(ctx context.Context, id, liaisonID, hospID uuid.UUID, comment string) error {
	args := m.Called(ctx, id, liaisonID, hospID, comment)
	return args.Error(0)
}

func (m *MockReferralUseCase) LiaisonReject(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, liaisonID, hospID, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) LiaisonRevise(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, liaisonID, hospID, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) RejectAfterSend(ctx context.Context, referralID, userID, hospID uuid.UUID, role entity.UserRole, reason string) error {
	args := m.Called(ctx, referralID, userID, hospID, role, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) LiaisonUnassignSpecialist(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, liaisonID, hospID, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) GetLiaisonDashboardStats(ctx context.Context, hospID uuid.UUID) (*iusecase.LiaisonDashboardStats, error) {
	args := m.Called(ctx, hospID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*iusecase.LiaisonDashboardStats), args.Error(1)
}

func (m *MockReferralUseCase) UpdateReviewChecklist(ctx context.Context, referralID, liaisonID, hospID uuid.UUID, req dto.ReviewChecklistRequest) error {
	args := m.Called(ctx, referralID, liaisonID, hospID, req)
	return args.Error(0)
}

func (m *MockReferralUseCase) GetReviewChecklist(ctx context.Context, referralID, hospID uuid.UUID) (*dto.ReviewChecklistResponse, error) {
	args := m.Called(ctx, referralID, hospID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ReviewChecklistResponse), args.Error(1)
}

func (m *MockReferralUseCase) ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, specialistID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, hospID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) SpecialistRead(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	args := m.Called(ctx, id, specialistID, hospID)
	return args.Error(0)
}

func (m *MockReferralUseCase) SpecialistAccept(ctx context.Context, id, specialistID, hospID uuid.UUID, severityScore *float64) error {
	args := m.Called(ctx, id, specialistID, hospID, severityScore)
	return args.Error(0)
}

func (m *MockReferralUseCase) SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, specialistID, hospID, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) SpecialistRelease(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, specialistID, hospID, reason)
	return args.Error(0)
}

func (m *MockReferralUseCase) SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	args := m.Called(ctx, id, specialistID, hospID)
	return args.Error(0)
}

func (m *MockReferralUseCase) GetMLPredictionForSpecialist(ctx context.Context, referralID, hospID uuid.UUID) (*entity.MLPrediction, error) {
	args := m.Called(ctx, referralID, hospID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.MLPrediction), args.Error(1)
}

func (m *MockReferralUseCase) RedirectReferral(ctx context.Context, id, specialistID, hospID, targetHospitalID uuid.UUID, reason string, newDeptID *uuid.UUID) error {
	args := m.Called(ctx, id, specialistID, hospID, targetHospitalID, reason, newDeptID)
	return args.Error(0)
}

func (m *MockReferralUseCase) ListRedirectionOptions(ctx context.Context, id, specialistID, hospID uuid.UUID, filterDeptID *uuid.UUID) ([]entity.Hospital, error) {
	args := m.Called(ctx, id, specialistID, hospID, filterDeptID)
	return args.Get(0).([]entity.Hospital), args.Error(1)
}

func (m *MockReferralUseCase) GetRedirectionHistory(ctx context.Context, referralID, userID uuid.UUID, role string, hospID uuid.UUID) ([]entity.ReferralRedirection, error) {
	args := m.Called(ctx, referralID, userID, role, hospID)
	return args.Get(0).([]entity.ReferralRedirection), args.Error(1)
}

func (m *MockReferralUseCase) ChangeDepartment(ctx context.Context, referralID, specialistID, hospID, newDeptID uuid.UUID) error {
	args := m.Called(ctx, referralID, specialistID, hospID, newDeptID)
	return args.Error(0)
}

func (m *MockReferralUseCase) ListForReceptionist(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, hospID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Referral), args.Error(1)
}


func (m *MockReferralUseCase) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) ListInboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) ListOutboundForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) ListPendingApprovalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) ListRejectedRedirectedForHospitalAdmin(ctx context.Context, hospID uuid.UUID, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetDetailsForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, hospID, referralID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) GetReferralStatusCountsForHospitalAdmin(ctx context.Context, hospID uuid.UUID) ([]irepository.ReferralStatusCount, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).([]irepository.ReferralStatusCount), args.Error(1)
}

func (m *MockReferralUseCase) GetMonthlyReferralTotalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, months int) ([]irepository.MonthlyReferralTotal, error) {
	args := m.Called(ctx, hospID, months)
	return args.Get(0).([]irepository.MonthlyReferralTotal), args.Error(1)
}

func (m *MockReferralUseCase) GetAcceptanceRejectionRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, float64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *MockReferralUseCase) GetMissedAppointmentRateForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockReferralUseCase) GetBusiestDepartmentsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.DepartmentReferralLoad, error) {
	args := m.Called(ctx, hospID, limit)
	return args.Get(0).([]irepository.DepartmentReferralLoad), args.Error(1)
}

func (m *MockReferralUseCase) GetAverageWaitTimeForHospitalAdmin(ctx context.Context, hospID uuid.UUID) (float64, error) {
	args := m.Called(ctx, hospID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockReferralUseCase) GetTopReferringHospitalsForHospitalAdmin(ctx context.Context, hospID uuid.UUID, limit int) ([]irepository.ReferringHospitalCount, error) {
	args := m.Called(ctx, hospID, limit)
	return args.Get(0).([]irepository.ReferringHospitalCount), args.Error(1)
}

func (m *MockReferralUseCase) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, limit, page)
	return args.Get(0).([]entity.ReferralStatusHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetReferralStatusHistoryForHospitalAdmin(ctx context.Context, hospID, referralID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, referralID, limit, page)
	return args.Get(0).([]entity.ReferralStatusHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetMohDashboardSummary(ctx context.Context, filter irepository.MohAnalyticsFilter) (*irepository.MohDashboardSummary, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*irepository.MohDashboardSummary), args.Error(1)
}

func (m *MockReferralUseCase) GetMohReferralTrends(ctx context.Context, filter irepository.MohAnalyticsFilter, granularity string) ([]irepository.MohReferralTrendPoint, error) {
	args := m.Called(ctx, filter, granularity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohReferralTrendPoint), args.Error(1)
}

func (m *MockReferralUseCase) GetMohHospitalLoad(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohHospitalLoadMetric, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohHospitalLoadMetric), args.Error(1)
}

func (m *MockReferralUseCase) GetMohDiseaseHotspots(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohDiseaseHotspot, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohDiseaseHotspot), args.Error(1)
}

func (m *MockReferralUseCase) GetMohSeverityDistribution(ctx context.Context, filter irepository.MohAnalyticsFilter) ([]irepository.MohSeverityDistribution, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]irepository.MohSeverityDistribution), args.Error(1)
}

func (m *MockReferralUseCase) IsValidStatus(status string) bool {
	args := m.Called(status)
	return args.Bool(0)
}

func (m *MockReferralUseCase) MarkDeceased(ctx context.Context, referralID, userID uuid.UUID, role entity.UserRole, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, referralID, userID, role, hospID, reason)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Mock: AttachmentRepository
// ---------------------------------------------------------------------------

type MockAttachmentRepo struct {
	mock.Mock
}

func (m *MockAttachmentRepo) Create(ctx context.Context, att *entity.Attachment) error {
	args := m.Called(ctx, att)
	return args.Error(0)
}

func (m *MockAttachmentRepo) FindByID(ctx context.Context, id interface{}) (*entity.Attachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Attachment), args.Error(1)
}

func (m *MockAttachmentRepo) Update(ctx context.Context, att *entity.Attachment) error {
	args := m.Called(ctx, att)
	return args.Error(0)
}

func (m *MockAttachmentRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAttachmentRepo) List(ctx context.Context) ([]entity.Attachment, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

func (m *MockAttachmentRepo) CountByReferralID(ctx context.Context, referralID uuid.UUID) (int64, error) {
	args := m.Called(ctx, referralID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockAttachmentRepo) GetByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error) {
	args := m.Called(ctx, referralID)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

func (m *MockAttachmentRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAttachmentRepo) FindByPublicID(ctx context.Context, publicID string) (*entity.Attachment, error) {
	args := m.Called(ctx, publicID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Attachment), args.Error(1)
}

func (m *MockAttachmentRepo) GetPendingAttachments(ctx context.Context) ([]entity.Attachment, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

func (m *MockAttachmentRepo) GetPendingAttachmentsBatch(ctx context.Context, limit int) ([]entity.Attachment, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

func (m *MockAttachmentRepo) UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status string, metadata map[string]interface{}, storagePath string, publicID string, rejectionReason string, rejectedAt *time.Time) error {
	args := m.Called(ctx, id, status, metadata, storagePath, publicID, rejectionReason, rejectedAt)
	return args.Error(0)
}

func (m *MockAttachmentRepo) CountByPublicIDPrefix(ctx context.Context, prefix string) (int64, error) {
	args := m.Called(ctx, prefix)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockAttachmentRepo) FindAll(ctx context.Context) ([]entity.Attachment, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: AttachmentUseCase
// ---------------------------------------------------------------------------

type MockAttachmentUseCase struct {
	mock.Mock
}

func (m *MockAttachmentUseCase) PrepareAttachmentEntity(referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) *entity.Attachment {
	args := m.Called(referralID, fileName, fileType, storagePath, publicID, category, fileSize)
	return args.Get(0).(*entity.Attachment)
}

func (m *MockAttachmentUseCase) GetAttachment(ctx context.Context, id uuid.UUID) (*entity.Attachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Attachment), args.Error(1)
}

func (m *MockAttachmentUseCase) GetAttachmentsByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.Attachment, error) {
	args := m.Called(ctx, referralID)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

func (m *MockAttachmentUseCase) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAttachmentUseCase) DeleteAttachmentFromReferral(ctx context.Context, referralID, attachmentID, doctorID uuid.UUID) error {
	args := m.Called(ctx, referralID, attachmentID, doctorID)
	return args.Error(0)
}

func (m *MockAttachmentUseCase) UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, fileBytes []byte, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error) {
	args := m.Called(ctx, referralID, doctorID, fileBytes, fileName, fileType, category, fileSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Attachment), args.Error(1)
}

func (m *MockAttachmentUseCase) VerifyAttachmentBytes(att *entity.Attachment, data []byte) (status string, metadata map[string]interface{}, reason string) {
	args := m.Called(att, data)
	return args.String(0), args.Get(1).(map[string]interface{}), args.String(2)
}

func (m *MockAttachmentUseCase) Storage() iinfra.StorageService {
	args := m.Called()
	return args.Get(0).(iinfra.StorageService)
}

// ---------------------------------------------------------------------------
// Mock: StorageService
// ---------------------------------------------------------------------------

type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) DeleteFile(ctx context.Context, publicID string) error {
	args := m.Called(ctx, publicID)
	return args.Error(0)
}

func (m *MockStorageService) GenerateUploadSignature(params map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(params)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockStorageService) RenameFile(ctx context.Context, publicID, newPublicID string) (string, string, error) {
	args := m.Called(ctx, publicID, newPublicID)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockStorageService) DeleteFilesByPrefix(ctx context.Context, prefix string) error {
	args := m.Called(ctx, prefix)
	return args.Error(0)
}

func (m *MockStorageService) UploadFile(ctx context.Context, file interface{}, folder string) (string, string, error) {
	args := m.Called(ctx, file, folder)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockStorageService) UploadBytes(ctx context.Context, data []byte, folder, fileName string) (string, string, error) {
	args := m.Called(ctx, data, folder, fileName)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockStorageService) ListFolders(ctx context.Context, prefix string) ([]string, error) {
	args := m.Called(ctx, prefix)
	return args.Get(0).([]string), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: TriageQueueRepository
// ---------------------------------------------------------------------------

type MockTriageQueueRepo struct {
	mock.Mock
}

func (m *MockTriageQueueRepo) Create(ctx context.Context, queue *entity.TriageQueue) error {
	args := m.Called(ctx, queue)
	return args.Error(0)
}

func (m *MockTriageQueueRepo) GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error) {
	args := m.Called(ctx, referralID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.TriageQueue), args.Error(1)
}


func (m *MockTriageQueueRepo) Update(ctx context.Context, queue *entity.TriageQueue) error {
	args := m.Called(ctx, queue)
	return args.Error(0)
}

func (m *MockTriageQueueRepo) DeleteByReferralID(ctx context.Context, referralID uuid.UUID) error {
	args := m.Called(ctx, referralID)
	return args.Error(0)
}

func (m *MockTriageQueueRepo) ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]entity.TriageQueue, int64, error) {
	args := m.Called(ctx, hospitalID, limit, offset)
	return args.Get(0).([]entity.TriageQueue), args.Get(1).(int64), args.Error(2)
}

func (m *MockTriageQueueRepo) ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID, start, end)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) GetWaitingByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) FindWaitingByHospitalAndDept(ctx context.Context, hospitalID, departmentID uuid.UUID) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, departmentID)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) FindActiveByHospitalAndDept(ctx context.Context, hospitalID, departmentID uuid.UUID) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, departmentID)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) FindScheduledByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]*entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID, startDate, endDate)
	return args.Get(0).([]*entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) FindByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error) {
	args := m.Called(ctx, hospitalID, deptID, limit, offset)
	return args.Get(0).([]*entity.TriageQueue), args.Get(1).(int64), args.Error(2)
}

func (m *MockTriageQueueRepo) FindAppointmentsForReminders(ctx context.Context, date time.Time) ([]*entity.TriageQueue, error) {
	args := m.Called(ctx, date)
	return args.Get(0).([]*entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) IncrementWaitingWeights(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTriageQueueRepo) FindAll(ctx context.Context) ([]entity.TriageQueue, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) FindByID(ctx context.Context, id interface{}) (*entity.TriageQueue, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTriageQueueRepo) ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error) {
	args := m.Called(ctx, hospitalID, limit, offset)
	return args.Get(0).([]*entity.TriageQueue), args.Get(1).(int64), args.Error(2)
}

func (m *MockTriageQueueRepo) FindMissedByDate(ctx context.Context, beforeDate time.Time) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, beforeDate)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) CountByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int64, error) {
	args := m.Called(ctx, hospitalID, deptID, date)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTriageQueueRepo) CountAssignedDoctorsByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int64, error) {
	args := m.Called(ctx, hospitalID, deptID, date)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTriageQueueRepo) FindScheduledByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID, date)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageQueueRepo) CountMissedByDeptInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) (int64, error) {
	args := m.Called(ctx, hospitalID, deptID, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTriageQueueRepo) CountScheduledByDeptInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) (int64, error) {
	args := m.Called(ctx, hospitalID, deptID, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTriageQueueRepo) OldestWaitingDaysByDept(ctx context.Context, hospitalID, deptID uuid.UUID) (int, error) {
	args := m.Called(ctx, hospitalID, deptID)
	return args.Int(0), args.Error(1)
}

func (m *MockTriageQueueRepo) ListTriageQueueFiltered(ctx context.Context, filter irepository.TriageQueueFilter) ([]entity.TriageQueue, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.TriageQueue), args.Get(1).(int64), args.Error(2)
}


// ---------------------------------------------------------------------------
// Mock: ClinicalUpdateRepository
// ---------------------------------------------------------------------------

type MockClinicalUpdateRepo struct {
	mock.Mock
}

func (m *MockClinicalUpdateRepo) Create(ctx context.Context, cu *entity.ClinicalUpdate) error {
	return m.Called(ctx, cu).Error(0)
}

func (m *MockClinicalUpdateRepo) FindByID(ctx context.Context, id interface{}) (*entity.ClinicalUpdate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ClinicalUpdate), args.Error(1)
}

func (m *MockClinicalUpdateRepo) FindAll(ctx context.Context) ([]entity.ClinicalUpdate, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.ClinicalUpdate), args.Error(1)
}

func (m *MockClinicalUpdateRepo) Update(ctx context.Context, cu *entity.ClinicalUpdate) error {
	return m.Called(ctx, cu).Error(0)
}

func (m *MockClinicalUpdateRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockClinicalUpdateRepo) ListByReferralID(ctx context.Context, referralID uuid.UUID) ([]entity.ClinicalUpdate, error) {
	args := m.Called(ctx, referralID)
	return args.Get(0).([]entity.ClinicalUpdate), args.Error(1)
}

func (m *MockClinicalUpdateRepo) ExistsForReferralAndDate(ctx context.Context, referralID uuid.UUID, reason string, date time.Time) (bool, error) {
	args := m.Called(ctx, referralID, reason, date)
	return args.Bool(0), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: ReferralOutcomeRepository
// ---------------------------------------------------------------------------

type MockReferralOutcomeRepo struct {
	mock.Mock
}

func (m *MockReferralOutcomeRepo) Create(ctx context.Context, o *entity.ReferralOutcome) error {
	return m.Called(ctx, o).Error(0)
}

func (m *MockReferralOutcomeRepo) FindByID(ctx context.Context, id interface{}) (*entity.ReferralOutcome, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ReferralOutcome), args.Error(1)
}

func (m *MockReferralOutcomeRepo) FindAll(ctx context.Context) ([]entity.ReferralOutcome, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.ReferralOutcome), args.Error(1)
}

func (m *MockReferralOutcomeRepo) Update(ctx context.Context, o *entity.ReferralOutcome) error {
	return m.Called(ctx, o).Error(0)
}

func (m *MockReferralOutcomeRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockReferralOutcomeRepo) GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.ReferralOutcome, error) {
	args := m.Called(ctx, referralID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ReferralOutcome), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: ReferralAccessRepository
// ---------------------------------------------------------------------------

type MockReferralAccessRepo struct {
	mock.Mock
}

func (m *MockReferralAccessRepo) Create(ctx context.Context, access *entity.ReferralAccess) error {
	args := m.Called(ctx, access)
	return args.Error(0)
}

func (m *MockReferralAccessRepo) GetAccess(ctx context.Context, referralID, userID uuid.UUID) (*entity.ReferralAccess, error) {
	args := m.Called(ctx, referralID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ReferralAccess), args.Error(1)
}

func (m *MockReferralAccessRepo) CheckAccess(ctx context.Context, referralID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, referralID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockReferralAccessRepo) ListActiveByReferral(ctx context.Context, referralID uuid.UUID) ([]entity.ReferralAccess, error) {
	args := m.Called(ctx, referralID)
	return args.Get(0).([]entity.ReferralAccess), args.Error(1)
}

func (m *MockReferralAccessRepo) ListAllByReferral(ctx context.Context, referralID uuid.UUID) ([]entity.ReferralAccess, error) {
	args := m.Called(ctx, referralID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.ReferralAccess), args.Error(1)
}

func (m *MockReferralAccessRepo) RevokeAllByReferral(ctx context.Context, referralID uuid.UUID, reason string) error {
	return m.Called(ctx, referralID, reason).Error(0)
}

func (m *MockReferralAccessRepo) ListByDoctor(ctx context.Context, doctorID uuid.UUID) ([]entity.ReferralAccess, error) {
	args := m.Called(ctx, doctorID)
	return args.Get(0).([]entity.ReferralAccess), args.Error(1)
}

func (m *MockReferralAccessRepo) GetActiveAccess(ctx context.Context, referralID, userID uuid.UUID) (*entity.ReferralAccess, error) {
	args := m.Called(ctx, referralID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ReferralAccess), args.Error(1)
}

func (m *MockReferralAccessRepo) Update(ctx context.Context, access *entity.ReferralAccess) error {
	args := m.Called(ctx, access)
	return args.Error(0)
}

func (m *MockReferralAccessRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockReferralAccessRepo) FindByID(ctx context.Context, id interface{}) (*entity.ReferralAccess, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ReferralAccess), args.Error(1)
}

func (m *MockReferralAccessRepo) FindAll(ctx context.Context) ([]entity.ReferralAccess, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.ReferralAccess), args.Error(1)
}


// ---------------------------------------------------------------------------
// Mock: SchedulingUseCase
// ---------------------------------------------------------------------------

type MockSchedulingUseCase struct {
	mock.Mock
}

func (m *MockSchedulingUseCase) GetCapacityStatus(ctx context.Context, hospitalID, deptID uuid.UUID, dateRangeDays int) ([]dto.CapacityStatusResponse, error) {
	args := m.Called(ctx, hospitalID, deptID, dateRangeDays)
	return args.Get(0).([]dto.CapacityStatusResponse), args.Error(1)
}

func (m *MockSchedulingUseCase) ScheduleAppointment(ctx context.Context, referralID, userID uuid.UUID, req dto.SchedulingRequest) (bool, error) {
	args := m.Called(ctx, referralID, userID, req)
	return args.Bool(0), args.Error(1)
}

func (m *MockSchedulingUseCase) ManualEmergencySchedule(ctx context.Context, referralID uuid.UUID, appointmentDate time.Time, justification string, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, referralID, appointmentDate, justification, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSchedulingUseCase) BatchSchedule(ctx context.Context, hospitalID, deptID, userID uuid.UUID, sendNotifications bool) (*dto.BatchScheduleResult, error) {
	args := m.Called(ctx, hospitalID, deptID, userID, sendNotifications)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BatchScheduleResult), args.Error(1)
}

func (m *MockSchedulingUseCase) ProcessMissedAppointments(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockSchedulingUseCase) EffectiveCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int, int, int64, error) {
	args := m.Called(ctx, hospitalID, deptID, date)
	return args.Int(0), args.Int(1), args.Get(2).(int64), args.Error(3)
}

func (m *MockSchedulingUseCase) ListScheduleOptions(ctx context.Context, referralID uuid.UUID, days int) ([]dto.ScheduleOption, error) {
	args := m.Called(ctx, referralID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ScheduleOption), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: TriageUseCase
// ---------------------------------------------------------------------------

type MockTriageUseCase struct {
	mock.Mock
}

func (m *MockTriageUseCase) LandInQueue(ctx context.Context, referralID uuid.UUID) error {
	return m.Called(ctx, referralID).Error(0)
}

func (m *MockTriageUseCase) CalculateCompositeScore(ctx context.Context, referralID uuid.UUID) (float64, error) {
	args := m.Called(ctx, referralID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTriageUseCase) ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error) {
	args := m.Called(ctx, hospitalID, limit, offset)
	return args.Get(0).([]dto.TriageListResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockTriageUseCase) ListForTriageByDepartment(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]dto.TriageListResponse, int64, error) {
	args := m.Called(ctx, hospitalID, deptID, limit, offset)
	return args.Get(0).([]dto.TriageListResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockTriageUseCase) ReviewTriage(ctx context.Context, referralID, userID uuid.UUID, req dto.TriageReviewRequest) error {
	return m.Called(ctx, referralID, userID, req).Error(0)
}

func (m *MockTriageUseCase) ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID, start, end)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockTriageUseCase) ListTriageFiltered(ctx context.Context, filter dto.TriageListFilter) ([]dto.TriageListItem, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dto.TriageListItem), args.Get(1).(int64), args.Error(2)
}

func (m *MockTriageUseCase) GetTriageDetailForSpecialist(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailSpecialistResponse, error) {
	args := m.Called(ctx, referralID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TriageDetailSpecialistResponse), args.Error(1)
}

func (m *MockTriageUseCase) GetTriageDetailForReceptionist(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailReceptionistResponse, error) {
	args := m.Called(ctx, referralID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TriageDetailReceptionistResponse), args.Error(1)
}

func (m *MockTriageUseCase) GetTriageDetailForDeptHead(ctx context.Context, referralID, userID uuid.UUID) (*dto.TriageDetailDeptHeadResponse, error) {
	args := m.Called(ctx, referralID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TriageDetailDeptHeadResponse), args.Error(1)
}


// ---------------------------------------------------------------------------
// Mock: ArrivalUseCase
// ---------------------------------------------------------------------------

type MockArrivalUseCase struct {
	mock.Mock
}

func (m *MockArrivalUseCase) GetTodayAndTomorrowSchedule(ctx context.Context, hospitalID, deptID uuid.UUID) ([]*entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.TriageQueue), args.Error(1)
}

func (m *MockArrivalUseCase) ConfirmArrival(ctx context.Context, queueID uuid.UUID, userID uuid.UUID, callerHospID, callerDeptID uuid.UUID) error {
	return m.Called(ctx, queueID, userID, callerHospID, callerDeptID).Error(0)
}

func (m *MockArrivalUseCase) ReturnToTriage(ctx context.Context, queueID, userID uuid.UUID, callerHospID, callerDeptID uuid.UUID) error {
	return m.Called(ctx, queueID, userID, callerHospID, callerDeptID).Error(0)
}

func (m *MockArrivalUseCase) AssignDoctor(ctx context.Context, queueID uuid.UUID, doctorID uuid.UUID, userID uuid.UUID, reason string, callerHospID, callerDeptID uuid.UUID) error {
	return m.Called(ctx, queueID, doctorID, userID, reason, callerHospID, callerDeptID).Error(0)
}

func (m *MockArrivalUseCase) MarkMissed(ctx context.Context, queueID uuid.UUID, missReason entity.MissReason, userID uuid.UUID, callerHospID, callerDeptID uuid.UUID) error {
	return m.Called(ctx, queueID, missReason, userID, callerHospID, callerDeptID).Error(0)
}


func (m *MockArrivalUseCase) RevokeDoctorAssignment(ctx context.Context, queueID, userID uuid.UUID, reason string, callerHospID, callerDeptID uuid.UUID) error {
	return m.Called(ctx, queueID, userID, reason, callerHospID, callerDeptID).Error(0)
}

func (m *MockArrivalUseCase) ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error) {
	args := m.Called(ctx, hospitalID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*entity.TriageQueue), args.Get(1).(int64), args.Error(2)
}

func (m *MockArrivalUseCase) GrantConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID) error {
	return m.Called(ctx, referralID, granterID, doctorID).Error(0)
}

func (m *MockArrivalUseCase) RevokeConsultAccess(ctx context.Context, referralID, granterID, doctorID uuid.UUID, reason string) error {
	return m.Called(ctx, referralID, granterID, doctorID, reason).Error(0)
}

func (m *MockArrivalUseCase) GetTriageQueueByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error) {
	args := m.Called(ctx, referralID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.TriageQueue), args.Error(1)
}



// ---------------------------------------------------------------------------
// Mock: ClinicalUseCase
// ---------------------------------------------------------------------------

type MockClinicalUseCase struct {
	mock.Mock
}

func (m *MockClinicalUseCase) AddClinicalUpdate(ctx context.Context, referralID, userID uuid.UUID, req dto.AddClinicalUpdateRequest) error {
	return m.Called(ctx, referralID, userID, req).Error(0)
}

func (m *MockClinicalUseCase) RecordOutcome(ctx context.Context, referralID, userID uuid.UUID, req dto.RecordOutcomeRequest) error {
	return m.Called(ctx, referralID, userID, req).Error(0)
}

func (m *MockClinicalUseCase) GetClinicalHistory(ctx context.Context, referralID, userID uuid.UUID) ([]entity.ClinicalUpdate, error) {
	args := m.Called(ctx, referralID, userID)
	return args.Get(0).([]entity.ClinicalUpdate), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: CapacityManagementUseCase
// ---------------------------------------------------------------------------

type MockCapacityManagementUseCase struct {
	mock.Mock
}

func (m *MockCapacityManagementUseCase) GetSchedule(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]entity.DailySchedule, error) {
	args := m.Called(ctx, hospitalID, deptID, startDate, endDate)
	return args.Get(0).([]entity.DailySchedule), args.Error(1)
}

func (m *MockCapacityManagementUseCase) GetOverrides(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.CapacityOverride, error) {
	args := m.Called(ctx, hospitalID, deptID)
	return args.Get(0).([]entity.CapacityOverride), args.Error(1)
}

func (m *MockCapacityManagementUseCase) CreateOverride(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, newLimit int, reason string, userID uuid.UUID) error {
	return m.Called(ctx, hospitalID, deptID, date, newLimit, reason, userID).Error(0)
}

func (m *MockCapacityManagementUseCase) DeleteOverride(ctx context.Context, overrideID, userID uuid.UUID) error {
	return m.Called(ctx, overrideID, userID).Error(0)
}

func (m *MockCapacityManagementUseCase) ListOverridesByYearMonth(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]entity.CapacityOverride, error) {
	args := m.Called(ctx, hospitalID, deptID, year, month)
	return args.Get(0).([]entity.CapacityOverride), args.Error(1)
}

func (m *MockCapacityManagementUseCase) GetOverride(ctx context.Context, overrideID uuid.UUID) (*entity.CapacityOverride, error) {
	args := m.Called(ctx, overrideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.CapacityOverride), args.Error(1)
}

func (m *MockCapacityManagementUseCase) GetCapacityDetail(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*dto.CapacityDetailResponse, error) {
	args := m.Called(ctx, hospitalID, deptID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CapacityDetailResponse), args.Error(1)
}

func (m *MockCapacityManagementUseCase) GetScheduledPatientsForDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) ([]entity.TriageQueue, error) {
	args := m.Called(ctx, hospitalID, deptID, date)
	return args.Get(0).([]entity.TriageQueue), args.Error(1)
}

func (m *MockCapacityManagementUseCase) BuildCapacityCalendar(ctx context.Context, hospitalID, deptID uuid.UUID, year, month int) ([]dto.CapacityCalendarDay, error) {
	args := m.Called(ctx, hospitalID, deptID, year, month)
	return args.Get(0).([]dto.CapacityCalendarDay), args.Error(1)
}

func (m *MockCapacityManagementUseCase) UpdateStaffCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, value int, userID uuid.UUID) error {
	return m.Called(ctx, hospitalID, deptID, value, userID).Error(0)
}

func (m *MockCapacityManagementUseCase) UpdateDailyCapacity(ctx context.Context, hospitalID, deptID uuid.UUID, standardDailyLimit, overbookLimit int, userID uuid.UUID) error {
	return m.Called(ctx, hospitalID, deptID, standardDailyLimit, overbookLimit, userID).Error(0)
}

func (m *MockCapacityManagementUseCase) GetDailyCapacity(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.DeptHeadDailyCapacityResponse, error) {
	args := m.Called(ctx, hospitalID, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.DeptHeadDailyCapacityResponse), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: DepartmentHeadDashboardUseCase
// ---------------------------------------------------------------------------

type MockDepartmentHeadDashboardUseCase struct {
	mock.Mock
}

func (m *MockDepartmentHeadDashboardUseCase) GetDashboardStats(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.DepartmentHeadDashboardStats, error) {
	args := m.Called(ctx, hospitalID, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.DepartmentHeadDashboardStats), args.Error(1)
}

func (m *MockDepartmentHeadDashboardUseCase) GetTrends(ctx context.Context, hospitalID, deptID uuid.UUID, days int) ([]dto.DepartmentHeadTrendPoint, error) {
	args := m.Called(ctx, hospitalID, deptID, days)
	return args.Get(0).([]dto.DepartmentHeadTrendPoint), args.Error(1)
}

func (m *MockDepartmentHeadDashboardUseCase) GetPriorityBuckets(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.PriorityBucketResponse, error) {
	args := m.Called(ctx, hospitalID, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PriorityBucketResponse), args.Error(1)
}

func (m *MockDepartmentHeadDashboardUseCase) GetStaffSummary(ctx context.Context, hospitalID, deptID uuid.UUID) (*dto.StaffSummaryResponse, error) {
	args := m.Called(ctx, hospitalID, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.StaffSummaryResponse), args.Error(1)
}

func (m *MockDepartmentHeadDashboardUseCase) GetActivity(ctx context.Context, hospitalID, deptID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dto.DepartmentHeadActivityItem, error) {
	args := m.Called(ctx, hospitalID, deptID, limit, startDate, endDate)
	return args.Get(0).([]dto.DepartmentHeadActivityItem), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: NotificationRepository
// ---------------------------------------------------------------------------

type MockNotificationRepo struct {
	mock.Mock
}

func (m *MockNotificationRepo) Create(ctx context.Context, n *entity.Notification) error {
	return m.Called(ctx, n).Error(0)
}

func (m *MockNotificationRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Notification), args.Error(1)
}

func (m *MockNotificationRepo) GetQueued(ctx context.Context, limit int) ([]entity.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]entity.Notification), args.Error(1)
}

func (m *MockNotificationRepo) GetQueuedByFilter(ctx context.Context, hospitalID, deptID *uuid.UUID, limit int) ([]entity.Notification, error) {
	args := m.Called(ctx, hospitalID, deptID, limit)
	return args.Get(0).([]entity.Notification), args.Error(1)
}

func (m *MockNotificationRepo) GetSent(ctx context.Context, limit int) ([]entity.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]entity.Notification), args.Error(1)
}

func (m *MockNotificationRepo) UpdateDelivery(ctx context.Context, id uuid.UUID, status entity.DeliveryStatus, messageID *string) error {
	return m.Called(ctx, id, status, messageID).Error(0)
}

// ---------------------------------------------------------------------------
// Mock: NotificationUseCase
// ---------------------------------------------------------------------------

type MockNotificationUseCase struct {
	mock.Mock
}

func (m *MockNotificationUseCase) QueueNotification(ctx context.Context, referralID uuid.UUID, notifType entity.NotificationType, message string) error {
	return m.Called(ctx, referralID, notifType, message).Error(0)
}

func (m *MockNotificationUseCase) TriggerManualSend(ctx context.Context, hospitalID, deptID *uuid.UUID) (*dto.NotificationSendSummary, error) {
	args := m.Called(ctx, hospitalID, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.NotificationSendSummary), args.Error(1)
}

func (m *MockNotificationUseCase) ProcessPendingSMS(ctx context.Context, limit int) (*dto.NotificationSendSummary, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.NotificationSendSummary), args.Error(1)
}

func (m *MockNotificationUseCase) UpdateStatus(ctx context.Context) (*dto.NotificationStatusSummary, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.NotificationStatusSummary), args.Error(1)
}

func (m *MockNotificationUseCase) ResendNotification(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Notification), args.Error(1)
}

func (m *MockNotificationUseCase) QueueReminders(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockNotificationUseCase) HandleSMSWebhook(ctx context.Context, messageID, status string) error {
	return m.Called(ctx, messageID, status).Error(0)
}

func (m *MockNotificationUseCase) ListNotifications(ctx context.Context, filter irepository.NotificationListFilter) ([]entity.Notification, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]entity.Notification), args.Get(1).(int64), args.Error(2)
}

// ---------------------------------------------------------------------------
// Mock: DailyWeightUseCase
// ---------------------------------------------------------------------------

type MockDailyWeightUseCase struct {
	mock.Mock
}

func (m *MockDailyWeightUseCase) Execute(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: SystemConfigRepository
// ---------------------------------------------------------------------------

type MockSystemConfigRepo struct {
	mock.Mock
}

func (m *MockSystemConfigRepo) Create(ctx context.Context, cfg *entity.SystemConfig) error {
	return m.Called(ctx, cfg).Error(0)
}

func (m *MockSystemConfigRepo) FindByID(ctx context.Context, id interface{}) (*entity.SystemConfig, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.SystemConfig), args.Error(1)
}

func (m *MockSystemConfigRepo) FindAll(ctx context.Context) ([]entity.SystemConfig, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.SystemConfig), args.Error(1)
}

func (m *MockSystemConfigRepo) Update(ctx context.Context, cfg *entity.SystemConfig) error {
	return m.Called(ctx, cfg).Error(0)
}

func (m *MockSystemConfigRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSystemConfigRepo) GetByKey(ctx context.Context, key string) (*entity.SystemConfig, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.SystemConfig), args.Error(1)
}

func (m *MockSystemConfigRepo) GetAll(ctx context.Context) (map[string]string, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockSystemConfigRepo) BulkUpdate(ctx context.Context, updates map[string]string) error {
	return m.Called(ctx, updates).Error(0)
}

func (m *MockSystemConfigRepo) GetBool(ctx context.Context, key string, defaultValue bool) (bool, error) {
	args := m.Called(ctx, key, defaultValue)
	return args.Bool(0), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: SchedulerCheckpointRepository
// ---------------------------------------------------------------------------

type MockSchedulerCheckpointRepo struct {
	mock.Mock
}

func (m *MockSchedulerCheckpointRepo) Create(ctx context.Context, cp *entity.SchedulerCheckpoint) error {
	return m.Called(ctx, cp).Error(0)
}

func (m *MockSchedulerCheckpointRepo) FindByID(ctx context.Context, id interface{}) (*entity.SchedulerCheckpoint, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.SchedulerCheckpoint), args.Error(1)
}

func (m *MockSchedulerCheckpointRepo) FindAll(ctx context.Context) ([]entity.SchedulerCheckpoint, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.SchedulerCheckpoint), args.Error(1)
}

func (m *MockSchedulerCheckpointRepo) Update(ctx context.Context, cp *entity.SchedulerCheckpoint) error {
	return m.Called(ctx, cp).Error(0)
}

func (m *MockSchedulerCheckpointRepo) Delete(ctx context.Context, id interface{}) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSchedulerCheckpointRepo) GetNextEligibleDepartment(ctx context.Context, minAge time.Duration, leaseHolder string, leaseDuration time.Duration) (*entity.SchedulerCheckpoint, error) {
	args := m.Called(ctx, minAge, leaseHolder, leaseDuration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.SchedulerCheckpoint), args.Error(1)
}

func (m *MockSchedulerCheckpointRepo) UpdateLastProcessed(ctx context.Context, hospitalID, deptID uuid.UUID, leaseHolder string) error {
	return m.Called(ctx, hospitalID, deptID, leaseHolder).Error(0)
}

func (m *MockSchedulerCheckpointRepo) ReleaseLease(ctx context.Context, hospitalID, deptID uuid.UUID, leaseHolder string) error {
	return m.Called(ctx, hospitalID, deptID, leaseHolder).Error(0)
}

func (m *MockSchedulerCheckpointRepo) AcquireLease(ctx context.Context, hospitalID, deptID uuid.UUID, leaseHolder string, leaseDuration time.Duration) (bool, error) {
	args := m.Called(ctx, hospitalID, deptID, leaseHolder, leaseDuration)
	return args.Bool(0), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: SchedulerServiceUseCase
// ---------------------------------------------------------------------------

type MockSchedulerServiceUseCase struct {
	mock.Mock
}

func (m *MockSchedulerServiceUseCase) RunSchedulerCycle(ctx context.Context, leaseHolder string) (*dto.BatchScheduleResult, error) {
	args := m.Called(ctx, leaseHolder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BatchScheduleResult), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: UserUseCase
// ---------------------------------------------------------------------------

type MockUserUseCase struct {
	mock.Mock
}

func (m *MockUserUseCase) CreateUser(ctx context.Context, user *entity.User, rawPassword string) error {
	args := m.Called(ctx, user, rawPassword)
	return args.Error(0)
}

func (m *MockUserUseCase) GetUserByID(ctx context.Context, id, requesterID uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id, requesterID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUseCase) GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUseCase) UpdateMyProfile(ctx context.Context, userID uuid.UUID, input iusecase.UpdateMyProfileInput) (*entity.User, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUseCase) UpdateUser(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserUseCase) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserUseCase) ListUsers(ctx context.Context, filter irepository.UserListFilter, requesterID uuid.UUID) ([]entity.User, int64, error) {
	args := m.Called(ctx, filter, requesterID)
	return args.Get(0).([]entity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserUseCase) ListDepartmentStaff(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.User, error) {
	args := m.Called(ctx, hospitalID, deptID)
	return args.Get(0).([]entity.User), args.Error(1)
}

func (m *MockUserUseCase) AssignRole(ctx context.Context, userID uuid.UUID, role entity.UserRole) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *MockUserUseCase) DeleteProfileImage(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserUseCase) ModerateProfileImage(ctx context.Context, userID, moderatorID uuid.UUID) error {
	args := m.Called(ctx, userID, moderatorID)
	return args.Error(0)
}

func (m *MockUserUseCase) UpdateProfileImage(ctx context.Context, userID uuid.UUID, file interface{}) error {
	args := m.Called(ctx, userID, file)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminCreateStaff(ctx context.Context, adminID uuid.UUID, user *entity.User, rawPassword string) error {
	args := m.Called(ctx, adminID, user, rawPassword)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminListStaff(ctx context.Context, adminID uuid.UUID, filter irepository.UserListFilter) ([]entity.User, int64, error) {
	args := m.Called(ctx, adminID, filter)
	return args.Get(0).([]entity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserUseCase) HospitalAdminGetStaffByID(ctx context.Context, adminID, staffID uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, adminID, staffID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUseCase) HospitalAdminChangeStaffRole(ctx context.Context, adminID, staffID uuid.UUID, role entity.UserRole) error {
	args := m.Called(ctx, adminID, staffID, role)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminSoftDeleteStaff(ctx context.Context, adminID, staffID uuid.UUID) error {
	args := m.Called(ctx, adminID, staffID)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminReplaceStaff(ctx context.Context, adminID, staffID uuid.UUID, input iusecase.HospitalAdminReplacementInput) error {
	args := m.Called(ctx, adminID, staffID, input)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminSetStaffActive(ctx context.Context, adminID, staffID uuid.UUID, isActive bool) error {
	args := m.Called(ctx, adminID, staffID, isActive)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminReassignStaffDepartment(ctx context.Context, adminID, staffID uuid.UUID, departmentID *uuid.UUID) error {
	args := m.Called(ctx, adminID, staffID, departmentID)
	return args.Error(0)
}

func (m *MockUserUseCase) HospitalAdminListActiveStaffSessions(ctx context.Context, adminID uuid.UUID, filter iusecase.HospitalAdminSessionFilter) ([]entity.Session, int64, error) {
	args := m.Called(ctx, adminID, filter)
	return args.Get(0).([]entity.Session), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserUseCase) HospitalAdminForceLogoutStaff(ctx context.Context, adminID, staffID uuid.UUID) (int64, error) {
	args := m.Called(ctx, adminID, staffID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserUseCase) GetPersonnelWidgetStats(ctx context.Context, adminID uuid.UUID) (*dto.HospitalAdminPersonnelWidgetResponse, error) {
	args := m.Called(ctx, adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.HospitalAdminPersonnelWidgetResponse), args.Error(1)
}

func (m *MockUserUseCase) GetDepartmentHeadsByHospital(ctx context.Context, hospitalID uuid.UUID) (map[uuid.UUID]*entity.User, error) {
	args := m.Called(ctx, hospitalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*entity.User), args.Error(1)
}


// ---------------------------------------------------------------------------
// Mock: HospitalUseCase
// ---------------------------------------------------------------------------

type MockHospitalUseCase struct {
	mock.Mock
}

func (m *MockHospitalUseCase) CreateHospital(ctx context.Context, hospital *entity.Hospital) error {
	args := m.Called(ctx, hospital)
	return args.Error(0)
}

func (m *MockHospitalUseCase) GetHospitalByID(ctx context.Context, id uuid.UUID) (*entity.Hospital, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Hospital), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockHospitalUseCase) UpdateHospital(ctx context.Context, hospital *entity.Hospital) error {
	args := m.Called(ctx, hospital)
	return args.Error(0)
}

func (m *MockHospitalUseCase) DeleteHospital(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockHospitalUseCase) ListHospitals(ctx context.Context, filter irepository.HospitalListFilter) ([]entity.Hospital, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Hospital), args.Get(1).(int64), args.Error(2)
}

func (m *MockHospitalUseCase) UpdateSystemConfig(ctx context.Context, userID uuid.UUID, req map[string]string) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockHospitalUseCase) GetSystemConfigs(ctx context.Context) (map[string]string, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]string), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: DepartmentUseCase
// ---------------------------------------------------------------------------

type MockDepartmentUseCase struct {
	mock.Mock
}

func (m *MockDepartmentUseCase) CreateDepartment(ctx context.Context, dept *entity.Department) error {
	args := m.Called(ctx, dept)
	return args.Error(0)
}

func (m *MockDepartmentUseCase) GetDepartmentByID(ctx context.Context, id uuid.UUID) (*entity.Department, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Department), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDepartmentUseCase) UpdateDepartment(ctx context.Context, dept *entity.Department) error {
	args := m.Called(ctx, dept)
	return args.Error(0)
}

func (m *MockDepartmentUseCase) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDepartmentUseCase) ListDepartments(ctx context.Context, filter irepository.DepartmentListFilter) ([]entity.Department, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Department), args.Get(1).(int64), args.Error(2)
}

func (m *MockDepartmentUseCase) LinkDepartmentToHospital(ctx context.Context, hospitalID, departmentID uuid.UUID, dailyLimit int) error {
	args := m.Called(ctx, hospitalID, departmentID, dailyLimit)
	return args.Error(0)
}

func (m *MockDepartmentUseCase) UnlinkDepartmentFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error {
	args := m.Called(ctx, hospitalID, departmentID)
	return args.Error(0)
}

func (m *MockDepartmentUseCase) ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error) {
	args := m.Called(ctx, hospitalID)
	return args.Get(0).([]entity.HospitalDepartment), args.Error(1)
}

func (m *MockDepartmentUseCase) SetHospitalDepartmentActive(ctx context.Context, hospitalID, departmentOrLinkID uuid.UUID, isActive bool) error {
	args := m.Called(ctx, hospitalID, departmentOrLinkID, isActive)
	return args.Error(0)
}

func (m *MockDepartmentUseCase) GetHospitalDepartmentLink(ctx context.Context, hospitalID, departmentOrLinkID uuid.UUID) (*entity.HospitalDepartment, error) {
	args := m.Called(ctx, hospitalID, departmentOrLinkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.HospitalDepartment), args.Error(1)
}

func (m *MockDepartmentUseCase) ValidateDepartmentForHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error {
	args := m.Called(ctx, hospitalID, departmentID)
	return args.Error(0)
}


// ---------------------------------------------------------------------------
// Mock: InAppNotificationRepository
// ---------------------------------------------------------------------------

type MockInAppNotificationRepo struct {
	mock.Mock
}

func (m *MockInAppNotificationRepo) Create(ctx context.Context, n *entity.InAppNotification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *MockInAppNotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, filter irepository.InAppNotificationFilter, limit, offset int) ([]entity.InAppNotification, int64, error) {
	args := m.Called(ctx, userID, filter, limit, offset)
	return args.Get(0).([]entity.InAppNotification), args.Get(1).(int64), args.Error(2)
}

func (m *MockInAppNotificationRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockInAppNotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockInAppNotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInAppNotificationRepo) FindByID(ctx context.Context, id interface{}) (*entity.InAppNotification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.InAppNotification), args.Error(1)
}

func (m *MockInAppNotificationRepo) FindAll(ctx context.Context) ([]entity.InAppNotification, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.InAppNotification), args.Error(1)
}

func (m *MockInAppNotificationRepo) Update(ctx context.Context, n *entity.InAppNotification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *MockInAppNotificationRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Mock: InAppNotificationUseCase
// ---------------------------------------------------------------------------

type MockInAppNotificationUseCase struct {
	mock.Mock
}

func (m *MockInAppNotificationUseCase) CreateForEvent(ctx context.Context, eventType string, referralID uuid.UUID, actorID uuid.UUID) error {
	args := m.Called(ctx, eventType, referralID, actorID)
	return args.Error(0)
}

func (m *MockInAppNotificationUseCase) ListForUser(ctx context.Context, userID uuid.UUID, filter irepository.InAppNotificationFilter, limit, page int) ([]entity.InAppNotification, int64, int64, error) {
	args := m.Called(ctx, userID, filter, limit, page)
	return args.Get(0).([]entity.InAppNotification), args.Get(1).(int64), args.Get(2).(int64), args.Error(3)
}

func (m *MockInAppNotificationUseCase) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockInAppNotificationUseCase) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockInAppNotificationUseCase) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: PatientUseCase
// ---------------------------------------------------------------------------

type MockPatientUseCase struct {
	mock.Mock
}

func (m *MockPatientUseCase) GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error) {
	args := m.Called(ctx, nationalID)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) LookupPatient(ctx context.Context, nationalID, phone string) (*entity.Patient, error) {
	args := m.Called(ctx, nationalID, phone)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error) {
	args := m.Called(ctx, req)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) SearchPatients(ctx context.Context, query string) ([]entity.Patient, error) {
	args := m.Called(ctx, query)
	if patients := args.Get(0); patients != nil {
		return patients.([]entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) LookupByNationalID(ctx context.Context, nationalID string) (*uuid.UUID, error) {
	args := m.Called(ctx, nationalID)
	if id := args.Get(0); id != nil {
		return id.(*uuid.UUID), args.Error(1)
	}
	return nil, args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: JobCheckpointRepository
// ---------------------------------------------------------------------------

type MockJobCheckpointRepo struct {
	mock.Mock
}

func (m *MockJobCheckpointRepo) GetLastRun(ctx context.Context, jobName string) (*time.Time, error) {
	args := m.Called(ctx, jobName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockJobCheckpointRepo) UpdateLastRun(ctx context.Context, jobName string, timestamp time.Time) error {
	return m.Called(ctx, jobName, timestamp).Error(0)
}

// ---------------------------------------------------------------------------
// Mock: MLUseCase
// ---------------------------------------------------------------------------

type MockMLUseCase struct {
	mock.Mock
}

func (m *MockMLUseCase) ScheduleScore(referralID uuid.UUID) {
	m.Called(referralID)
}

func (m *MockMLUseCase) ScheduleScoreForce(referralID uuid.UUID) {
	m.Called(referralID)
}

func (m *MockMLUseCase) ScoreReferral(ctx context.Context, referralID uuid.UUID) error {
	return m.Called(ctx, referralID).Error(0)
}

func (m *MockMLUseCase) ScoreReferralForce(ctx context.Context, referralID uuid.UUID) error {
	return m.Called(ctx, referralID).Error(0)
}

func (m *MockMLUseCase) SendFeedbackAccept(ctx context.Context, referralID uuid.UUID) error {
	return m.Called(ctx, referralID).Error(0)
}

func (m *MockMLUseCase) SendFeedbackOverride(ctx context.Context, referralID uuid.UUID, correctedScore float64, doctorExplanation string) error {
	return m.Called(ctx, referralID, correctedScore, doctorExplanation).Error(0)
}

func (m *MockMLUseCase) MLSeverityOverride(ctx context.Context, referralID, userID uuid.UUID, score float64, justification string) error {
	return m.Called(ctx, referralID, userID, score, justification).Error(0)
}

func (m *MockMLUseCase) ProcessMLResult(ctx context.Context, referralID uuid.UUID, score float64, confidence float64, severityTier string, explanation json.RawMessage, modelVersion string, inputFeatures json.RawMessage, externalPredictionID *string, processingTimeMs *float64, triggerReason string) error {
	return m.Called(ctx, referralID, score, confidence, severityTier, explanation, modelVersion, inputFeatures, externalPredictionID, processingTimeMs, triggerReason).Error(0)
}

