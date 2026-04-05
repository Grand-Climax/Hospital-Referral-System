package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/delivery/http/dto"
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

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login godoc
// @Summary      User login
// @Description  Authenticate a user and return access/refresh token pair
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200 {object} dto.LoginResponse "Token pair"
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

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	tokenPair, err := h.authUseCase.Login(c.Request.Context(), req.Email, req.Password, ipAddress, userAgent)
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
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Login successful",
		},
	})
}

// Refresh godoc
// @Summary      Refresh token
// @Description  Exchange a valid refresh token for a new access/refresh token pair
// @Tags         Auth
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
// @Description  Revoke both access and refresh tokens
// @Tags         Auth
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
