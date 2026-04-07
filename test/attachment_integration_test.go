package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
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

	t.Run("Bulk Register Attachments - Valid", func(t *testing.T) {
		reqPayload := dto.BulkAttachmentRequest{
			Attachments: []dto.CreateAttachmentRequest{
				{
					FileName: "scan.pdf",
					FileType: "application/pdf",
					FileURL:  "https://cloudinary.com/scan.pdf",
					PublicID: "cloudinary_123",
					FileSize: 1024,
					Category: "GENERAL_CLINICAL",
				},
			},
		}

		returnedAttachments := []entity.Attachment{
			{
				ID:          uuid.New(),
				ReferralID:  referralID,
				FileName:    "scan.pdf",
				FileType:    "application/pdf",
				FileSize:    1024,
				Category:    "GENERAL_CLINICAL",
				StoragePath: "https://cloudinary.com/scan.pdf",
				PublicID:    "cloudinary_123",
				UploadedAt:  time.Now(),
			},
		}

		mockUC.On("AddAttachmentsToReferral", mock.Anything, referralID, doctorID, reqPayload.Attachments).Return(returnedAttachments, nil)

		body, _ := json.Marshal(reqPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals/"+referralID.String()+"/attachments", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockUC.AssertExpectations(t)
	})

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
