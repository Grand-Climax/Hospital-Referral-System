package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type MohAnalyticsHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewMohAnalyticsHandler(referralUC iusecase.ReferralUseCase) *MohAnalyticsHandler {
	return &MohAnalyticsHandler{referralUC: referralUC}
}

func parseMohAnalyticsFilter(c *gin.Context) (irepository.MohAnalyticsFilter, bool) {
	filter := irepository.MohAnalyticsFilter{}

	if from := strings.TrimSpace(c.Query("from")); from != "" {
		fromDate, err := time.Parse("2006-01-02", from)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid from date format, use YYYY-MM-DD"})
			return filter, false
		}
		filter.From = &fromDate
	}

	if to := strings.TrimSpace(c.Query("to")); to != "" {
		toDate, err := time.Parse("2006-01-02", to)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid to date format, use YYYY-MM-DD"})
			return filter, false
		}
		// Inclusive day range for date-only input.
		endOfDay := toDate.Add(24*time.Hour - time.Nanosecond)
		filter.To = &endOfDay
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "from date must be before or equal to to date"})
		return filter, false
	}

	if region := strings.TrimSpace(c.Query("region")); region != "" {
		filter.Region = &region
	}

	if hospitalID := strings.TrimSpace(c.Query("hospital_id")); hospitalID != "" {
		id, err := uuid.Parse(hospitalID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid hospital_id"})
			return filter, false
		}
		filter.HospitalID = &id
	}

	return filter, true
}

// GetDashboardSummary godoc
// @Summary      MoH dashboard summary
// @Description  Aggregated MoH KPI metrics for referrals in the selected range.
// @Description  **Roles:** MOH_ANALYST
// @Description  **Privacy:** Aggregate-only response, no patient PII.
// @Tags         MoH Analytics
// @Produce      json
// @Param        from query string false "Start date (YYYY-MM-DD)"
// @Param        to query string false "End date (YYYY-MM-DD)"
// @Param        region query string false "Patient home region filter"
// @Param        hospital_id query string false "Hospital ID filter"
// @Success      200 {object} dto.MohDashboardSummaryResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/moh/dashboard/summary [get]
func (h *MohAnalyticsHandler) GetDashboardSummary(c *gin.Context) {
	filter, ok := parseMohAnalyticsFilter(c)
	if !ok {
		return
	}

	summary, err := h.referralUC.GetMohDashboardSummary(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MohDashboardSummaryResponse{
		TotalReferrals:        summary.TotalReferrals,
		TotalAccepted:         summary.TotalAccepted,
		TotalRejected:         summary.TotalRejected,
		TotalAdmitted:         summary.TotalAdmitted,
		AcceptanceRate:        summary.AcceptanceRate,
		AverageMLSeverity:     summary.AverageMLSeverity,
		AverageTurnaroundHour: summary.AverageTurnaroundHr,
		BaseResponse:          dto.BaseResponse{Success: true, Message: "MoH dashboard summary retrieved successfully"},
	})
}

// GetReferralTrends godoc
// @Summary      MoH referral trends
// @Description  Time-series referral trends grouped by day/week/month.
// @Description  **Roles:** MOH_ANALYST
// @Description  **Privacy:** Aggregate-only response, no patient PII.
// @Tags         MoH Analytics
// @Produce      json
// @Param        from query string false "Start date (YYYY-MM-DD)"
// @Param        to query string false "End date (YYYY-MM-DD)"
// @Param        region query string false "Patient home region filter"
// @Param        hospital_id query string false "Hospital ID filter"
// @Param        granularity query string false "day, week, month" default(month)
// @Success      200 {object} dto.MohReferralTrendsResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/moh/referral-trends [get]
func (h *MohAnalyticsHandler) GetReferralTrends(c *gin.Context) {
	filter, ok := parseMohAnalyticsFilter(c)
	if !ok {
		return
	}

	granularity := strings.ToLower(strings.TrimSpace(c.DefaultQuery("granularity", "month")))
	if granularity != "day" && granularity != "week" && granularity != "month" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid granularity; allowed values are day, week, month"})
		return
	}

	points, err := h.referralUC.GetMohReferralTrends(c.Request.Context(), filter, granularity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.MohReferralTrendPointResponse, 0, len(points))
	for _, p := range points {
		resp = append(resp, dto.MohReferralTrendPointResponse{
			Period:             p.Period,
			TotalReferrals:     p.TotalReferrals,
			AcceptedReferrals:  p.AcceptedReferrals,
			RejectedReferrals:  p.RejectedReferrals,
			EmergencyReferrals: p.EmergencyReferrals,
		})
	}

	c.JSON(http.StatusOK, dto.MohReferralTrendsResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "MoH referral trends retrieved successfully"},
	})
}

