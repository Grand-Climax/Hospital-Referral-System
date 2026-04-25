package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ReceptionistHandler struct {
	referralUC iusecase.ReferralUseCase
	arrivalUC  iusecase.ArrivalUseCase
}

func NewReceptionistHandler(referralUC iusecase.ReferralUseCase, arrivalUC iusecase.ArrivalUseCase) *ReceptionistHandler {
	return &ReceptionistHandler{
		referralUC: referralUC,
		arrivalUC:  arrivalUC,
	}
}

func (h *ReceptionistHandler) getHospitalAndDept(c *gin.Context) (uuid.UUID, uuid.UUID) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	} else if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	}

	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	} else if dID, ok := deptIdVal.(uuid.UUID); ok {
		deptID = dID
	}

	return hospID, deptID
}

func (h *ReceptionistHandler) ListReferrals(c *gin.Context) {
	hospID, _ := h.getHospitalAndDept(c)
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid hospital scope"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	filter := irepository.ReferralFilter{
		Status:      c.Query("status"),
		Region:      c.Query("region"),
		PatientName: c.Query("patient_name"),
		Sort:        c.Query("sort"),
		Limit:       limit,
		Page:        page,
	}

	referrals, total, err := h.referralUC.ListForReceptionist(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referrals retrieved successfully"},
		Data:         toListReferralResponseSlice(referrals),
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

func (h *ReceptionistHandler) GetReferral(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	hospID, _ := h.getHospitalAndDept(c)
	ref, err := h.referralUC.GetDetailsForReceptionist(c.Request.Context(), id, hospID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ReferralDetailResponse{
		Referral: *ref,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referral details retrieved successfully"},
	})
}

func (h *ReceptionistHandler) GetSchedule(c *gin.Context) {
	hospID, deptID := h.getHospitalAndDept(c)
	if hospID == uuid.Nil || deptID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes (hospital/department missing)"})
		return
	}

	schedules, err := h.arrivalUC.GetTodayAndTomorrowSchedule(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schedules,
	})
}

func (h *ReceptionistHandler) ConfirmArrival(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	if err := h.arrivalUC.ConfirmArrival(c.Request.Context(), queueID, userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient arrival confirmed"})
}

func (h *ReceptionistHandler) AssignDoctor(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	var req dto.AssignDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	if err := h.arrivalUC.AssignDoctor(c.Request.Context(), queueID, req.DoctorID, userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Doctor assigned successfully"})
}

func (h *ReceptionistHandler) RegisterWalkIn(c *gin.Context) {
	var req dto.WalkInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	hospID, deptID := h.getHospitalAndDept(c)
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	queue, err := h.arrivalUC.RegisterWalkIn(c.Request.Context(), req.ReferralID, hospID, deptID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Walk-in registered successfully",
		"data":    queue,
	})
}

func (h *ReceptionistHandler) MarkMissed(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	var req dto.MarkMissedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	if err := h.arrivalUC.MarkMissed(c.Request.Context(), queueID, entity.MissReason(req.MissReason), userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient marked as missed"})
}
