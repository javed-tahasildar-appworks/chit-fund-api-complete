package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

// ─── Dashboard Handler ────────────────────────────────────────────────────────

type DashboardHandler struct {
	service *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: svc}
}

// GetDashboard godoc
// @Summary Full dashboard bundle
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.DashboardBundle
// @Router /dashboard [get]
func (h *DashboardHandler) GetBundle(c *gin.Context) {
	claims := middleware.GetClaims(c)
	bundle, err := h.service.GetBundle(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch dashboard"})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

// GetKPIs godoc
// @Summary KPI cards only — lightweight
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.DashboardKPIs
// @Router /dashboard/kpis [get]
func (h *DashboardHandler) GetKPIs(c *gin.Context) {
	claims := middleware.GetClaims(c)
	kpis, err := h.service.GetKPIs(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch KPIs"})
		return
	}
	c.JSON(http.StatusOK, kpis)
}

// GetCollections godoc
// @Summary Monthly collection chart data
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.MonthlyCollectionItem
// @Router /dashboard/collections [get]
func (h *DashboardHandler) GetCollections(c *gin.Context) {
	claims := middleware.GetClaims(c)
	items, err := h.service.GetMonthlyCollections(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch collection data"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// ─── Report Handler ───────────────────────────────────────────────────────────

type ReportHandler struct {
	service *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{service: svc}
}

// MonthlyCollections godoc
// @Summary Monthly collection summary
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.MonthlyCollectionItem
// @Router /reports/monthly-collections [get]
func (h *ReportHandler) MonthlyCollections(c *gin.Context) {
	claims := middleware.GetClaims(c)
	items, err := h.service.GetMonthlyCollections(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "total": len(items)})
}

// DefaultersReport godoc
// @Summary Defaulters report
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.DefaulterReport
// @Router /reports/defaulters [get]
func (h *ReportHandler) DefaultersReport(c *gin.Context) {
	claims := middleware.GetClaims(c)
	reports, err := h.service.GetDefaultersReport(c.Request.Context(), claims.CompanyID, map[string]interface{}{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reports, "total": len(reports)})
}

// ChitPerformance godoc
// @Summary Chit group performance
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.ChitPerformanceReport
// @Router /reports/chit-performance [get]
func (h *ReportHandler) ChitPerformance(c *gin.Context) {
	claims := middleware.GetClaims(c)
	reports, err := h.service.GetChitPerformance(c.Request.Context(), claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reports, "total": len(reports)})
}

// SubscriberLedger godoc
// @Summary Subscriber ledger
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber ID"
// @Success 200 {array} domain.SubscriberLedgerEntry
// @Router /reports/subscriber-ledger/{id} [get]
func (h *ReportHandler) SubscriberLedger(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	entries, err := h.service.GetSubscriberLedger(c.Request.Context(), id, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate ledger"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entries, "total": len(entries)})
}

// AuditLogs godoc
// @Summary Audit log (paginated + filter)
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Param table_name query string false "Filter by table"
// @Param user_id query int false "Filter by user"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /audit-logs [get]
func (h *ReportHandler) AuditLogs(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if t := c.Query("table_name"); t != "" {
		filters["table_name"] = t
	}
	if u, err := strconv.Atoi(c.Query("user_id")); err == nil && u > 0 {
		filters["user_id"] = u
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	logs, total, err := h.service.GetAuditLogs(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch audit logs"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: logs, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// SystemLogs godoc
// @Summary System error logs
// @Tags Reports & Audit
// @Security BearerAuth
// @Produce json
// @Param level query string false "Filter by level (ERROR|WARN|INFO)"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /system-logs [get]
func (h *ReportHandler) SystemLogs(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if l := c.Query("level"); l != "" {
		filters["level"] = l
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	logs, total, err := h.service.GetSystemLogs(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch system logs"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: logs, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}
