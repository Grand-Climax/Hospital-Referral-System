package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ReferralHandler struct {
	referralUseCase iusecase.ReferralUseCase
}

func NewReferralHandler(uc iusecase.ReferralUseCase) *ReferralHandler {
	return &ReferralHandler{referralUseCase: uc}
}

// Create godoc
// @Summary      Create Referral
// @Description  Submit a new referral (Draft or Submitted). Roles: REFERRING_DOCTOR, RECEPTIONIST.
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        X-Doctor-ID header string true "Doctor ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        X-Hospital-ID header string true "Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        body body dto.CreateReferralRequest true "Referral payload"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals [post]
func (h *ReferralHandler) Create(c *gin.Context) {
	var req dto.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// For Sprint 4 testing, we extract ID logic headers since we are bypassing complex JWT extraction here
	doctorIDStr := c.GetHeader("X-Doctor-ID")
	hospitalIDStr := c.GetHeader("X-Hospital-ID")

	if doctorIDStr == "" || hospitalIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context headers"})
		return
	}

	doctorID, _ := uuid.Parse(doctorIDStr)
	hospitalID, _ := uuid.Parse(hospitalIDStr)

	referral, err := h.referralUseCase.CreateReferral(c.Request.Context(), doctorID, hospitalID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create referral transaction", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Referral created successfully",
		"data":    referral,
	})
}

// List godoc
// @Summary      List Referrals
// @Description  Get referrals scoped by hospital and role. Doctors see sent referrals; specialists see incoming; admins see all.
// @Tags         Referrals
// @Produce      json
// @Param        status query string false "Filter by Status"
// @Param        date_from query string false "Start Date"
// @Param        date_to query string false "End Date"
// @Success      200 {object} map[string]interface{}
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals [get]
func (h *ReferralHandler) List(c *gin.Context) {
	// Extract secure claims from context (provided by AuthMiddleware)
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")
	hospID, _ := c.Get("hospID")
	deptID, _ := c.Get("deptID")

	uID, _ := userID.(uuid.UUID)
	r, _ := role.(entity.UserRole)
	
	// HospID and DeptID are pointers in the JWT payload
	var hID, dID uuid.UUID
	if h, ok := hospID.(*uuid.UUID); ok && h != nil {
		hID = *h
	}
	if d, ok := deptID.(*uuid.UUID); ok && d != nil {
		dID = *d
	}

	// Forbid Hospital Admin completely based on document rules
	if r == entity.RoleHospitalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hospital Admins are restricted from viewing clinical patient records"})
		return
	}

	status := c.Query("status")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	referrals, err := h.referralUseCase.ListReferrals(c.Request.Context(), uID, hID, dID, r, status, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referrals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": referrals,
	})
}

// GetByID godoc
// @Summary      Get Referral by ID
// @Description  Retrieve full referral details by ID. All authenticated roles with access to the referral can view it.
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id} [get]
func (h *ReferralHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	userID, _ := c.Get("userID")
	role, _ := c.Get("role")
	hospID, _ := c.Get("hospID")
	deptID, _ := c.Get("deptID")

	uID, _ := userID.(uuid.UUID)
	r, _ := role.(entity.UserRole)
	
	var hID, dID uuid.UUID
	if h, ok := hospID.(*uuid.UUID); ok && h != nil {
		hID = *h
	}
	if d, ok := deptID.(*uuid.UUID); ok && d != nil {
		dID = *d
	}

	// Forbid Hospital Admin completely
	if r == entity.RoleHospitalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hospital Admins are restricted from viewing clinical patient records"})
		return
	}

	referral, err := h.referralUseCase.GetReferral(c.Request.Context(), id, uID, hID, dID, r)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": referral})
}

// UpdateDraft godoc
// @Summary      Update Referral Draft
// @Description  Update a DRAFT referral before submission. Only REFERRING_DOCTOR who created it.
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.CreateReferralRequest true "Referral update payload"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id} [put]
func (h *ReferralHandler) UpdateDraft(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req dto.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	uID, _ := userID.(uuid.UUID)

	referral, err := h.referralUseCase.UpdateDraft(c.Request.Context(), id, uID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update draft", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Draft updated successfully",
		"data":    referral,
	})
}

// DeleteDraft godoc
// @Summary      Delete Referral Draft
// @Description  Permanently discard a DRAFT referral. Only REFERRING_DOCTOR before submission.
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id} [delete]
func (h *ReferralHandler) DeleteDraft(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	userID, _ := c.Get("userID")
	uID, _ := userID.(uuid.UUID)

	if err := h.referralUseCase.DeleteDraft(c.Request.Context(), id, uID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete draft", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Draft deleted successfully"})
}

// UpdateStatus godoc
// @Summary      Update Referral Status
// @Description  Advance or reverse the referral state machine. Transitions vary by role: LIAISON_OFFICER can FORWARD or set NEEDS_REVISION; RECEIVING_SPECIALIST can accept (SPECIALIST_ASSIGNED) or reject back to UNDER_LIAISON_REVIEW; REFERRING_DOCTOR resubmits NEEDS_REVISION → UNDER_LIAISON_REVIEW.
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        X-Doctor-ID header string true "Doctor ID deciding" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        body body object true "Status Payload"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/status [patch]
func (h *ReferralHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status payload", "details": err.Error()})
		return
	}

	// Assuming we extract the user ID deciding the change from the JWT/Headers
	userIDStr := c.GetHeader("X-Doctor-ID")
	userID, _ := uuid.Parse(userIDStr)

	if err := h.referralUseCase.UpdateReferralStatus(c.Request.Context(), id, entity.ReferralStatus(req.Status), userID, req.Reason); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Status transition failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Referral status updated to " + req.Status})
}
