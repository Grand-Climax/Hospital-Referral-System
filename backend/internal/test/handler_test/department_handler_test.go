package handlertest

import (
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	customerrors "Hospital-Referral-System/internal/errors"
	mockusecase "Hospital-Referral-System/internal/test/mocks/mock_usecase"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DepartmentHandlerTestSuite struct {
	suite.Suite
	handler    *handlers.DepartmentHandler
	mockUsecase *mockusecase.MockDepartmentUsecase
	ginCtx     *gin.Context
	recorder   *httptest.ResponseRecorder
}

func (suite *DepartmentHandlerTestSuite) SetupTest() {
	suite.mockUsecase = mockusecase.NewDepartmentUsecaseMock()
	suite.handler = handlers.NewDepartmentHandler(suite.mockUsecase)
	suite.recorder = httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	suite.ginCtx, _ = gin.CreateTestContext(suite.recorder)
}

func TestDepartmentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DepartmentHandlerTestSuite))
}

func (suite *DepartmentHandlerTestSuite) TestCreateDepartment_Success() {
	department := &entity.Department{Name: "Engineering"}
	
	
	suite.mockUsecase.On("CreateDepartment", department).Return(department, nil)

	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.CreateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusCreated, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Successfully Created", response["message"])
	assert.Equal(suite.T(), "Engineering", response["data"].(map[string]interface{})["name"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestCreateDepartment_InvalidJSON() {
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer([]byte("invalid json")))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.CreateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusBadRequest, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Invalid Input", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestCreateDepartment_EmptyName() {
	department := &entity.Department{Name: ""}
	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.CreateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusUnprocessableEntity, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Name field can not be empty", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestCreateDepartment_DuplicateError() {
	department := &entity.Department{Name: "Engineering"}
	suite.mockUsecase.On("CreateDepartment", department).Return((*entity.Department)(nil), customerrors.ErrDuplicate)

	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.CreateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusConflict, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "department name already exists can't create a department with the given name", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestCreateDepartment_InternalError() {
	department := &entity.Department{Name: "Engineering"}
	suite.mockUsecase.On("CreateDepartment", department).Return((*entity.Department)(nil), errors.New("internal error"))

	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.CreateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusInternalServerError, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Something went wrong", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartments_Success() {
	departments := []*entity.Department{{ID: 1, Name: "Engineering"}, {ID: 2, Name: "HR"}}
	suite.mockUsecase.On("GetDepartments").Return(departments, nil)

	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments", nil)

	suite.handler.GetDepartments(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusOK, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Successfully retrieved", response["messages"])
	assert.Len(suite.T(), response["data"].([]interface{}), 2)
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartments_Empty() {
	suite.mockUsecase.On("GetDepartments").Return([]*entity.Department{}, nil)

	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments", nil)

	suite.handler.GetDepartments(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusOK, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "No data to be displayed", response["message"])
	assert.Len(suite.T(), response["data"].([]interface{}), 0)
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartments_Error() {
	suite.mockUsecase.On("GetDepartments").Return([]*entity.Department{}, errors.New("internal error"))

	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments", nil)

	suite.handler.GetDepartments(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusInternalServerError, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.NotNil(suite.T(), response["error"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartmentByName_Success() {
	name := "Engineering"
	department := &entity.Department{ID: 1, Name: "engineering"}
	suite.mockUsecase.On("GetDepartmentByName", name).Return(department, nil)

	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments?name=Engineering", nil)

	suite.handler.GetDepartmentByName(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusOK, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Successfully retrieved", response["message"])
	assert.Equal(suite.T(), "engineering", response["data"].(map[string]interface{})["name"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartmentByName_EmptyName() {
	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments?name=", nil)

	suite.handler.GetDepartmentByName(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusUnprocessableEntity, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Name can not be empty", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartmentByName_NotFound() {
	name := "Engineering"
	suite.mockUsecase.On("GetDepartmentByName", name).Return((*entity.Department)(nil), customerrors.ErrNotFound)

	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments?name=Engineering", nil)

	suite.handler.GetDepartmentByName(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusNotFound, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "department with the given name is not found", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestGetDepartmentByName_InternalError() {
	name := "Engineering"
	suite.mockUsecase.On("GetDepartmentByName", name).Return((*entity.Department)(nil), errors.New("internal error"))

	suite.ginCtx.Request = httptest.NewRequest(http.MethodGet, "/departments?name=Engineering", nil)

	suite.handler.GetDepartmentByName(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusInternalServerError, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Something went wrong", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestUpdateDepartment_Success() {
	department := &entity.Department{ID: 1, Name: "Engineering"}
	suite.mockUsecase.On("UpdateDepartment", department).Return(department, nil)

	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPut, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.UpdateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusOK, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Succesfully Updated", response["message"])
	assert.Equal(suite.T(), "Engineering", response["data"].(map[string]interface{})["name"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestUpdateDepartment_InvalidJSON() {
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPut, "/departments", bytes.NewBuffer([]byte("invalid json")))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.UpdateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusBadRequest, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Invalid input", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestUpdateDepartment_EmptyName() {
	department := &entity.Department{ID: 1, Name: ""}
	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPut, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.UpdateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusUnprocessableEntity, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Name field is empty", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestUpdateDepartment_MissingID() {
	department := &entity.Department{Name: "Engineering"}
	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPut, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.UpdateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusUnprocessableEntity, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Id field is not provided", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestUpdateDepartment_NotFound() {
	department := &entity.Department{ID: 1, Name: "Engineering"}
	suite.mockUsecase.On("UpdateDepartment", department).Return((*entity.Department)(nil), customerrors.ErrNotFound)

	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPut, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.UpdateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusNotFound, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Department doesn't exist", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestUpdateDepartment_InternalError() {
	department := &entity.Department{ID: 1, Name: "Engineering"}
	suite.mockUsecase.On("UpdateDepartment", department).Return((*entity.Department)(nil), errors.New("internal error"))

	body, _ := json.Marshal(department)
	suite.ginCtx.Request = httptest.NewRequest(http.MethodPut, "/departments", bytes.NewBuffer(body))
	suite.ginCtx.Request.Header.Set("Content-Type", "application/json")

	suite.handler.UpdateDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusInternalServerError, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Couldn't update department", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestDeleteDepartment_Success() {
	depID := uint(1)
	suite.mockUsecase.On("DeleteDepartment", depID).Return(nil)

	suite.ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/departments/1", nil)
	suite.ginCtx.Params = []gin.Param{{Key: "id", Value: "1"}}

	suite.handler.DeleteDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusOK, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Successfully deleted", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestDeleteDepartment_MissingID() {
	suite.ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/departments/", nil)
	suite.ginCtx.Params = []gin.Param{{Key: "id", Value: ""}}

	suite.handler.DeleteDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusBadRequest, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Id not provided", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestDeleteDepartment_InvalidIDFormat() {
	suite.ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/departments/invalid", nil)
	suite.ginCtx.Params = []gin.Param{{Key: "id", Value: "invalid"}}

	suite.handler.DeleteDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusBadRequest, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Invalid id format", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestDeleteDepartment_NonPositiveID() {
	suite.ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/departments/0", nil)
	suite.ginCtx.Params = []gin.Param{{Key: "id", Value: "0"}}

	suite.handler.DeleteDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusUnprocessableEntity, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Id field is not positive", response["message"])
}

func (suite *DepartmentHandlerTestSuite) TestDeleteDepartment_NotFound() {
	depID := uint(1)
	suite.mockUsecase.On("DeleteDepartment", depID).Return(customerrors.ErrNotFound)

	suite.ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/departments/1", nil)
	suite.ginCtx.Params = []gin.Param{{Key: "id", Value: "1"}}

	suite.handler.DeleteDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusNotFound, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "department not found", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}

func (suite *DepartmentHandlerTestSuite) TestDeleteDepartment_InternalError() {
	depID := uint(1)
	suite.mockUsecase.On("DeleteDepartment", depID).Return(errors.New("internal error"))

	suite.ginCtx.Request = httptest.NewRequest(http.MethodDelete, "/departments/1", nil)
	suite.ginCtx.Params = []gin.Param{{Key: "id", Value: "1"}}

	suite.handler.DeleteDepartment(suite.ginCtx)

	assert.Equal(suite.T(), http.StatusInternalServerError, suite.recorder.Code)
	var response map[string]interface{}
	json.Unmarshal(suite.recorder.Body.Bytes(), &response)
	assert.Equal(suite.T(), "Id not found", response["message"])
	suite.mockUsecase.AssertExpectations(suite.T())
}