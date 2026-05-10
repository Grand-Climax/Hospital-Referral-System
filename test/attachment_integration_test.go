package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/handlers"
)

// MockAttachmentUseCase resides in mocks_test.go


func TestAttachmentEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockAttachmentUseCase)
	handler := handlers.NewAttachmentHandler(mockUC)

	referralID := uuid.New()
	doctorID := uuid.New()

	router := gin.Default()
	// Mock auth middleware for "userID" (standardized across handlers)
	router.Use(func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Next()
	})
	router.POST("/api/v1/referrals/:id/attachments", handler.UploadAttachment)
	router.DELETE("/api/v1/referrals/:id/attachments/:attachment_id", handler.DeleteFromReferral)


	t.Run("Delete Attachment From Referral", func(t *testing.T) {
		attachmentID := uuid.New()
		mockUC.On("DeleteAttachmentFromReferral", mock.Anything, referralID, attachmentID, doctorID).Return(nil)

		url := "/api/v1/referrals/" + referralID.String() + "/attachments/" + attachmentID.String()
		req := httptest.NewRequest(http.MethodDelete, url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})
}
