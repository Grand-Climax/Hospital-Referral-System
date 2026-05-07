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

// ListForTriage godoc
// @Summary      List Triage Queue
// @Description  Returns a prioritized list of patients waiting for triage for the specialist's hospital.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Sorting:** Severity score descending, waiting time ascending.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        limit query int false "Pagination limit" default(10)
// @Param        offset query int false "Pagination offset" default(0)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/triage [get]
func (h *TriageHandler) ListForTriage(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	} else if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	queues, count, err := h.triageUC.ListForTriage(c.Request.Context(), hospID, limit, offset)
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

// Review godoc
// @Summary      Review Triage Prediction
// @Description  Allows a specialist to review and finalize the ML-generated triage score (APPROVE/OVERRIDE/REJECT).
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** Referral must have an ML prediction or manual score set.
// @Description  **State Transition:** Finalizes the triage state for the referral.
// @Description  **Common Errors:**
// @Description  - 400 invalid format or input
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.TriageReviewRequest true "Review details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/triage-review [post]
func (h *TriageHandler) Review(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid referral ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

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
