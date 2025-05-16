package handlertest

import (
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	mockusecase "Hospital-Referral-System/internal/test/mocks/mock_usecase"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateRole(t *testing.T) {
	mockRoleUsecase := new(mockusecase.MockRoleUsecase)
	handler := handlers.NewRoleHandler(mockRoleUsecase)

	//Setup router
	router := gin.Default()
	router.POST("/roles", handler.CreateRole)

	//declare input type and output type
	inputRole := &entity.Role{Name: "Admin"}
	expectedRole := &entity.Role{ID: 1, Name: "Admin"}

	mockRoleUsecase.On("CreateRole", mock.AnythingOfType("*entity.Role")).Return(expectedRole,nil)
	
	// Convert input to JSON
	jsonBody, _ := json.Marshal(inputRole)
	req, _ := http.NewRequest(http.MethodPost, "/roles", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	//record the response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	//assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Successfully created", response["message"])
	assert.Equal(t, "Admin", response["data"].(map[string]interface{})["name"])

	mockRoleUsecase.AssertExpectations(t)

}

func TestGetRole(t *testing.T) {
	//create a mock usecase
	mockRoleusecase := new(mockusecase.MockRoleUsecase)
	handler := handlers.NewRoleHandler(mockRoleusecase)

	//Set up router
	router := gin.Default()
	router.GET("/roles", handler.GetRole)

	//declare expected output
	expectedRoles := []*entity.Role{
		{ID: 1, Name: "Admin"},
		{ID: 2, Name: "Doctor"},
	}
	mockRoleusecase.On("GetRole").Return(expectedRoles)

	//send request using http
	req, _ := http.NewRequest(http.MethodGet, "/roles",nil)
	req.Header.Set("Content-Type","application/json")

	//record the response
	w := httptest.NewRecorder()
	router.ServeHTTP(w,req)

	//assert
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(),&response)

	assert.NoError(t, err)
	assert.Equal(t, "Successfully retrieved", response["message"])

	//validate data
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
	assert.Equal(t, "Admin", data[0].(map[string]interface{})["name"])
	assert.Equal(t, "Doctor", data[1].(map[string]interface{})["name"])

	mockRoleusecase.AssertExpectations(t)

}

func TestUpdateRole(t *testing.T) {
	//mock usecase
	mockRoleusecase := new(mockusecase.MockRoleUsecase)
	handler := handlers.NewRoleHandler(mockRoleusecase)

	//setup router
	router := gin.Default()
	router.PUT("/roles", handler.UpdateRole)

	//declare input and expected output
	inputRole := &entity.Role{Name: "Doctor"}
	expectedRole := &entity.Role{ID: 1, Name: "Doctor"}

	mockRoleusecase.On("UpdateRole", mock.AnythingOfType("*entity.Role")).Return(expectedRole,nil)

	//convert the input role into json
	jsonBody, _ := json.Marshal(inputRole)
	req, _ := http.NewRequest(http.MethodPut,"/roles",bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type","application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	//convert the json response back
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(),&response)

	//assert
	assert.NoError(t, err)
	assert.Equal(t, "Successfully Updated", response["message"])
	assert.Equal(t, "Doctor", response["data"].(map[string]interface{})["name"])

	mockRoleusecase.AssertExpectations(t)
}

func TestDeleteRole(t *testing.T) {
	type testCase struct {
		name              string
		roleID            string
		expectUsecaseCall bool
		mockUsecaseError  error
		expectedStatus    int
		expectedMessage   string
	}

	tests := []testCase{
		{
			name:              "Success",
			roleID:            "1",
			expectUsecaseCall: true,
			mockUsecaseError:  nil,
			expectedStatus:    http.StatusOK,
			expectedMessage:   "Successfully Deleted",
		},
		{
			name:              "Invalid ID Format",
			roleID:            "abc",
			expectUsecaseCall: false,
			expectedStatus:    http.StatusBadRequest,
			expectedMessage:   "Invalid role id format",
		},
		{
			name:              "Zero ID",
			roleID:            "0",
			expectUsecaseCall: false,
			expectedStatus:    http.StatusBadRequest,
			expectedMessage:   "id should be positive integer",
		},
		{
			name:              "Usecase Error",
			roleID:            "2",
			expectUsecaseCall: true,
			mockUsecaseError:  errors.New("deletion failed"),
			expectedStatus:    http.StatusInternalServerError,
			expectedMessage:   "Couldn't delete role with the given id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			// Mock and handler
			mockRoleUsecase := new(mockusecase.MockRoleUsecase)
			handler := handlers.NewRoleHandler(mockRoleUsecase)

			// Setup router
			router := gin.Default()
			router.DELETE("/roles/:id", handler.DeleteRole)

			// Conditionally set expectation
			if tc.expectUsecaseCall {
				idNum, _ := strconv.Atoi(tc.roleID)
				mockRoleUsecase.On("DeleteRole", uint(idNum)).Return(tc.mockUsecaseError)
			}

			// Create test request
			req, _ := http.NewRequest(http.MethodDelete, "/roles/"+tc.roleID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedMessage)

			mockRoleUsecase.AssertExpectations(t)
		})
	}
}
