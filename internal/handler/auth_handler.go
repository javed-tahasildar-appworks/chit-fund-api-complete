package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

// Login godoc
// @Summary Staff login
// @Description Authenticate staff and return JWT pair
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "Login credentials"
// @Success 200 {object} domain.LoginResponse
// @Failure 400,401 {object} domain.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Code: "VALIDATION_ERROR", Details: err.Error()})
		return
	}
	resp, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Logout godoc
// @Summary Invalidate session
// @Description Blacklist current access token server-side
// @Tags Authentication
// @Security BearerAuth
// @Success 200 {object} domain.SuccessResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	_ = h.service.Logout(c.Request.Context(), token)
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Logged out successfully"})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token (no auth header needed)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} domain.LoginResponse
// @Failure 400,401 {object} domain.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}
	resp, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: "Invalid or expired token"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetMe godoc
// @Summary Get current user profile
// @Description Return profile of the currently authenticated user
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.User
// @Failure 401 {object} domain.ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: "Unauthorized"})
		return
	}
	user, err := h.service.GetMe(c.Request.Context(), claims.UserID, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to get profile"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// ChangePassword godoc
// @Summary Change own password
// @Description Change the authenticated user's password
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.ChangePasswordRequest true "Password change request"
// @Success 200 {object} domain.SuccessResponse
// @Failure 400,401 {object} domain.ErrorResponse
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: "Unauthorized"})
		return
	}
	var req domain.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), claims.UserID, &req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Failed to change password"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Password changed successfully"})
}
