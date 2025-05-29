package usecasetest

import (
	"Hospital-Referral-System/internal/domain/entity"
	mockrepository "Hospital-Referral-System/internal/test/mocks/mock_repository"
    customerrors "Hospital-Referral-System/internal/errors"
	"Hospital-Referral-System/internal/usecase"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDepartment_Success(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	department := &entity.Department{Name: "Cardio"}
	expected := &entity.Department{ID: 1, Name: "Cardio"}

	mockrepo.On("GetDepartmentByName", "Cardio").Return(nil, customerrors.ErrNotFound)
	mockrepo.On("CreateDepartment", department).Return(expected, nil)

	result, err := usecase.CreateDepartment(department)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	mockrepo.AssertExpectations(t)
}

func TestCreateDepartment_Duplicate(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	department := &entity.Department{Name: "Cardio"}
	existing := &entity.Department{ID: 1, Name: "Cardio"}

	mockrepo.On("GetDepartmentByName", "Cardio").Return(existing, nil)

	result, err := usecase.CreateDepartment(department)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, customerrors.ErrDuplicate)

	mockrepo.AssertExpectations(t)
}

func TestCreateDepartment_InternalError(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	department := &entity.Department{Name: "Cardio"}

	mockrepo.On("GetDepartmentByName", "Cardio").Return(nil, customerrors.ErrInternal)

	result, err := usecase.CreateDepartment(department)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, customerrors.ErrInternal)

	mockrepo.AssertExpectations(t)
}

func TestGetDepartments_Success(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	expected := []*entity.Department{
		{ID: 1, Name: "Cardio"},
		{ID: 2, Name: "Surgery"},
	}

	mockrepo.On("GetDepartments").Return(expected, nil)

	result, err := usecase.GetDepartments()

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	mockrepo.AssertExpectations(t)
}

func TestGetDepartmentByName_Success(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	expected := &entity.Department{ID: 1, Name: "Cardio"}

	mockrepo.On("GetDepartmentByName", "Cardio").Return(expected, nil)

	result, err := usecase.GetDepartmentByName("Cardio")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	mockrepo.AssertExpectations(t)
}

func TestUpdateDepartment_Success(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	input := &entity.Department{ID: 1, Name: "Updated"}
	mockrepo.On("GetDepartmentByID", input.ID).Return(input, nil)
	mockrepo.On("UpdateDepartment", input).Return(input, nil)

	result, err := usecase.UpdateDepartment(input)

	assert.NoError(t, err)
	assert.Equal(t, input, result)

	mockrepo.AssertExpectations(t)
}

func TestUpdateDepartment_NotFound(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	input := &entity.Department{ID: 99, Name: "Unknown"}

	mockrepo.On("GetDepartmentByID", input.ID).Return(nil, customerrors.ErrNotFound)

	result, err := usecase.UpdateDepartment(input)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, customerrors.ErrNotFound)

	mockrepo.AssertExpectations(t)
}

func TestDeleteDepartment_Success(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	id := uint(1)
	department := &entity.Department{ID: id, Name: "Cardio"}

	mockrepo.On("GetDepartmentByID", id).Return(department, nil)
	mockrepo.On("DeleteDepartment", id).Return(nil)

	err := usecase.DeleteDepartment(id)

	assert.NoError(t, err)

	mockrepo.AssertExpectations(t)
}

func TestDeleteDepartment_NotFound(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	usecase := usecase.NewDepartmentUsecase(mockrepo)

	id := uint(100)

	mockrepo.On("GetDepartmentByID", id).Return(nil, customerrors.ErrNotFound)

	err := usecase.DeleteDepartment(id)

	assert.ErrorIs(t, err, customerrors.ErrNotFound)

	mockrepo.AssertExpectations(t)
}
