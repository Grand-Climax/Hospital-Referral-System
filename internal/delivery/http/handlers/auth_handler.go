package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/usecase"
)

type AuthHandler struct {
	authUseCase iusecase.AuthUseCase
}

func NewAuthHandler(authUseCase iusecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email" example:"superadmin@moh.gov.et"`
	Password   string `json:"password" binding:"required" example:"password123"`
	MFAChannel string `json:"mfa_channel,omitempty" binding:"omitempty,oneof=sms email" example:"email"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type MFAVerifyRequest struct {
	Code string `json:"code" binding:"required,len=6,numeric"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login godoc
// @Summary      User login
// @Description  Authenticate with email/password, then send OTP through configured MFA channel.
// @Description  **Roles:** Any user with an active account.
// @Description  **Common Errors:**
// @Description  - 401 (Invalid Credentials)
// @Description  - 401 (Account Inactive)
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200 {object} dto.LoginResponse "MFA intermediate token and channel"
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload",
		})
		return
	}

	loginResult, err := h.authUseCase.Login(c.Request.Context(), req.Email, req.Password, req.MFAChannel)
	if err != nil {
		if err == usecase.ErrInvalidCredentials || err == usecase.ErrInactiveAccount {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		if err == usecase.ErrInvalidMFAChannel || err == usecase.ErrMissingSMSContact || err == usecase.ErrMissingEmailContact {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		if err == usecase.ErrMFAOTPResendCooldown {
			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to process login",
		})
		return
	}

	msg := "Primary authentication successful. Verify OTP to continue."
	if loginResult.AccessToken != "" {
		msg = "Login successful."
	}

	c.JSON(http.StatusOK, dto.LoginResponse{
		MFAToken:     loginResult.MFAToken,
		Channel:      loginResult.Channel,
		AccessToken:  loginResult.AccessToken,
		RefreshToken: loginResult.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: msg,
		},
	})
}

// VerifyOTP godoc
// @Summary      Verify OTP
// @Description  Verify one-time code from configured channel and issue full access tokens.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body MFAVerifyRequest true "OTP verification payload"
// @Success      200 {object} dto.MFAVerifyResponse "Token pair"
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/mfa/verify [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload",
		})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid authentication context",
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()
	tokenPair, err := h.authUseCase.VerifyOTP(c.Request.Context(), userID, req.Code, ipAddress, userAgent)
	if err != nil {
		switch err {
		case usecase.ErrInvalidOTPCode, usecase.ErrOTPExpired, usecase.ErrOTPAttemptsExceeded:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrOTPNotRequested:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrInactiveAccount:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to verify OTP"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.MFAVerifyResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "OTP verified successfully",
		},
	})
}

// Refresh godoc
// @Summary      Refresh token
// @Description  Exchange a valid refresh token for a new access/refresh token pair. Used to maintain session without re-login.
// @Description  **Roles:** Any user with a valid refresh token.
// @Description  **Constraints:** Refresh token must not be blacklisted or expired.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized (token invalid/expired)
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body RefreshRequest true "Refresh token"
// @Success      200 {object} dto.RefreshResponse "New token pair"
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload",
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	tokenPair, err := h.authUseCase.Refresh(c.Request.Context(), req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.RefreshResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Token refreshed successfully",
		},
	})
}

// Logout godoc
// @Summary      User logout
// @Description  Revoke both access and refresh tokens. Blacklists the tokens to prevent further use.
// @Description  **Roles:** Any authenticated user.
// @Description  **Prerequisites:** Requires a valid Access Token in Authorization header.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body LogoutRequest true "Refresh token to revoke"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload",
		})
		return
	}

	// Extract access token safely from context or header
	authHeader := c.GetHeader("Authorization")
	accessToken := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		accessToken = authHeader[7:]
	}

	err := h.authUseCase.Logout(c.Request.Context(), accessToken, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to process logout",
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}
