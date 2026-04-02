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

	// Extract securely from Context populated by Auth middleware
	userIDVal, _ := c.Get("userID")
	hospIDVal, _ := c.Get("hospID")

	doctorID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid user ID format"})
		return
	}

	hospIDPtr, _ := hospIDVal.(*uuid.UUID)
	if hospIDPtr == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: User is not linked to a Sender Hospital"})
		return
	}

	hospitalID := *hospIDPtr

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

// Submit godoc
// @Summary      Submit Referral Draft
// @Description  Transition referral from DRAFT to SUBMITTED state. Enforces clinical validation. Only creator.
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/submit [post]
func (h *ReferralHandler) Submit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.referralUseCase.SubmitReferral(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to submit referral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Referral submitted successfully",
		"newStatus": entity.StatusSubmitted,
	})
}

// Resubmit godoc
// @Summary      Resubmit Rejected Referral
// @Description  Transition referral from NEEDS_REVISION to UNDER_LIAISON_REVIEW. Clears old rejection reason. Only creator.
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/resubmit [post]
func (h *ReferralHandler) Resubmit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	userIDVal, _ := c.Get("userID")
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.referralUseCase.ResubmitReferral(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to resubmit referral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Referral resubmitted successfully",
		"newStatus": entity.StatusUnderLiaisonReview,
	})
}

// Cancel godoc
// @Summary      Cancel Referral
// @Description  Transition referral to CANCELLED state. Available from DRAFT or NEEDS_REVISION.
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body object false "Cancel Payload (reason optional)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/cancel [post]
func (h *ReferralHandler) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	// Note: ShouldBindJSON is okay here if body is empty it returns error, but we can safely ignore it to make it optional.
	_ = c.ShouldBindJSON(&req)

	userIDVal, _ := c.Get("userID")
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.referralUseCase.CancelReferral(c.Request.Context(), id, userID, req.Reason); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to cancel referral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Referral cancelled successfully",
		"newStatus": entity.StatusCancelled,
	})
}
