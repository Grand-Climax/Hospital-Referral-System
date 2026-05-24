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

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"doctor@hospital.et"`
}

type ForgotPasswordVerifyRequest struct {
	Email string `json:"email" binding:"required,email" example:"doctor@hospital.et"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8" example:"newpassword123"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"password123"`
	NewPassword     string `json:"new_password" binding:"required,min=8" example:"newpassword456"`
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

// ForgotPassword godoc
// @Summary      Request password reset OTP
// @Description  Sends a one-time password reset code to the user's email if an active account exists.
// @Description  Always returns the same success message to prevent email enumeration.
// @Description  **Roles:** Public (no authentication required).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body ForgotPasswordRequest true "Account email"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      429 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid request payload"})
		return
	}

	err := h.authUseCase.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		if err == usecase.ErrPasswordResetCooldown {
			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to process password reset request"})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "If an account with that email exists, a reset code has been sent",
	})
}

// VerifyForgotPasswordOTP godoc
// @Summary      Verify password reset OTP
// @Description  Validates the email OTP and returns a short-lived reset token for setting a new password.
// @Description  **Roles:** Public (no authentication required).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body ForgotPasswordVerifyRequest true "Email and OTP code"
// @Success      200 {object} dto.PasswordResetVerifyResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/auth/forgot-password/verify [post]
func (h *AuthHandler) VerifyForgotPasswordOTP(c *gin.Context) {
	var req ForgotPasswordVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid request payload"})
		return
	}

	result, err := h.authUseCase.VerifyForgotPasswordOTP(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		switch err {
		case usecase.ErrInvalidOTPCode, usecase.ErrOTPExpired, usecase.ErrOTPAttemptsExceeded:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrOTPNotRequested:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to verify reset code"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.PasswordResetVerifyResponse{
		ResetToken: result.ResetToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Reset code verified successfully",
		},
	})
}

// ResetPassword godoc
// @Summary      Reset password after OTP verification
// @Description  Sets a new password using the reset token from OTP verification and logs the user in.
// @Description  **Roles:** Public (requires password reset confirmation token).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body ResetPasswordRequest true "New password"
// @Success      200 {object} dto.ResetPasswordResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid request payload"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid authentication context"})
		return
	}

	tokenPair, err := h.authUseCase.ResetPassword(c.Request.Context(), userID, req.NewPassword, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		switch err {
		case usecase.ErrSamePassword:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrInactiveAccount, usecase.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to reset password"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.ResetPasswordResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Password reset successfully",
		},
	})
}

// ChangePassword godoc
// @Summary      Change password (logged in)
// @Description  Updates the password for the currently authenticated user. Requires the current password; no OTP needed.
// @Description  Returns a new token pair and revokes all existing sessions.
// @Description  **Roles:** Any authenticated user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body body ChangePasswordRequest true "Current and new password"
// @Success      200 {object} dto.ChangePasswordResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/me/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid request payload"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid authentication context"})
		return
	}

	tokenPair, err := h.authUseCase.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		switch err {
		case usecase.ErrIncorrectPassword:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrSamePassword:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrInactiveAccount:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to change password"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.ChangePasswordResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Password changed successfully",
		},
	})
}
