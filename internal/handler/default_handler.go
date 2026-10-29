package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

// ─── Defaults Handler ─────────────────────────────────────────────────────────

type DefaultHandler struct {
	service     *service.DefaultService
	legalSvc    *service.LegalNoticeService
}

func NewDefaultHandler(svc *service.DefaultService, legalSvc *service.LegalNoticeService) *DefaultHandler {
	return &DefaultHandler{service: svc, legalSvc: legalSvc}
}

// ListDefaults godoc
// @Summary List defaults (filter by status/branch)
// @Tags Defaults & Legal
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (ACTIVE|RESOLVED|LEGAL)"
// @Param branch_id query int false "Filter by branch"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /defaults [get]
func (h *DefaultHandler) List(c *gin.Context) {
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

	defs, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch defaults"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: defs, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// DetectDefaults godoc
// @Summary Run default detection job
// @Tags Defaults & Legal
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.SuccessResponse
// @Router /defaults/detect [post]
func (h *DefaultHandler) Detect(c *gin.Context) {
	claims := middleware.GetClaims(c)
	count, err := h.service.Detect(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to run detection"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{
		Message: "Default detection completed",
		Data:    gin.H{"new_defaults_detected": count},
	})
}

// FlagDefault godoc
// @Summary Manually flag as default
// @Tags Defaults & Legal
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Default ID (subscriber_id)"
// @Param body body domain.FlagDefaultRequest true "Flag reason"
// @Success 201 {object} domain.Default
// @Router /defaults/{id}/flag [patch]
func (h *DefaultHandler) Flag(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.FlagDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Reason is required", Details: err.Error()})
		return
	}

	branchID := 0
	if claims.BranchID != nil {
		branchID = *claims.BranchID
	}

	// For manual flag, chit_id and chit_subscriber_id come from body or query
	chitID, _ := strconv.Atoi(c.Query("chit_id"))
	csID, _ := strconv.Atoi(c.Query("chit_subscriber_id"))

	def, err := h.service.Flag(c.Request.Context(), claims.CompanyID, branchID, subscriberID, chitID, csID, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to flag default"})
		return
	}
	c.JSON(http.StatusCreated, def)
}

// ResolveDefault godoc
// @Summary Mark default as resolved
// @Tags Defaults & Legal
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Default ID"
// @Param body body domain.ResolveDefaultRequest true "Resolution notes"
// @Success 200 {object} domain.SuccessResponse
// @Router /defaults/{id}/resolve [patch]
func (h *DefaultHandler) Resolve(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid default ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.ResolveDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Resolution notes are required", Details: err.Error()})
		return
	}
	if err := h.service.Resolve(c.Request.Context(), id, claims.CompanyID, claims.UserID, &req); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Default not found or already resolved"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to resolve default"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Default marked as resolved"})
}

// SendReminder godoc
// @Summary Send payment reminder
// @Tags Defaults & Legal
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Default ID"
// @Param body body domain.SendReminderRequest true "Reminder details"
// @Success 200 {object} domain.SuccessResponse
// @Router /defaults/{id}/reminder [post]
func (h *DefaultHandler) SendReminder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid default ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.SendReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.SendReminder(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Default not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to send reminder"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Reminder sent successfully"})
}

// ─── Legal Notice Handler ─────────────────────────────────────────────────────

type LegalNoticeHandler struct {
	service *service.LegalNoticeService
}

func NewLegalNoticeHandler(svc *service.LegalNoticeService) *LegalNoticeHandler {
	return &LegalNoticeHandler{service: svc}
}

// ListLegalNotices godoc
// @Summary List legal notices
// @Tags Defaults & Legal
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /legal-notices [get]
func (h *LegalNoticeHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	notices, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch legal notices"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: notices, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// IssueLegalNotice godoc
// @Summary Issue legal notice
// @Tags Defaults & Legal
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.IssueLegalNoticeRequest true "Notice details"
// @Success 201 {object} domain.LegalNotice
// @Router /legal-notices [post]
func (h *LegalNoticeHandler) Issue(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.IssueLegalNoticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	notice, err := h.service.Issue(c.Request.Context(), claims.CompanyID, claims.UserID, &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to issue legal notice"})
		return
	}
	c.JSON(http.StatusCreated, notice)
}

// GetLegalNotice godoc
// @Summary Get legal notice
// @Tags Defaults & Legal
// @Security BearerAuth
// @Produce json
// @Param id path int true "Notice ID"
// @Success 200 {object} domain.LegalNotice
// @Router /legal-notices/{id} [get]
func (h *LegalNoticeHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid notice ID"})
		return
	}
	claims := middleware.GetClaims(c)
	notice, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Legal notice not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch legal notice"})
		return
	}
	c.JSON(http.StatusOK, notice)
}

// EscalateLegalNotice godoc
// @Summary Escalate to court summon
// @Tags Defaults & Legal
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Notice ID"
// @Param body body domain.EscalateLegalNoticeRequest true "Escalation details"
// @Success 200 {object} domain.SuccessResponse
// @Router /legal-notices/{id}/escalate [patch]
func (h *LegalNoticeHandler) Escalate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid notice ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.EscalateLegalNoticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.Escalate(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Legal notice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to escalate notice"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Notice escalated to court summon"})
}
