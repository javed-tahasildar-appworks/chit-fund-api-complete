package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type MobileHandler struct {
	mobileService *service.MobileService
	authService   *service.AuthService // reused for agent login
}

func NewMobileHandler(mobileSvc *service.MobileService, authSvc *service.AuthService) *MobileHandler {
	return &MobileHandler{mobileService: mobileSvc, authService: authSvc}
}

// ────────────────────────────────────────────────────────────────────────────
// SUBSCRIBER APP — Auth
// ────────────────────────────────────────────────────────────────────────────

// AppRegister handles POST /app/subscriber/register
// Subscriber registers with their existing subscriber_id + mobile + password.
func (h *MobileHandler) AppRegister(c *gin.Context) {
	var req domain.AppRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	if err := h.mobileService.Register(c.Request.Context(), &req); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Registration failed"})
		return
	}
	c.JSON(http.StatusCreated, domain.SuccessResponse{Message: "Registration successful. Please verify your mobile number."})
}

// AppLogin handles POST /app/subscriber/login
func (h *MobileHandler) AppLogin(c *gin.Context) {
	var req domain.AppLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	resp, err := h.mobileService.Login(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Login failed"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// AppSendOTP handles POST /app/subscriber/send-otp
func (h *MobileHandler) AppSendOTP(c *gin.Context) {
	var req domain.AppSendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	otp, err := h.mobileService.SendOTP(c.Request.Context(), req.Mobile)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to send OTP"})
		return
	}

	// In production: send OTP via SMS provider and do NOT return it in response.
	// For development/testing, the OTP is included in the response.
	// TODO: Integrate SMS gateway (Twilio, MSG91, Exotel) and remove otp from response.
	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent to your mobile number",
		"otp":     otp, // REMOVE IN PRODUCTION
	})
}

// AppVerifyOTP handles POST /app/subscriber/verify-otp
func (h *MobileHandler) AppVerifyOTP(c *gin.Context) {
	var req domain.AppVerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	if err := h.mobileService.VerifyOTP(c.Request.Context(), &req); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "OTP verification failed"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Mobile number verified successfully"})
}

// AppUpdateDevice handles PUT /app/subscriber/device
// Call this on every app launch to keep the device_token fresh for push notifications.
func (h *MobileHandler) AppUpdateDevice(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	var req domain.AppUpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	if err := h.mobileService.UpdateDevice(c.Request.Context(), appClaims.AppUserID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to update device"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Device updated"})
}

// ────────────────────────────────────────────────────────────────────────────
// SUBSCRIBER APP — Data (all routes require AppSubscriberAuth)
// ────────────────────────────────────────────────────────────────────────────

// AppGetDashboard handles GET /app/subscriber/dashboard
func (h *MobileHandler) AppGetDashboard(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	dash, err := h.mobileService.GetDashboard(c.Request.Context(), appClaims.AppUserID, appClaims.SubscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch dashboard"})
		return
	}
	c.JSON(http.StatusOK, dash)
}

// AppGetChits handles GET /app/subscriber/chits
func (h *MobileHandler) AppGetChits(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	chits, err := h.mobileService.GetChits(c.Request.Context(), appClaims.SubscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch chits"})
		return
	}
	if chits == nil {
		chits = []*domain.AppChitSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"data": chits, "total": len(chits)})
}

// AppGetDues handles GET /app/subscriber/dues
// Returns pending/overdue installments — the "Pay Now" screen data.
func (h *MobileHandler) AppGetDues(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	dues, err := h.mobileService.GetDues(c.Request.Context(), appClaims.SubscriberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch dues"})
		return
	}
	if dues == nil {
		dues = []*domain.AppInstallmentDue{}
	}
	c.JSON(http.StatusOK, gin.H{"data": dues, "total": len(dues)})
}

// AppGetPaymentHistory handles GET /app/subscriber/payments
func (h *MobileHandler) AppGetPaymentHistory(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.mobileService.GetPaymentHistory(c.Request.Context(),
		appClaims.SubscriberID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch payment history"})
		return
	}
	if items == nil {
		items = []*domain.AppPaymentHistoryItem{}
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: items, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// AppGetNotifications handles GET /app/subscriber/notifications
func (h *MobileHandler) AppGetNotifications(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifs, total, err := h.mobileService.GetNotifications(c.Request.Context(),
		appClaims.SubscriberID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch notifications"})
		return
	}
	if notifs == nil {
		notifs = []*domain.AppNotification{}
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: notifs, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// AppMarkNotificationRead handles PATCH /app/subscriber/notifications/:id/read
func (h *MobileHandler) AppMarkNotificationRead(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid notification ID"})
		return
	}
	if err := h.mobileService.MarkNotificationRead(c.Request.Context(), id, appClaims.SubscriberID); err != nil {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Marked as read"})
}

// AppMarkAllNotificationsRead handles PATCH /app/subscriber/notifications/read-all
func (h *MobileHandler) AppMarkAllNotificationsRead(c *gin.Context) {
	appClaims := middleware.GetAppClaims(c)
	_ = h.mobileService.MarkAllNotificationsRead(c.Request.Context(), appClaims.SubscriberID)
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "All notifications marked as read"})
}

// ────────────────────────────────────────────────────────────────────────────
// AGENT APP — Auth
// ────────────────────────────────────────────────────────────────────────────

// AgentLogin handles POST /app/agent/login
// Reuses existing staff auth — same JWT, but we check role_name = COLLECTION_AGENT.
func (h *MobileHandler) AgentLogin(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}
	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Login failed"})
		return
	}
	// Validate role — only agent-eligible roles can use agent app
	allowedRoles := map[string]bool{
		"COLLECTION_AGENT": true, "BRANCH_MANAGER": true,
		"ADMIN": true, "SUPER_ADMIN": true,
	}
	if !allowedRoles[resp.User.RoleName] {
		c.JSON(http.StatusForbidden, domain.ErrorResponse{
			Error: "Access denied. This login is for collection agents only.",
			Code:  "FORBIDDEN",
		})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ────────────────────────────────────────────────────────────────────────────
// AGENT APP — Data (all routes require standard AuthRequired + RequireAgentRole)
// ────────────────────────────────────────────────────────────────────────────

// AgentDashboard handles GET /app/agent/dashboard
func (h *MobileHandler) AgentDashboard(c *gin.Context) {
	claims := middleware.GetClaims(c)
	dash, err := h.mobileService.GetAgentDashboard(c.Request.Context(), claims.UserID, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch dashboard"})
		return
	}
	c.JSON(http.StatusOK, dash)
}

// AgentGetSubscribers handles GET /app/agent/subscribers
// Supports ?search=, ?overdue=true, ?limit=, ?offset=
func (h *MobileHandler) AgentGetSubscribers(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if s := c.Query("search"); s != "" {
		filters["search"] = s
	}
	if c.Query("overdue") == "true" {
		filters["overdue"] = true
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	items, total, err := h.mobileService.GetSubscriberList(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch subscribers"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: items, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// AgentCollect handles POST /app/agent/collect
// Records a cash or cheque payment collected in the field.
func (h *MobileHandler) AgentCollect(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.AgentRecordCashPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request", Details: err.Error()})
		return
	}

	branchID := 0
	if claims.BranchID != nil {
		branchID = *claims.BranchID
	}

	result, err := h.mobileService.RecordCashPayment(c.Request.Context(),
		&req, claims.UserID, claims.CompanyID, branchID)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to record payment"})
		return
	}
	c.JSON(http.StatusCreated, domain.SuccessResponse{
		Message: "Payment recorded successfully",
		Data:    result,
	})
}
