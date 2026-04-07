package test

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
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

func (m *MockReferralRepo) CreateStatusHistory(ctx context.Context, history *entity.ReferralStatusHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockReferralRepo) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralRepo) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, limit, page)
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

func (m *MockReferralUseCase) LiaisonUnassignSpecialist(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, liaisonID, hospID, reason)
	return args.Error(0)
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

func (m *MockReferralUseCase) ConfirmAttendance(ctx context.Context, id, receptionistID, hospID uuid.UUID, status string) error {
	args := m.Called(ctx, id, receptionistID, hospID, status)
	return args.Error(0)
}

func (m *MockReferralUseCase) ListForSystemAdmin(ctx context.Context, filter irepository.ReferralFilter) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, page int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, limit, page)
	return args.Get(0).([]entity.ReferralStatusHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockReferralUseCase) IsValidStatus(status string) bool {
	args := m.Called(status)
	return args.Bool(0)
}

// ---------------------------------------------------------------------------
// Mock: AttachmentUseCase
// ---------------------------------------------------------------------------

type MockAttachmentUseCase struct {
	mock.Mock
}

func (m *MockAttachmentUseCase) SaveAttachment(ctx context.Context, referralID uuid.UUID, fileName, fileType, storagePath, publicID, category string, fileSize int64) (*entity.Attachment, error) {
	args := m.Called(ctx, referralID, fileName, fileType, storagePath, publicID, category, fileSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Attachment), args.Error(1)
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

func (m *MockAttachmentUseCase) GenerateSignature(hospitalID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(hospitalID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAttachmentUseCase) AddAttachmentsToReferral(ctx context.Context, referralID, doctorID uuid.UUID, reqs []dto.CreateAttachmentRequest) ([]entity.Attachment, error) {
	args := m.Called(ctx, referralID, doctorID, reqs)
	return args.Get(0).([]entity.Attachment), args.Error(1)
}

func (m *MockAttachmentUseCase) DeleteAttachmentFromReferral(ctx context.Context, referralID, attachmentID, doctorID uuid.UUID) error {
	args := m.Called(ctx, referralID, attachmentID, doctorID)
	return args.Error(0)
}

func (m *MockAttachmentUseCase) ProcessWebhookAttachment(ctx context.Context, payload map[string]interface{}) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockAttachmentUseCase) UploadAndAddAttachment(ctx context.Context, referralID, doctorID uuid.UUID, file interface{}, fileName, fileType, category string, fileSize int64) (*entity.Attachment, error) {
	args := m.Called(ctx, referralID, doctorID, file, fileName, fileType, category, fileSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Attachment), args.Error(1)
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

func (m *MockStorageService) VerifyWebhookSignature(headers map[string]string, body []byte) (bool, error) {
	args := m.Called(headers, body)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageService) UploadFile(ctx context.Context, file interface{}, folder string) (string, string, error) {
	args := m.Called(ctx, file, folder)
	return args.String(0), args.String(1), args.Error(2)
}
