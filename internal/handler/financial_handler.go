package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type FinancialHandler struct {
	svc *service.FinancialService
}

func NewFinancialHandler(svc *service.FinancialService) *FinancialHandler {
	return &FinancialHandler{svc: svc}
}

func parseReq(c *gin.Context) domain.FinancialStatementRequest {
	var req domain.FinancialStatementRequest
	c.ShouldBindQuery(&req)
	if req.Period == "" {
		req.Period = "fy" // default Indian FY
	}
	return req
}

// GET /api/v1/finance/pl
func (h *FinancialHandler) ProfitLoss(c *gin.Context) {
	claims := middleware.GetClaims(c)
	pl, err := h.svc.GetProfitLoss(c.Request.Context(), claims.CompanyID, parseReq(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate P&L statement: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, pl)
}

// GET /api/v1/finance/balance-sheet
func (h *FinancialHandler) BalanceSheet(c *gin.Context) {
	claims := middleware.GetClaims(c)
	bs, err := h.svc.GetBalanceSheet(c.Request.Context(), claims.CompanyID, parseReq(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate Balance Sheet: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, bs)
}

// GET /api/v1/finance/cash-flow
func (h *FinancialHandler) CashFlow(c *gin.Context) {
	claims := middleware.GetClaims(c)
	cf, err := h.svc.GetCashFlow(c.Request.Context(), claims.CompanyID, parseReq(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate Cash Flow statement: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cf)
}

// GET /api/v1/finance/creditors-debtors
func (h *FinancialHandler) CreditorDebtors(c *gin.Context) {
	claims := middleware.GetClaims(c)
	sched, err := h.svc.GetCreditorDebtorSchedule(c.Request.Context(), claims.CompanyID, parseReq(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate Creditor/Debtor schedule: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, sched)
}

// GET /api/v1/finance/gst
func (h *FinancialHandler) GSTSummary(c *gin.Context) {
	claims := middleware.GetClaims(c)
	gst, err := h.svc.GetGSTSummary(c.Request.Context(), claims.CompanyID, parseReq(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate GST summary: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gst)
}

// POST /api/v1/finance/expenses
func (h *FinancialHandler) CreateExpense(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	expense, err := h.svc.CreateExpense(c.Request.Context(), claims.CompanyID, claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to create expense: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, expense)
}

// GET /api/v1/finance/expenses
func (h *FinancialHandler) ListExpenses(c *gin.Context) {
	claims := middleware.GetClaims(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	category := c.Query("category")

	expenses, total, err := h.svc.ListExpenses(c.Request.Context(), claims.CompanyID,
		parseReq(c), category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch expenses: " + err.Error()})
		return
	}
	page := 1
	if limit > 0 {
		page = offset/limit + 1
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: expenses, Total: total, Page: page, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// POST /api/v1/finance/assets
func (h *FinancialHandler) CreateAsset(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	asset, err := h.svc.CreateAsset(c.Request.Context(), claims.CompanyID, claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to record asset: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, asset)
}

// POST /api/v1/finance/capital
func (h *FinancialHandler) CreateCapital(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CreateCapitalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	cap, err := h.svc.CreateCapital(c.Request.Context(), claims.CompanyID, claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to record capital: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cap)
}
