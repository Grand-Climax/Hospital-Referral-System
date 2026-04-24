package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type TriageHandler struct {
	triageUC iusecase.TriageUseCase
}

func NewTriageHandler(triageUC iusecase.TriageUseCase) *TriageHandler {
	return &TriageHandler{triageUC: triageUC}
}

func (h *TriageHandler) ListForTriage(c *gin.Context) {
	hospitalID, _ := uuid.Parse(c.GetString("hospital_id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	queues, count, err := h.triageUC.ListForTriage(c.Request.Context(), hospitalID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    queues,
		"total":   count,
	})
}

func (h *TriageHandler) Review(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid referral ID"})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.TriageReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	if err := h.triageUC.ReviewTriage(c.Request.Context(), id, userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Triage reviewed successfully"})
}
