package middleware

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

// AuditLogger writes every authenticated API call to the audit_logs table.
// It captures timestamp, user ID, action (HTTP method + path), affected resource, response status, IP and user-agent.
// Writing is done asynchronously so it does not block the HTTP response.
func AuditLogger(repo repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request first
		c.Next()

		// Only audit authenticated requests (userID must be set by RequireAuth)
		userIDVal, exists := c.Get("userID")
		if !exists {
			return
		}
		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			return
		}

		method := c.Request.Method
		path := c.FullPath() // route pattern, e.g. /api/v1/users/:id
		if path == "" {
			path = c.Request.URL.Path
		}
		statusCode := c.Writer.Status()
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Build action string from method + path
		action := fmt.Sprintf("%s %s", method, path)

		// Determine the affected resource from the route pattern
		resource := deriveResource(path)
		resourceID := deriveResourceID(c)

		auditEntry := &entity.AuditLog{
			UserID:     userID,
			ActionType: entity.ActionAPICall,
			Resource:   &resource,
			ResourceID: resourceID,
			IPAddress:  &ipAddress,
			UserAgent:  &userAgent,
			Timestamp:  time.Now(),
		}

		// Store the full action description in NewValue as JSON
		actionDetail := fmt.Sprintf(`{"method":"%s","path":"%s","status":%d}`, method, c.Request.URL.Path, statusCode)
		auditEntry.NewValue = &actionDetail

		// Override ActionType with a more specific action if applicable
		auditEntry.ActionType = mapActionType(action)

		// Write asynchronously to avoid blocking the response
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := repo.Create(ctx, auditEntry); err != nil {
				log.Printf("[AUDIT] Failed to write audit log: %v", err)
			}
		}()
	}
}

// deriveResource extracts the resource name from the route pattern.
func deriveResource(path string) string {
	// e.g. /api/v1/users/:id -> "users", /api/v1/hospitals/:id/departments -> "hospital_departments"
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	if len(parts) == 0 {
		return "unknown"
	}

	var resources []string
	for _, p := range parts {
		if !strings.HasPrefix(p, ":") && p != "" {
			resources = append(resources, p)
		}
	}
	if len(resources) == 0 {
		return "unknown"
	}
	return strings.Join(resources, "_")
}

// deriveResourceID extracts the primary resource ID from route params.
func deriveResourceID(c *gin.Context) *string {
	// Try "id" first, then "deptId"
	if id := c.Param("id"); id != "" {
		return &id
	}
	return nil
}

// mapActionType maps the HTTP action to a more specific ActionType where possible.
func mapActionType(action string) entity.ActionType {
	lower := strings.ToLower(action)

	switch {
	case strings.Contains(lower, "/users"):
		return entity.ActionManageUsers
	case strings.Contains(lower, "/hospitals"):
		return entity.ActionManageHospitals
	case strings.Contains(lower, "/departments"):
		return entity.ActionManageDepts
	case strings.Contains(lower, "/auth/login"):
		return entity.ActionLogin
	case strings.Contains(lower, "/auth/logout"):
		return entity.ActionLogout
	default:
		return entity.ActionAPICall
	}
}
