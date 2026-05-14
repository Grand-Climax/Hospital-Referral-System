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
	Email    string `json:"email" binding:"required,email" example:"superadmin@moh.gov.et"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type MFAVerifyRequest struct {
	Code string `json:"code" binding:"required,len=6,numeric"`
}

type MFAResetRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login godoc
// @Summary      User login
// @Description  Authenticate with email and password as step 1 of MFA.
// @Description  Returns a short-lived intermediate token (`mfa_token`) that is only valid for MFA setup/verification flows.
// @Description  If `mfa_setup_required=true`, client must call MFA setup first, then verify.
// @Description  **Roles:** Any user with an active account.
// @Description  **Common Errors:**
// @Description  - 401 (Invalid Credentials)
// @Description  - 401 (Account Inactive)
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200 {object} dto.LoginResponse "Intermediate MFA token response"
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

	loginResult, err := h.authUseCase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == usecase.ErrInvalidCredentials || err == usecase.ErrInactiveAccount {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
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

	c.JSON(http.StatusOK, dto.LoginResponse{
		MFAToken:         loginResult.MFAToken,
		MFASetupRequired: loginResult.MFASetupRequired,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Primary authentication successful. Complete MFA verification.",
		},
	})
}

// VerifyMFA godoc
// @Summary      Verify MFA code
// @Description  Complete step 2 of login by verifying a 6-digit TOTP code.
// @Description  Requires the intermediate token from `/api/v1/auth/login`.
// @Description  On success, returns full access and refresh tokens.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body MFAVerifyRequest true "6-digit TOTP code"
// @Success      200 {object} dto.MFAVerifyResponse "Full authentication token pair"
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/mfa/verify [post]
func (h *AuthHandler) VerifyMFA(c *gin.Context) {
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

	tokenPair, err := h.authUseCase.VerifyMFA(c.Request.Context(), userID, req.Code, ipAddress, userAgent)
	if err != nil {
		switch err {
		case usecase.ErrInvalidMFACode:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrMFANotConfigured:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrInactiveAccount:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to verify MFA code"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.MFAVerifyResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "MFA verification successful",
		},
	})
}

// SetupMFA godoc
// @Summary      Setup MFA secret
// @Description  Initialize MFA for the current user when no MFA secret exists yet.
// @Description  Generates and persists a TOTP secret, then returns an `otpauth://` URI for QR generation.
// @Description  Accepts either a full access token or the intermediate MFA token from login.
// @Tags         Authentication
// @Produce      json
// @Success      200 {object} dto.MFASetupResponse "MFA setup URI"
// @Failure      401 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/mfa/setup [post]
func (h *AuthHandler) SetupMFA(c *gin.Context) {
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

	otpauthURL, err := h.authUseCase.SetupMFA(c.Request.Context(), userID)
	if err != nil {
		switch err {
		case usecase.ErrMFAAlreadySet:
			c.JSON(http.StatusConflict, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrInactiveAccount:
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to initialize MFA setup"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.MFASetupResponse{
		OTPAuthURL: otpauthURL,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "MFA setup initialized successfully",
		},
	})
}

// ResetMFA godoc
// @Summary      Reset user MFA (Hospital Admin)
// @Description  Clear MFA secret for a target user, forcing setup on next login.
// @Description  Action is audited with `RESET_MFA`.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        body body MFAResetRequest true "Target user ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/mfa/reset [post]
func (h *AuthHandler) ResetMFA(c *gin.Context) {
	var req MFAResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload",
		})
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid user_id",
		})
		return
	}

	adminIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	adminID, ok := adminIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid authentication context"})
		return
	}

	if err := h.authUseCase.ResetMFA(c.Request.Context(), adminID, targetUserID); err != nil {
		if err == usecase.ErrInvalidCredentials {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Target user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to reset MFA"})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "MFA reset successful",
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
