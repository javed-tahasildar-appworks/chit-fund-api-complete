package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type ChitGroupHandler struct {
	service   *service.ChitGroupService
	validator *validator.Validate
}

func NewChitGroupHandler(svc *service.ChitGroupService) *ChitGroupHandler {
	return &ChitGroupHandler{service: svc, validator: validator.New()}
}

// Create godoc
// @Summary Create chit group
// @Tags Chit Groups
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateChitGroupRequest true "Chit group details"
// @Success 201 {object} domain.ChitGroup
// @Router /chit-groups [post]
func (h *ChitGroupHandler) Create(c *gin.Context) {
	var req domain.CreateChitGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	claims := middleware.GetClaims(c)
	branchID := 0
	if claims.BranchID != nil {
		branchID = *claims.BranchID
	}
	chit, err := h.service.Create(c.Request.Context(), &req, claims.CompanyID, branchID, claims.UserID)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to create chit group"})
		return
	}
	c.JSON(http.StatusCreated, chit)
}

// List godoc
// @Summary List chit groups (paginated + filter)
// @Tags Chit Groups
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status"
// @Param branch_id query int false "Filter by branch"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /chit-groups [get]
func (h *ChitGroupHandler) List(c *gin.Context) {
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

	chits, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch chit groups", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": chits, "limit": limit, "offset": offset})
}

// GetByID godoc
// @Summary Get chit group details
// @Tags Chit Groups
// @Security BearerAuth
// @Produce json
// @Param id path int true "Chit Group ID"
// @Success 200 {object} domain.ChitGroup
// @Router /chit-groups/{id} [get]
func (h *ChitGroupHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit group ID"})
		return
	}
	claims := middleware.GetClaims(c)
	chit, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Chit group not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch chit group", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, chit)
}

// Update godoc
// @Summary Update chit group
// @Tags Chit Groups
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Chit Group ID"
// @Param body body domain.UpdateChitGroupRequest true "Update fields"
// @Success 200 {object} domain.SuccessResponse
// @Router /chit-groups/{id} [put]
func (h *ChitGroupHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit group ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UpdateChitGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Build a minimal ChitGroup with only the fields to update.
	// We do NOT call GetByID first — that was causing stale status overwrites.
	chit := &domain.ChitGroup{
		ChitID:    id,
		CompanyID: claims.CompanyID,
	}
	if req.ChitName != nil {
		chit.ChitName = *req.ChitName
	}
	if req.Status != nil {
		chit.Status = *req.Status
	}
	if req.CurrentMonth != nil {
		chit.CurrentMonth = *req.CurrentMonth
	}

	if err := h.service.Update(c.Request.Context(), chit); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Chit group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to update chit group",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Chit group updated successfully"})
}

// Close godoc
// @Summary Close/terminate chit group
// @Tags Chit Groups
// @Security BearerAuth
// @Produce json
// @Param id path int true "Chit Group ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /chit-groups/{id}/close [patch]
func (h *ChitGroupHandler) Close(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit group ID"})
		return
	}
	claims := middleware.GetClaims(c)
	// Build minimal chit object — only set status to CLOSED.
	// Repository uses COALESCE so other fields are untouched.
	chit := &domain.ChitGroup{
		ChitID:    id,
		CompanyID: claims.CompanyID,
		Status:    "CLOSED",
	}
	if err := h.service.Update(c.Request.Context(), chit); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Chit group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to close chit group",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Chit group closed successfully"})
}

// ListSubscribers godoc
// @Summary List enrolled subscribers
// @Tags Chit Groups
// @Security BearerAuth
// @Produce json
// @Param id path int true "Chit Group ID"
// @Success 200 {array} domain.ChitSubscriber
// @Router /chit-groups/{id}/subscribers [get]
func (h *ChitGroupHandler) ListSubscribers(c *gin.Context) {
	chitID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit group ID"})
		return
	}
	claims := middleware.GetClaims(c)
	subscribers, err := h.service.GetSubscribers(c.Request.Context(), chitID, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch subscribers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": subscribers})
}

// AddSubscriber godoc
// @Summary Enroll subscriber into group
// @Tags Chit Groups
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Chit Group ID"
// @Param body body domain.AddSubscriberToChitRequest true "Enrollment details"
// @Success 201 {object} domain.SuccessResponse
// @Router /chit-groups/{id}/subscribers [post]
func (h *ChitGroupHandler) AddSubscriber(c *gin.Context) {
	chitID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit group ID"})
		return
	}
	var req domain.AddSubscriberToChitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	claims := middleware.GetClaims(c)
	cs, err := h.service.AddSubscriber(c.Request.Context(), chitID, req.SubscriberID, claims.CompanyID, req.TicketCount, req.SuretyRequired)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to enroll subscriber",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":            "Subscriber enrolled successfully",
		"chit_subscriber_id": cs.ChitSubscriberID,
		"chit_id":            cs.ChitID,
		"subscriber_id":      cs.SubscriberID,
		"slot_number":        cs.SlotNumber,
		"ticket_count":       cs.TicketCount,
		"surety_required":    cs.SuretyRequired,
		"status":             cs.Status,
		"join_date":          cs.JoinDate.Format("2006-01-02"),
		"enrollment_fee":     cs.EnrollmentFee,
		"is_prized":          false,
	})
}

// RemoveEnrollment godoc
// @Summary Remove enrollment
// @Tags Chit Groups
// @Security BearerAuth
// @Produce json
// @Param id path int true "Chit Group ID"
// @Param csId path int true "Chit Subscriber ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /chit-groups/{id}/subscribers/{csId} [delete]
func (h *ChitGroupHandler) RemoveEnrollment(c *gin.Context) {
	chitID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit group ID"})
		return
	}
	csID, err := strconv.Atoi(c.Param("csId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid chit subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.RemoveEnrollment(c.Request.Context(), chitID, csID, claims.CompanyID); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Enrollment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to remove enrollment"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Enrollment removed successfully"})
}
