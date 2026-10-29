package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type InstallmentHandler struct {
	service *service.InstallmentService
}

func NewInstallmentHandler(svc *service.InstallmentService) *InstallmentHandler {
	return &InstallmentHandler{service: svc}
}

// ListInstallments godoc
// @Summary List installments (filter by group/month)
// @Tags Installments & Payments
// @Security BearerAuth
// @Produce json
// @Param chit_id query int false "Filter by chit group"
// @Param month_number query int false "Filter by month"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /installments [get]
func (h *InstallmentHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if v, err := strconv.Atoi(c.Query("chit_id")); err == nil && v > 0 {
		filters["chit_id"] = v
	}
	if v, err := strconv.Atoi(c.Query("month_number")); err == nil && v > 0 {
		filters["month_number"] = v
	}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	insts, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch installments"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: insts, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// GenerateBatch godoc
// @Summary Generate monthly installment batch
// @Tags Installments & Payments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.GenerateInstallmentRequest true "Batch details"
// @Success 201 {object} domain.Installment
// @Router /installments/generate [post]
func (h *InstallmentHandler) GenerateBatch(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.GenerateInstallmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	inst, err := h.service.GenerateBatch(c.Request.Context(), &req, claims.CompanyID)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to generate installment batch"})
		return
	}
	c.JSON(http.StatusCreated, inst)
}

// GetInstallment godoc
// @Summary Get installment details
// @Tags Installments & Payments
// @Security BearerAuth
// @Produce json
// @Param id path int true "Installment ID"
// @Success 200 {object} domain.Installment
// @Router /installments/{id} [get]
func (h *InstallmentHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid installment ID"})
		return
	}
	claims := middleware.GetClaims(c)
	inst, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Installment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch installment"})
		return
	}
	c.JSON(http.StatusOK, inst)
}

// ListSubscriberInstallments godoc
// @Summary List subscriber-level installments
// @Tags Installments & Payments
// @Security BearerAuth
// @Produce json
// @Param chit_id query int false "Filter by chit"
// @Param subscriber_id query int false "Filter by subscriber"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /subscriber-installments [get]
func (h *InstallmentHandler) ListSubscriberInstallments(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if v, err := strconv.Atoi(c.Query("chit_id")); err == nil && v > 0 {
		filters["chit_id"] = v
	}
	if v, err := strconv.Atoi(c.Query("subscriber_id")); err == nil && v > 0 {
		filters["subscriber_id"] = v
	}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	sis, total, err := h.service.ListSubscriberInstallments(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch subscriber installments"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: sis, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// GetSubscriberInstallment godoc
// @Summary Get subscriber installment
// @Tags Installments & Payments
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber Installment ID"
// @Success 200 {object} domain.SubscriberInstallment
// @Router /subscriber-installments/{id} [get]
func (h *InstallmentHandler) GetSubscriberInstallmentByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid ID"})
		return
	}
	claims := middleware.GetClaims(c)
	si, err := h.service.GetSubscriberInstallmentByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Subscriber installment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch subscriber installment"})
		return
	}
	c.JSON(http.StatusOK, si)
}

// ListPaymentTransactions godoc
// @Summary List payment transactions
// @Tags Installments & Payments
// @Security BearerAuth
// @Produce json
// @Param subscriber_id query int false "Filter by subscriber"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /payment-transactions [get]
func (h *InstallmentHandler) ListPaymentTransactions(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if v, err := strconv.Atoi(c.Query("subscriber_id")); err == nil && v > 0 {
		filters["subscriber_id"] = v
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	txns, total, err := h.service.ListPaymentTransactions(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch transactions"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: txns, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// RecordPayment godoc
// @Summary Record payment
// @Tags Installments & Payments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.RecordPaymentRequest true "Payment details"
// @Success 201 {object} domain.PaymentTransaction
// @Router /payment-transactions [post]
func (h *InstallmentHandler) RecordPayment(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var body struct {
		SubscriberInstallmentID int     `json:"subscriber_installment_id" binding:"required"`
		Amount                  float64 `json:"amount" binding:"required,gt=0"`
		PaymentMode             string  `json:"payment_mode" binding:"required,oneof=CASH CHEQUE NEFT RTGS UPI CARD ONLINE"`
		TransactionReference    string  `json:"transaction_reference"`
		Remarks                 string  `json:"remarks"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	req := &domain.RecordPaymentRequest{
		Amount:               body.Amount,
		PaymentMode:          body.PaymentMode,
		TransactionReference: body.TransactionReference,
		Remarks:              body.Remarks,
	}
	txn, err := h.service.RecordPayment(c.Request.Context(), body.SubscriberInstallmentID, claims.UserID, req)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Subscriber installment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to record payment"})
		return
	}
	c.JSON(http.StatusCreated, txn)
}

// GetPaymentTransactionByID godoc
// @Summary Get transaction details
// @Tags Installments & Payments
// @Security BearerAuth
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} domain.PaymentTransaction
// @Router /payment-transactions/{id} [get]
func (h *InstallmentHandler) GetPaymentTransactionByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid transaction ID"})
		return
	}
	claims := middleware.GetClaims(c)
	txn, err := h.service.GetPaymentTransactionByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch transaction"})
		return
	}
	c.JSON(http.StatusOK, txn)
}

// Create kept for backward compat
func (h *InstallmentHandler) Create(c *gin.Context) {
	h.GenerateBatch(c)
}
