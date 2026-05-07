package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type InAppNotificationHandler struct {
	notifUC iusecase.InAppNotificationUseCase
}

func NewInAppNotificationHandler(notifUC iusecase.InAppNotificationUseCase) *InAppNotificationHandler {
	return &InAppNotificationHandler{
		notifUC: notifUC,
	}
}

// ListNotifications godoc
// @Summary      List My In-App Notifications
// @Description  Get a paginated list of notifications for the current authenticated user with filters.
// @Tags         In-App Notifications
// @Produce      json
// @Param        limit        query int    false "Pagination limit" default(20)
// @Param        page         query int    false "Page number" default(1)
// @Param        event_type   query string false "Filter by event type"
// @Param        is_read      query bool   false "Filter by read status"
// @Param        referral_id  query string false "Filter by referral ID"
// @Param        start_date   query string false "Filter by start date (YYYY-MM-DD)"
// @Param        end_date     query string false "Filter by end date (YYYY-MM-DD)"
// @Param        search       query string false "Search in title and message"
// @Success      200 {object} dto.PaginatedNotificationResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/me/notifications [get]
func (h *InAppNotificationHandler) ListNotifications(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	filter := irepository.InAppNotificationFilter{
		EventType: c.Query("event_type"),
		Search:    c.Query("search"),
	}

	if isReadStr := c.Query("is_read"); isReadStr != "" {
		isRead, err := strconv.ParseBool(isReadStr)
		if err == nil {
			filter.IsRead = &isRead
		}
	}

	if refIdStr := c.Query("referral_id"); refIdStr != "" {
		refID, err := uuid.Parse(refIdStr)
		if err == nil {
			filter.ReferralID = &refID
		}
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			filter.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			// Set to end of day
			endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.EndDate = &endDate
		}
	}

	notifications, total, unreadCount, err := h.notifUC.ListForUser(c.Request.Context(), userID, filter, limit, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedNotificationResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No notifications found",
			},
			Data:        []dto.InAppNotificationResponse{},
			Total:       0,
			UnreadCount: unreadCount,
			Page:        page,
			PageSize:    limit,
		})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedNotificationResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Notifications retrieved successfully",
		},
		Data:        dto.ToInAppNotificationResponseSlice(notifications),
		Total:       total,
		UnreadCount: unreadCount,
		Page:        page,
		PageSize:    limit,
	})
}

// MarkRead godoc
// @Summary      Mark Notification as Read
// @Description  Mark a specific in-app notification as read.
// @Tags         In-App Notifications
// @Produce      json
// @Param        id path string true "Notification ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/me/notifications/{id}/read [post]
func (h *InAppNotificationHandler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid notification ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	if err := h.notifUC.MarkRead(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Notification marked as read"})
}

// MarkAllRead godoc
// @Summary      Mark All Notifications as Read
// @Description  Mark all unread in-app notifications as read for the current user.
// @Tags         In-App Notifications
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/me/notifications/read-all [post]
func (h *InAppNotificationHandler) MarkAllRead(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	if err := h.notifUC.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "All notifications marked as read"})
}

// GetUnreadCount godoc
// @Summary      Get Unread Notification Count
// @Description  Get the count of unread notifications for the current user.
// @Tags         In-App Notifications
// @Produce      json
// @Success      200 {object} dto.UnreadCountResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/me/notifications/unread-count [get]
func (h *InAppNotificationHandler) GetUnreadCount(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	count, err := h.notifUC.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.UnreadCountResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Unread count retrieved",
		},
		UnreadCount: count,
	})
}
