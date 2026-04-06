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