// GetHospitalLoad godoc
// @Summary      MoH hospital load analytics
// @Description  Per-hospital referral load and performance metrics.
// @Description  **Roles:** MOH_ANALYST
// @Description  **Privacy:** Aggregate-only response, no patient PII.
// @Tags         MoH Analytics
// @Produce      json
// @Param        from query string false "Start date (YYYY-MM-DD)"
// @Param        to query string false "End date (YYYY-MM-DD)"
// @Param        region query string false "Hospital region filter"
// @Param        hospital_id query string false "Hospital ID filter"
// @Param        tier_level query string false "Hospital tier level (PRIMARY, SECONDARY, SPECIALIZED, TERTIARY)"
// @Success      200 {object} dto.MohHospitalLoadResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/moh/hospital-load [get]
func (h *MohAnalyticsHandler) GetHospitalLoad(c *gin.Context) {
	filter, ok := parseMohAnalyticsFilter(c)
	if !ok {
		return
	}

	if tier := strings.ToUpper(strings.TrimSpace(c.Query("tier_level"))); tier != "" {
		t := entity.HospitalTier(tier)
		switch t {
		case entity.PrimaryHosp, entity.SecondaryHosp, entity.SpecializedHosp, entity.TertiaryHosp:
			filter.TierLevel = &t
		default:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid tier_level"})
			return
		}
	}

	rows, err := h.referralUC.GetMohHospitalLoad(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.MohHospitalLoadMetricResponse, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, dto.MohHospitalLoadMetricResponse{
			HospitalID:             rw.HospitalID.String(),
			HospitalName:           rw.HospitalName,
			TierLevel:              string(rw.TierLevel),
			Region:                 rw.Region,
			TotalReferralsReceived: rw.TotalReceived,
			TotalAccepted:          rw.TotalAccepted,
			TotalRejected:          rw.TotalRejected,
			RejectionRate:          rw.RejectionRate,
			AverageSeverityScore:   rw.AverageSeverity,
		})
	}

	c.JSON(http.StatusOK, dto.MohHospitalLoadResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "MoH hospital load metrics retrieved successfully"},
	})
}

// GetDiseaseHotspots godoc
// @Summary      MoH disease hotspots
// @Description  Geographic referral concentration grouped by patient region and department specialty.
// @Description  **Roles:** MOH_ANALYST
// @Description  **Privacy:** Aggregate-only response, no patient PII.
// @Tags         MoH Analytics
// @Produce      json
// @Param        from query string false "Start date (YYYY-MM-DD)"
// @Param        to query string false "End date (YYYY-MM-DD)"
// @Param        region query string false "Patient home region filter"
// @Param        hospital_id query string false "Hospital ID filter"
// @Success      200 {object} dto.MohDiseaseHotspotsResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/moh/disease-hotspots [get]
func (h *MohAnalyticsHandler) GetDiseaseHotspots(c *gin.Context) {
	filter, ok := parseMohAnalyticsFilter(c)
	if !ok {
		return
	}

	rows, err := h.referralUC.GetMohDiseaseHotspots(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.MohDiseaseHotspotResponseItem, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, dto.MohDiseaseHotspotResponseItem{
			Region:               rw.Region,
			DepartmentName:       rw.DepartmentName,
			ReferralCount:        rw.ReferralCount,
			AverageSeverityScore: rw.AverageSeverity,
		})
	}

	c.JSON(http.StatusOK, dto.MohDiseaseHotspotsResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "MoH disease hotspots retrieved successfully"},
	})
}

