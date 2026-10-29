package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type CompanyHandler struct {
	service *service.CompanyService
	branch  *service.BranchService
}

func NewCompanyHandler(svc *service.CompanyService, branch *service.BranchService) *CompanyHandler {
	return &CompanyHandler{service: svc, branch: branch}
}

// ─── Company ──────────────────────────────────────────────────────────────────

// GetProfile godoc
// @Summary Get company profile
// @Tags Company & Configuration
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.Company
// @Router /company [get]
func (h *CompanyHandler) GetProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	company, err := h.service.GetProfile(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Company not found"})
		return
	}
	c.JSON(http.StatusOK, company)
}

// UpdateProfile godoc
// @Summary Update company details
// @Tags Company & Configuration
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.UpdateCompanyRequest true "Update fields"
// @Success 200 {object} domain.SuccessResponse
// @Router /company [put]
func (h *CompanyHandler) UpdateProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.Update(c.Request.Context(), claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to update company"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Company updated successfully"})
}

// ─── Branches ─────────────────────────────────────────────────────────────────

// ListBranches godoc
// @Summary List all branches
// @Tags Company & Configuration
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.Branch
// @Router /branches [get]
func (h *CompanyHandler) ListBranches(c *gin.Context) {
	claims := middleware.GetClaims(c)
	branches, err := h.branch.ListBranches(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch branches"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": branches, "total": len(branches)})
}

// CreateBranch godoc
// @Summary Create new branch
// @Tags Company & Configuration
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateBranchRequest true "Branch details"
// @Success 201 {object} domain.Branch
// @Router /branches [post]
func (h *CompanyHandler) CreateBranch(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	branch, err := h.branch.CreateBranch(c.Request.Context(), claims.CompanyID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to create branch"})
		return
	}
	c.JSON(http.StatusCreated, branch)
}

// GetBranch godoc
// @Summary Get branch by ID
// @Tags Company & Configuration
// @Security BearerAuth
// @Produce json
// @Param id path int true "Branch ID"
// @Success 200 {object} domain.Branch
// @Router /branches/{id} [get]
func (h *CompanyHandler) GetBranch(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid branch ID"})
		return
	}
	claims := middleware.GetClaims(c)
	branch, err := h.branch.GetBranch(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Branch not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch branch"})
		return
	}
	c.JSON(http.StatusOK, branch)
}

// UpdateBranch godoc
// @Summary Update branch
// @Tags Company & Configuration
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Branch ID"
// @Param body body domain.UpdateBranchRequest true "Update fields"
// @Success 200 {object} domain.SuccessResponse
// @Router /branches/{id} [put]
func (h *CompanyHandler) UpdateBranch(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid branch ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.branch.UpdateBranch(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to update branch"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Branch updated successfully"})
}

// ─── States ───────────────────────────────────────────────────────────────────

// ListStates godoc
// @Summary List states with regulatory config
// @Tags Company & Configuration
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.State
// @Router /states [get]
func (h *CompanyHandler) ListStates(c *gin.Context) {
	states, err := h.branch.ListStates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch states"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": states})
}

// GetState godoc
// @Summary Get state regulatory values
// @Tags Company & Configuration
// @Security BearerAuth
// @Produce json
// @Param id path int true "State ID"
// @Success 200 {object} domain.State
// @Router /states/{id} [get]
func (h *CompanyHandler) GetState(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid state ID"})
		return
	}
	state, err := h.branch.GetState(c.Request.Context(), id)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "State not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch state"})
		return
	}
	c.JSON(http.StatusOK, state)
}

// ─── Roles ────────────────────────────────────────────────────────────────────

// ListRoles godoc
// @Summary List all roles
// @Tags Users & Access
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.Role
// @Router /roles [get]
func (h *CompanyHandler) ListRoles(c *gin.Context) {
	claims := middleware.GetClaims(c)
	roles, err := h.branch.ListRoles(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch roles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

// GetRole godoc
// @Summary Get role with permissions
// @Tags Users & Access
// @Security BearerAuth
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} domain.Role
// @Router /roles/{id} [get]
func (h *CompanyHandler) GetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid role ID"})
		return
	}
	claims := middleware.GetClaims(c)
	role, err := h.branch.GetRole(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Role not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch role"})
		return
	}
	c.JSON(http.StatusOK, role)
}

// UpdateRolePermissions godoc
// @Summary Update role permissions
// @Tags Users & Access
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param body body domain.UpdateRolePermissionsRequest true "Permissions JSON"
// @Success 200 {object} domain.SuccessResponse
// @Router /roles/{id} [put]
func (h *CompanyHandler) UpdateRolePermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid role ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UpdateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.branch.UpdateRolePermissions(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Role permissions updated"})
}
