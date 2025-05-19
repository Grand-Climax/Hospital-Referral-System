package usecasetest

import (
	"Hospital-Referral-System/internal/domain/entity"
	mockrepository "Hospital-Referral-System/internal/test/mocks/mock_repository"
	"Hospital-Referral-System/internal/usecase"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDepartment(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	departmentusecase := usecase.NewDepartmentUsecase(mockrepo)

	// create the department
	department := &entity.Department{Name: "Cardio"}
	expectedDepartment := &entity.Department{ID: 1, Name: "Cardio"}
	
	mockrepo.On("CreateDepartment", department).Return(expectedDepartment,nil)

	result, err := departmentusecase.CreateDepartment(department)

	assert.NoError(t, err)
	assert.Equal(t, result, expectedDepartment)

	mockrepo.AssertExpectations(t)
}

func TestGetDepartments(t *testing.T) {
    mockrepo := new(mockrepository.MockDepartmentRepository)
    departmentusecase := usecase.NewDepartmentUsecase(mockrepo)

    department1 := &entity.Department{Name: "Cardio"}
    department2 := &entity.Department{Name: "Surgery"}

    // Mock CreateDepartment for department1
    mockrepo.On("CreateDepartment", department1).Return(department1, nil)
    // Mock CreateDepartment for department2
    mockrepo.On("CreateDepartment", department2).Return(department2, nil)

    // Create first department
    _, err := departmentusecase.CreateDepartment(department1)
    if err != nil {
        t.Fatalf("Couldn't add a new department %v", err)
    }

    // Create second department
    _, err = departmentusecase.CreateDepartment(department2)
    if err != nil {
        t.Fatalf("Couldn't add a new department %v", err)
    }

    // Expected departments as a slice (to match GetDepartments return type)
    expectedDepartments := []*entity.Department{department1, department2}

    // Mock GetDepartments
    mockrepo.On("GetDepartments").Return(expectedDepartments, nil)

    // Call GetDepartments
    departments, err := departmentusecase.GetDepartments()

    // Assert results
    assert.NoError(t, err)
    assert.Len(t, departments, 2)
    assert.Equal(t, departments[0].Name, expectedDepartments[0].Name)
    assert.Equal(t, departments[1].Name, expectedDepartments[1].Name)

    // Verify all expectations were met
    mockrepo.AssertExpectations(t)
}

func TestUpdateDepartment(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	departmentusecase := usecase.NewDepartmentUsecase(mockrepo)

	input := &entity.Department{ID: 1, Name: "Cardio"}
	mockrepo.On("UpdateDepartment", input).Return(input, nil)

	department, err := departmentusecase.UpdateDepartment(input)

	assert.NoError(t, err)
	assert.Equal(t, department, input)

	mockrepo.AssertExpectations(t)
}

func TestDeleteDepartment(t *testing.T) {
	mockrepo := new(mockrepository.MockDepartmentRepository)
	departmentusecase := usecase.NewDepartmentUsecase(mockrepo)

	id := uint(1)
	mockrepo.On("DeleteDepartment", uint(1)).Return(nil)

	err := departmentusecase.DeleteDepartment(id)

	assert.NoError(t, err)

	mockrepo.AssertExpectations(t)
}