// GetSeverityDistribution godoc
// @Summary      MoH severity distribution
// @Description  Severity-tier distribution (critical, urgent, routine) grouped by patient region.
// @Description  **Roles:** MOH_ANALYST
// @Description  **Privacy:** Aggregate-only response, no patient PII.
// @Tags         MoH Analytics
// @Produce      json
// @Param        from query string false "Start date (YYYY-MM-DD)"
// @Param        to query string false "End date (YYYY-MM-DD)"
// @Param        region query string false "Patient home region filter"
// @Param        hospital_id query string false "Hospital ID filter"
// @Success      200 {object} dto.MohSeverityDistributionResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/moh/severity-distribution [get]
func (h *MohAnalyticsHandler) GetSeverityDistribution(c *gin.Context) {
	filter, ok := parseMohAnalyticsFilter(c)
	if !ok {
		return
	}

	rows, err := h.referralUC.GetMohSeverityDistribution(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.MohSeverityDistributionResponseItem, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, dto.MohSeverityDistributionResponseItem{
			Region:         rw.Region,
			CriticalCount:  rw.CriticalCount,
			UrgentCount:    rw.UrgentCount,
			RoutineCount:   rw.RoutineCount,
			TotalReferrals: rw.TotalReferrals,
		})
	}

	c.JSON(http.StatusOK, dto.MohSeverityDistributionResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "MoH severity distribution retrieved successfully"},
	})
}

// ExportReport godoc
// @Summary      Export MoH analytics report
// @Description  Combined aggregate report of dashboard summary and hospital load metrics.
// @Description  **Roles:** MOH_ANALYST
// @Description  **Privacy:** Aggregate-only response, no patient PII.
// @Tags         MoH Analytics
// @Produce      json
// @Param        from query string false "Start date (YYYY-MM-DD)"
// @Param        to query string false "End date (YYYY-MM-DD)"
// @Param        region query string false "Region filter"
// @Param        hospital_id query string false "Hospital ID filter"
// @Param        tier_level query string false "Hospital tier level (PRIMARY, SECONDARY, SPECIALIZED, TERTIARY)"
// @Success      200 {object} dto.MohAnalyticsExportResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/moh/reports/export [get]
func (h *MohAnalyticsHandler) ExportReport(c *gin.Context) {
	filter, ok := parseMohAnalyticsFilter(c)
	if !ok {
		return
	}

	if tier := strings.ToUpper(strings.TrimSpace(c.Query("tier_level"))); tier != "" {
		t := entity.HospitalTier(tier)
		switch t {
		case entity.PrimaryHosp, entity.SecondaryHosp, entity.SpecializedHosp, entity.TertiaryHosp:
			filter.TierLevel = &t
		default:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid tier_level"})
			return
		}
	}

	summary, err := h.referralUC.GetMohDashboardSummary(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	loadRows, err := h.referralUC.GetMohHospitalLoad(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	loadResp := make([]dto.MohHospitalLoadMetricResponse, 0, len(loadRows))
	for _, rw := range loadRows {
		loadResp = append(loadResp, dto.MohHospitalLoadMetricResponse{
			HospitalID:             rw.HospitalID.String(),
			HospitalName:           rw.HospitalName,
			TierLevel:              string(rw.TierLevel),
			Region:                 rw.Region,
			TotalReferralsReceived: rw.TotalReceived,
			TotalAccepted:          rw.TotalAccepted,
			TotalRejected:          rw.TotalRejected,
			RejectionRate:          rw.RejectionRate,
			AverageSeverityScore:   rw.AverageSeverity,
		})
	}

	c.JSON(http.StatusOK, dto.MohAnalyticsExportResponse{
		Summary: dto.MohDashboardSummaryResponse{
			TotalReferrals:        summary.TotalReferrals,
			TotalAccepted:         summary.TotalAccepted,
			TotalRejected:         summary.TotalRejected,
			TotalAdmitted:         summary.TotalAdmitted,
			AcceptanceRate:        summary.AcceptanceRate,
			AverageMLSeverity:     summary.AverageMLSeverity,
			AverageTurnaroundHour: summary.AverageTurnaroundHr,
		},
		HospitalLoad: loadResp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "MoH analytics export generated successfully"},
	})
}
