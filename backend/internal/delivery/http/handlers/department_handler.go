package handlers

import (
	"Hospital-Referral-System/internal/domain/entity"
	customerrors "Hospital-Referral-System/internal/errors"
	usecase "Hospital-Referral-System/internal/domain/contract/usecase_interface"
	"errors"
	"strconv"
	"strings"

	"net/http"

	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	depusecase usecase.DepartmentUsecaseInterface
}

func NewDepartmentHandler(usecase usecase.DepartmentUsecaseInterface) *DepartmentHandler{
	return &DepartmentHandler{depusecase: usecase}
}

func (handler *DepartmentHandler) CreateDepartment(ctx *gin.Context) {
	var department entity.Department
	if err := ctx.ShouldBindJSON(&department); err != nil {
		RespondErrors(ctx, http.StatusBadRequest, "Invalid Input", err)
		return
	}
	if department.Name == "" {
		Respond(ctx, http.StatusUnprocessableEntity, "Name field can not be empty")
		return
	}

	department.Name = strings.ToLower(department.Name)
	created, err := handler.depusecase.CreateDepartment(&department)
	if err != nil {
		if errors.Is(err, customerrors.ErrDuplicate){
			ctx.IndentedJSON(http.StatusConflict, gin.H{"message": "department name already exists can't create a department with the given name", "error": err})
			return
		}
		ctx.IndentedJSON(http.StatusInternalServerError,gin.H{"message": "Something went wrong", "error": err})
		return
	}

	ctx.IndentedJSON(http.StatusCreated, gin.H{"message":"Successfully Created", "data": created})
}

func (handler *DepartmentHandler) GetDepartments(ctx *gin.Context) {
	departments, err := handler.depusecase.GetDepartments()
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if len(departments) == 0{
		ctx.IndentedJSON(http.StatusOK, gin.H{"message": "No data to be displayed", "data": departments})
		return
	}
	ctx.IndentedJSON(http.StatusOK, gin.H{"messages": "Successfully retrieved", "data": departments})
}

func (handler *DepartmentHandler) GetDepartmentByName(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		ctx.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"message": "Name can not be empty", "error": "empty field"})
		return
	}
	name = strings.ToLower(name)
	dep, err := handler.depusecase.GetDepartmentByName(name)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "department with the given name is not found", "error": err})
			return
		}

		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong", "error": err})
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully retrieved", "data": dep})
}
func (handler *DepartmentHandler) UpdateDepartment(ctx *gin.Context) {
	var department entity.Department
	if err := ctx.ShouldBindJSON(&department); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid input", "error": err})
		return
	}

	if department.Name == ""{
		ctx.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"message": "Name field is empty", "error": "name cannot be empty"})
		return
	}

	if department.ID == 0 {
		ctx.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"message": "Id field is not provided", "error": "Id is required"})
		return
	}
	
	updatedDepartment, err := handler.depusecase.UpdateDepartment(&department)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound){
			ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "Department doesn't exist", "error": err})
			return
		}

		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldn't update department", "error": err})
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Succesfully Updated", "data": updatedDepartment})
}

func (handler *DepartmentHandler) DeleteDepartment(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Id not provided", "error": "Id field cannot be empty"})
		return
	}
	dep_id, err := strconv.Atoi(id)
	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid id format", "error": err})
		return
	}

	if dep_id <= 0{
		ctx.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"message": "Id field is not positive", "error": "Id field cannot be non positive"})
		return
	}
	err = handler.depusecase.DeleteDepartment(uint(dep_id))
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound){
			ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "department not found", "error": err})
			return
		}
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Id not found", "error": err})
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully deleted"})
}