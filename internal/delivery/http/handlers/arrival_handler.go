package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ArrivalHandler struct {
	arrivalUC iusecase.ArrivalUseCase
}

func NewArrivalHandler(arrivalUC iusecase.ArrivalUseCase) *ArrivalHandler {
	return &ArrivalHandler{arrivalUC: arrivalUC}
}

func (h *ArrivalHandler) ConfirmArrival(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid referral ID"})
		return
	}

	receptionistID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.arrivalUC.ConfirmArrival(c.Request.Context(), id, receptionistID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Arrival confirmed successfully"})
}
