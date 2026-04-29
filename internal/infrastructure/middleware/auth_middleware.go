package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/pkg/auth"
)

// RequireAuth validates the JWT from the Authorization header, checks the blacklist, and sets user claims in context.
func RequireAuth(blacklist cache.TokenBlacklist, userRepo irepository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   "Authorization header format must be Bearer {token}",
			})
			return
		}

		tokenStr := parts[1]

		// 1. Check if token is blacklisted in Redis
		if blacklist != nil {
			isBlacklisted, err := blacklist.IsBlacklisted(c.Request.Context(), tokenStr)
			if err != nil {
				// Failing closed or safely open depending on strictness. We fail safely for DB errors.
				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{
					Success: false,
					Error:   "Failed to verify token status",
				})
				return
			}
			if isBlacklisted {
				c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
					Success: false,
					Error:   "Token has been revoked",
				})
				return
			}
		}

		// 2. Cryptographically validate payload
		payload, err := auth.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   "Invalid or expired access token",
			})
			return
		}

		// Set claims securely into the Gin Context
		if userRepo != nil {
			user, lookupErr := userRepo.FindByID(c.Request.Context(), payload.UserID)
			if lookupErr != nil || user.IsDeleted || !user.IsActive {
				if blacklist != nil && payload.ExpiresAt != nil {
					ttl := time.Until(payload.ExpiresAt.Time)
					if ttl > 0 {
						_ = blacklist.Add(c.Request.Context(), tokenStr, ttl)
					}
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
					Success: false,
					Error:   "Account is inactive or deleted",
				})
				return
			}
		}

		// Set claims securely into the Gin Context
		c.Set("userID", payload.UserID)
		c.Set("role", payload.Role)
		c.Set("hospID", payload.HospID)
		c.Set("deptID", payload.DeptID)

		c.Next()
	}
}

// RequireRole restricts the route to specific user roles.
// Must be used AFTER RequireAuth middleware.
func RequireRole(allowedRoles ...entity.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "User role not found in context. Is RequireAuth missing?"})
			return
		}

		userRole, ok := roleVal.(entity.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid role type in context"})
			return
		}

		isAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error:   "Forbidden: You do not have permission to access this resource",
			})
			return
		}

		c.Next()
	}
}

// RequirePermission abstracts the specific role check to a capability check.
func RequirePermission(requiredAction entity.ActionType) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "User role not found in context. Is RequireAuth missing?"})
			return
		}

		userRole, ok := roleVal.(entity.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid role type in context"})
			return
		}

		if !auth.HasPermission(userRole, requiredAction) {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error:   "Forbidden: You do not have permission to perform this action",
			})
			return
		}

		c.Next()
	}
}
