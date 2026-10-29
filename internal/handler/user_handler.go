package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// ListUsers godoc
// @Summary List staff users (paginated)
// @Tags Users & Access
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (ACTIVE|INACTIVE)"
// @Param branch_id query int false "Filter by branch"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /users [get]
func (h *UserHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	if b, err := strconv.Atoi(c.Query("branch_id")); err == nil && b > 0 {
		filters["branch_id"] = b
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	users, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: users, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// CreateUser godoc
// @Summary Create staff user
// @Tags Users & Access
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateUserRequest true "User details"
// @Success 201 {object} domain.User
// @Router /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	user, err := h.service.Create(c.Request.Context(), claims.CompanyID, &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to create user"})
		return
	}
	c.JSON(http.StatusCreated, user)
}

// GetUser godoc
// @Summary Get user by ID
// @Tags Users & Access
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} domain.User
// @Router /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid user ID"})
		return
	}
	claims := middleware.GetClaims(c)
	user, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch user"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user
// @Tags Users & Access
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body domain.UpdateUserRequest true "Update fields"
// @Success 200 {object} domain.SuccessResponse
// @Router /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid user ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.Update(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to update user"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "User updated successfully"})
}

// DeactivateUser godoc
// @Summary Deactivate user account
// @Tags Users & Access
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /users/{id}/deactivate [patch]
func (h *UserHandler) Deactivate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid user ID"})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.Deactivate(c.Request.Context(), id, claims.CompanyID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to deactivate user"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "User deactivated successfully"})
}

// ActivateUser godoc
// @Summary Re-activate user account
// @Tags Users & Access
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /users/{id}/activate [patch]
func (h *UserHandler) Activate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid user ID"})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.Activate(c.Request.Context(), id, claims.CompanyID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to activate user"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "User activated successfully"})
}
