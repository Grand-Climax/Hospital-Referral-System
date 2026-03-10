package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/usecase"
)

type ReferenceHandler struct {
	referenceUseCase usecase.ReferenceUseCase
}

func NewReferenceHandler(uc usecase.ReferenceUseCase) *ReferenceHandler {
	return &ReferenceHandler{referenceUseCase: uc}
}

func (h *ReferenceHandler) GetHospitals(c *gin.Context) {
	tier := c.Query("tier")
	hospitals, err := h.referenceUseCase.GetHospitals(c.Request.Context(), tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hospitals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hospitals})
}

func (h *ReferenceHandler) GetDepartments(c *gin.Context) {
	depts, err := h.referenceUseCase.GetDepartments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": depts})
}

func (h *ReferenceHandler) SearchICD(c *gin.Context) {
	q := c.Query("q")
	codes, err := h.referenceUseCase.SearchICDCodes(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search ICD codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": codes})
}

func (h *ReferenceHandler) GetNetworkedHospitals(c *gin.Context) {
	senderHospitalID, err := uuid.Parse(c.GetHeader("X-Hospital-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid requesting hospital ID"})
		return
	}

	hospitals, err := h.referenceUseCase.GetNetworkedHospitals(c.Request.Context(), senderHospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch networked hospitals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hospitals})
}

func (h *ReferenceHandler) GetHospitalDepartments(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target hospital ID format"})
		return
	}

	depts, err := h.referenceUseCase.GetHospitalDepartments(c.Request.Context(), hospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hospital departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": depts})
}
