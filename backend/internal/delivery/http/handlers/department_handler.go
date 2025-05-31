package handlers

import (
	"Hospital-Referral-System/internal/domain/entity"
	customerrors "Hospital-Referral-System/internal/errors"
	usecase "Hospital-Referral-System/internal/domain/contract/usecase_interface"
	"errors"
	"strconv"

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

	created, err := handler.depusecase.CreateDepartment(&department)
	if err != nil {
		if errors.Is(err, customerrors.ErrDuplicate){
			RespondErrors(ctx, http.StatusConflict, "department name already exists can't create a department with the given name", err)
			return
		}
		RespondErrors(ctx, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	ctx.IndentedJSON(http.StatusCreated, gin.H{"message":"Successfully Created", "data": created})
}

func (handler *DepartmentHandler) GetDepartments(ctx *gin.Context) {
	departments, err := handler.depusecase.GetDepartments()
	if err != nil {
		RespondErrors(ctx, http.StatusInternalServerError, "Something Went wrong", err)
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
		Respond(ctx, http.StatusUnprocessableEntity, "Name can not be empty")
		return
	}
	dep, err := handler.depusecase.GetDepartmentByName(name)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			RespondErrors(ctx, http.StatusNotFound, "department with the given name is not found", err)
			return
		}
		RespondErrors(ctx, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully retrieved", "data": dep})
}
func (handler *DepartmentHandler) UpdateDepartment(ctx *gin.Context) {
	var department entity.Department
	if err := ctx.ShouldBindJSON(&department); err != nil {
		RespondErrors(ctx, http.StatusBadRequest, "Invalid input", err)
		return
	}

	if department.Name == ""{
		Respond(ctx, http.StatusUnprocessableEntity, "Name field is empty")
		return
	}

	if department.ID == 0 {
		Respond(ctx, http.StatusUnprocessableEntity, "Id field is not provided")
		return
	}
	
	updatedDepartment, err := handler.depusecase.UpdateDepartment(&department)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound){
			RespondErrors(ctx, http.StatusNotFound, "Department doesn't exist", err)
			return
		}
		RespondErrors(ctx, http.StatusInternalServerError, "Couldn't update department", err)
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Succesfully Updated", "data": updatedDepartment})
}

func (handler *DepartmentHandler) DeleteDepartment(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Respond(ctx, http.StatusBadRequest, "Id not provided")
		return
	}
	dep_id, err := strconv.Atoi(id)
	if err != nil {
		RespondErrors(ctx, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	if dep_id <= 0{
		Respond(ctx, http.StatusUnprocessableEntity, "Id field is not positive")
		return
	}
	err = handler.depusecase.DeleteDepartment(uint(dep_id))
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound){
			RespondErrors(ctx, http.StatusNotFound, "department not found", err)
			return
		}
		RespondErrors(ctx, http.StatusInternalServerError, "Id not found", err)
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully deleted"})
}