package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type PayoutHandler struct {
	service *service.PayoutService
}

func NewPayoutHandler(svc *service.PayoutService) *PayoutHandler {
	return &PayoutHandler{service: svc}
}

// ListPayouts godoc
// @Summary List payouts (filter by status/group)
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (PENDING|APPROVED|DISBURSED)"
// @Param chit_id query int false "Filter by chit group"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /payouts [get]
func (h *PayoutHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	if v, err := strconv.Atoi(c.Query("chit_id")); err == nil && v > 0 {
		filters["chit_id"] = v
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	payouts, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch payouts"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: payouts, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// InitiatePayout godoc
// @Summary Initiate payout for winner
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.InitiatePayoutRequest true "Payout details"
// @Success 201 {object} domain.Payout
// @Router /payouts [post]
func (h *PayoutHandler) Initiate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.InitiatePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	payout, err := h.service.Initiate(c.Request.Context(), claims.CompanyID, &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to initiate payout"})
		return
	}
	c.JSON(http.StatusCreated, payout)
}

// GetPayout godoc
// @Summary Get payout details
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Produce json
// @Param id path int true "Payout ID"
// @Success 200 {object} domain.Payout
// @Router /payouts/{id} [get]
func (h *PayoutHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid payout ID"})
		return
	}
	claims := middleware.GetClaims(c)
	payout, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Payout not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch payout"})
		return
	}
	c.JSON(http.StatusOK, payout)
}

// ApprovePayout godoc
// @Summary Approve payout (manager)
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Payout ID"
// @Param body body domain.ApprovePayoutRequest true "Approval remarks"
// @Success 200 {object} domain.SuccessResponse
// @Router /payouts/{id}/approve [patch]
func (h *PayoutHandler) Approve(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid payout ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.ApprovePayoutRequest
	c.ShouldBindJSON(&req) // optional body
	if err := h.service.Approve(c.Request.Context(), id, claims.CompanyID, claims.UserID, &req); err != nil {
		if err == domain.ErrPayoutNotPending {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Payout is not in pending state"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to approve payout"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Payout approved successfully"})
}

// DisbursePayout godoc
// @Summary Mark as disbursed
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Payout ID"
// @Param body body domain.DisbursePayoutRequest true "Disbursement details"
// @Success 200 {object} domain.SuccessResponse
// @Router /payouts/{id}/disburse [patch]
func (h *PayoutHandler) Disburse(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid payout ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.DisbursePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.Disburse(c.Request.Context(), id, claims.CompanyID, claims.UserID, &req); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to disburse payout"})
		return
	}
	payout, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Payout disbursed successfully"})
		return
	}
	c.JSON(http.StatusOK, payout)
}

// ListDividends godoc
// @Summary List dividend records
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Produce json
// @Param chit_id query int false "Filter by chit group"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /dividends [get]
func (h *PayoutHandler) ListDividends(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if v, err := strconv.Atoi(c.Query("chit_id")); err == nil && v > 0 {
		filters["chit_id"] = v
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	divs, total, err := h.service.ListDividends(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch dividends"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: divs, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// CalculateDividend godoc
// @Summary Calculate dividend for auction
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CalculateDividendRequest true "Auction ID"
// @Success 200 {object} domain.Dividend
// @Router /dividends/calculate [post]
func (h *PayoutHandler) CalculateDividend(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CalculateDividendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	div, err := h.service.CalculateDividend(c.Request.Context(), req.AuctionID, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Auction not found or not completed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to calculate dividend", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, div)
}

// PostDividend godoc
// @Summary Post dividend to installments
// @Tags Payouts & Dividends
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.PostDividendRequest true "Dividend ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /dividends/post [post]
func (h *PayoutHandler) PostDividend(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.PostDividendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.PostDividend(c.Request.Context(), req.DividendID, claims.CompanyID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to post dividend: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Dividend posted to installments"})
}
