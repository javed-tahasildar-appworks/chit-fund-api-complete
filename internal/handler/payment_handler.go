package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
	rzp "chit-fund-api/pkg/razorpay"
)

type PaymentHandler struct {
	svc           *service.PaymentService
	webhookSecret string
}

func NewPaymentHandler(svc *service.PaymentService, webhookSecret string) *PaymentHandler {
	return &PaymentHandler{svc: svc, webhookSecret: webhookSecret}
}

// CreateOrder godoc
// POST /api/v1/payments/orders
// POST /api/mobile/subscriber/payments/orders (subscriber app)
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	var req domain.CreatePaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	companyID, branchID, userID := resolvePaymentCaller(c)
	order, err := h.svc.CreateInstallmentOrder(c.Request.Context(), companyID, branchID, userID, &req)
	if err != nil {
		handlePaymentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, order)
}

// VerifyPayment godoc
// POST /api/v1/payments/verify
// POST /api/mobile/subscriber/payments/verify
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	var req domain.VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	companyID, _, _ := resolvePaymentCaller(c)
	pm, err := h.svc.VerifyAndCapturePayment(c.Request.Context(), companyID, &req)
	if err != nil {
		handlePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, pm)
}

// GetPayment godoc
// GET /api/v1/payments/:id
// GET /api/mobile/subscriber/payments/:id
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid payment ID"})
		return
	}
	companyID, _, _ := resolvePaymentCaller(c)
	pm, err := h.svc.GetPayment(c.Request.Context(), id, companyID)
	if err != nil {
		handlePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, pm)
}

// ListPayments godoc
// GET /api/v1/payments
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	companyID, _, _ := resolvePaymentCaller(c)
	filters := map[string]interface{}{}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	if v, err := strconv.Atoi(c.Query("subscriber_id")); err == nil && v > 0 {
		filters["subscriber_id"] = v
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	payments, total, err := h.svc.ListPayments(c.Request.Context(), companyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch payments"})
		return
	}
	if payments == nil {
		payments = []*domain.Payment{}
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: payments, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// RazorpayWebhook godoc
// POST /webhooks/razorpay — no auth, called directly by Razorpay servers
func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Cannot read body"})
		return
	}

	signature := c.GetHeader("X-Razorpay-Signature")

	// Verify signature first — reject immediately if invalid
	if !rzp.VerifyWebhookSignature(rawBody, signature, h.webhookSecret) {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Error: "Invalid webhook signature", Code: "SIGNATURE_INVALID",
		})
		return
	}

	// Parse event type for routing (Webhook processing is Feature 3 — stub here)
	var meta struct {
		Event string `json:"event"`
	}
	_ = json.Unmarshal(rawBody, &meta)

	// Always return 200 to Razorpay — prevents retry storms even if processing fails.
	// Actual event processing happens asynchronously (Feature 3: Webhook Processing).
	c.JSON(http.StatusOK, gin.H{
		"status":    "received",
		"event":     meta.Event,
		"timestamp": time.Now().Unix(),
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

// resolvePaymentCaller works for both staff JWT and subscriber app JWT.
func resolvePaymentCaller(c *gin.Context) (companyID, branchID, userID int) {
	// Try subscriber app claims first
	if appClaims := middleware.GetAppClaims(c); appClaims != nil {
		return appClaims.CompanyID, 0, appClaims.AppUserID
	}
	// Fall back to staff JWT
	if claims := middleware.GetClaims(c); claims != nil {
		branchID = 0
		if claims.BranchID != nil {
			branchID = *claims.BranchID
		}
		return claims.CompanyID, branchID, claims.UserID
	}
	return 0, 0, 0
}

func handlePaymentError(c *gin.Context, err error) {
	if appErr, ok := err.(*domain.AppError); ok {
		switch appErr.Code {
		case "NOT_FOUND":
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		case "SIGNATURE_INVALID":
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		case "INSTALLMENT_NOT_FOUND", "IDEMPOTENCY_CHECK_FAILED",
			"GATEWAY_NOT_CONFIGURED", "CAPTURE_FAILED":
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		default:
			c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		}
		return
	}
	c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Internal payment error"})
}
