package handlers

import (
	usecase "Hospital-Referral-System/internal/domain/contract/usecase_interface"
	"Hospital-Referral-System/internal/domain/entity"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleusecase usecase.RoleUsecaseInterface
}

func NewRoleHandler(roleusecase usecase.RoleUsecaseInterface) *RoleHandler {
	return &RoleHandler{roleusecase: roleusecase}
}

func (handler *RoleHandler) CreateRole(ctx *gin.Context) {
	var role entity.Role

	if err := ctx.ShouldBindJSON(&role); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdRole, err := handler.roleusecase.CreateRole(&role)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"+ err.Error()})
		return
	}

	ctx.IndentedJSON(http.StatusCreated, gin.H{"message": "Successfully created", "data": createdRole})

}

func (handler *RoleHandler) GetRole(ctx *gin.Context) {
	roles := handler.roleusecase.GetRole()
	
	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully retrieved", "data": roles})
}

func (handler *RoleHandler) UpdateRole(ctx *gin.Context) {
	var role entity.Role

	if err := ctx.ShouldBindJSON(&role); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid user input", "error": err.Error()})
		return
	}

	updatedRole, err := handler.roleusecase.UpdateRole(&role)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldn't update role", "error": err.Error()})
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully Updated", "data": updatedRole})
}

func (handler *RoleHandler) DeleteRole(ctx *gin.Context) {
	id := ctx.Param("id")

	role_id, err := strconv.Atoi(id)

	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid role id format", "error": err.Error()})
		return
	}

	if role_id <= 0 {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "id should be positive integer"})
		return
	}

	err = handler.roleusecase.DeleteRole(uint(role_id))
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Couldn't delete role with the given id", "error": err.Error()})
		return
	}

	ctx.IndentedJSON(http.StatusOK, gin.H{"message": "Successfully Deleted"})

}