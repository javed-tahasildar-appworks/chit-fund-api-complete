package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// caller extracts the three identity fields needed for visibility queries.
func (h *NotificationHandler) caller(c *gin.Context) (companyID, userID int, roleName string, branchID *int) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return
	}
	return claims.CompanyID, claims.UserID, claims.RoleName, claims.BranchID
}

// GET /api/v1/notifications
func (h *NotificationHandler) List(c *gin.Context) {
	companyID, userID, roleName, branchID := h.caller(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	filters := domain.NotificationListFilters{
		Status: c.DefaultQuery("status", "all"),
		Type:   c.Query("type"),
		Limit:  limit,
		Offset: offset,
	}

	notifs, total, err := h.svc.List(c.Request.Context(), companyID, userID, roleName, branchID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch notifications"})
		return
	}

	page := 1
	if limit > 0 {
		page = offset/limit + 1
	}
	totalPages := 1
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data:       notifs,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	})
}

// GET /api/v1/notifications/count
func (h *NotificationHandler) Count(c *gin.Context) {
	companyID, userID, roleName, branchID := h.caller(c)
	count, err := h.svc.UnreadCount(c.Request.Context(), companyID, userID, roleName, branchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch count"})
		return
	}
	c.JSON(http.StatusOK, domain.NotificationUnreadCount{Unread: count})
}

// POST /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	nid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid notification ID"})
		return
	}
	companyID, userID, roleName, branchID := h.caller(c)

	notif, err := h.svc.MarkRead(c.Request.Context(), nid, companyID, userID, roleName, branchID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Notification not found or not visible to this user"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to mark notification read"})
		return
	}
	c.JSON(http.StatusOK, notif)
}

// POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	companyID, userID, roleName, branchID := h.caller(c)

	var body domain.ReadAllRequest
	_ = c.ShouldBindJSON(&body) // optional body — ignore bind error

	count, err := h.svc.MarkAllRead(c.Request.Context(), companyID, userID, roleName, branchID, body.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to mark all read"})
		return
	}
	c.JSON(http.StatusOK, domain.ReadAllResponse{MarkedRead: count})
}

// POST /api/v1/notifications/:id/dismiss
func (h *NotificationHandler) Dismiss(c *gin.Context) {
	nid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid notification ID"})
		return
	}
	companyID, userID, roleName, branchID := h.caller(c)

	if err := h.svc.Dismiss(c.Request.Context(), nid, companyID, userID, roleName, branchID); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Notification not found or not visible to this user"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to dismiss notification"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dismissed": true})
}
