package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type AuctionHandler struct {
	service   *service.AuctionService
	validator *validator.Validate
}

func NewAuctionHandler(svc *service.AuctionService) *AuctionHandler {
	return &AuctionHandler{service: svc, validator: validator.New()}
}

// List godoc
// @Summary List auctions (filter by group/status)
// @Tags Auctions
// @Security BearerAuth
// @Produce json
// @Param chit_id query int false "Filter by chit group"
// @Param status query string false "Filter by status"
// @Param month_number query int false "Filter by month"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /auctions [get]
func (h *AuctionHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if chitID, err := strconv.Atoi(c.Query("chit_id")); err == nil && chitID > 0 {
		filters["chit_id"] = chitID
	}
	if s := c.Query("status"); s != "" {
		filters["status"] = s
	}
	if m, err := strconv.Atoi(c.Query("month_number")); err == nil && m > 0 {
		filters["month_number"] = m
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	auctions, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch auctions"})
		return
	}
	c.JSON(http.StatusOK, domain.PaginatedResponse{
		Data: auctions, Total: total,
		Page: offset/limit + 1, PageSize: limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// Create godoc
// @Summary Schedule auction
// @Tags Auctions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateAuctionRequest true "Auction details"
// @Success 201 {object} domain.Auction
// @Router /auctions [post]
func (h *AuctionHandler) Create(c *gin.Context) {
	var req domain.CreateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	auctionDate, err := time.Parse("2006-01-02", req.AuctionDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid date format, use YYYY-MM-DD"})
		return
	}
	claims := middleware.GetClaims(c)
	auction, err := h.service.Create(c.Request.Context(), req.ChitID, claims.CompanyID, claims.UserID, auctionDate, req.AuctionTime, req.Venue)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{
				Error:   appErr.Message,
				Code:    appErr.Code,
				Details: appErr.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to schedule auction",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, auction)
}

// GetByID godoc
// @Summary Get auction details
// @Tags Auctions
// @Security BearerAuth
// @Produce json
// @Param id path int true "Auction ID"
// @Success 200 {object} domain.Auction
// @Router /auctions/{id} [get]
func (h *AuctionHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	claims := middleware.GetClaims(c)
	auction, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Auction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch auction"})
		return
	}
	c.JSON(http.StatusOK, auction)
}

// Update godoc
// @Summary Update auction details
// @Tags Auctions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Auction ID"
// @Param body body domain.UpdateAuctionRequest true "Update fields"
// @Success 200 {object} domain.SuccessResponse
// @Router /auctions/{id} [put]
func (h *AuctionHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UpdateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}
	if err := h.service.Update(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to update auction"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Auction updated successfully"})
}

// Conduct godoc
// @Summary Mark auction as conducted
// @Tags Auctions
// @Security BearerAuth
// @Produce json
// @Param id path int true "Auction ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /auctions/{id}/conduct [patch]
func (h *AuctionHandler) Conduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.Conduct(c.Request.Context(), id, claims.CompanyID, claims.UserID); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to mark auction as conducted"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Auction marked as conducted"})
}

// PostResult godoc
// @Summary Post winner + discount
// @Tags Auctions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Auction ID"
// @Param body body domain.PostResultRequest true "Winner and discount details"
// @Success 200 {object} domain.SuccessResponse
// @Router /auctions/{id}/post-result [patch]
func (h *AuctionHandler) PostResult(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.PostResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.Finalize(c.Request.Context(), id, claims.CompanyID, claims.UserID, &domain.FinalizeAuctionRequest{
		WinningBidID: req.WinningBidID,
		Remarks:      req.Remarks,
	}); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to post auction result"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Auction result posted successfully"})
}

// ListBids godoc
// @Summary List bids for auction
// @Tags Auctions
// @Security BearerAuth
// @Produce json
// @Param id path int true "Auction ID"
// @Success 200 {array} domain.AuctionBid
// @Router /auctions/{id}/bids [get]
func (h *AuctionHandler) ListBids(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	claims := middleware.GetClaims(c)
	bids, err := h.service.GetBids(c.Request.Context(), id, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch bids"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bids})
}

// SubmitBid godoc
// @Summary Submit bid
// @Tags Auctions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Auction ID"
// @Param body body domain.PlaceBidRequest true "Bid details"
// @Success 201 {object} domain.SuccessResponse
// @Router /auctions/{id}/bids [post]
func (h *AuctionHandler) SubmitBid(c *gin.Context) {
	auctionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	var req domain.PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.PlaceBid(c.Request.Context(), auctionID, claims.CompanyID, &req); err != nil {
		switch err {
		case domain.ErrAuctionClosed:
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Auction is closed"})
		case domain.ErrInvalidBid:
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Bid exceeds maximum allowed discount"})
		default:
			c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to submit bid"})
		}
		return
	}
	c.JSON(http.StatusCreated, domain.SuccessResponse{Message: "Bid submitted successfully"})
}

// MarkBidWinner godoc
// @Summary Mark bid as winner
// @Tags Auctions
// @Security BearerAuth
// @Produce json
// @Param id path int true "Auction ID"
// @Param bidId path int true "Bid ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /auctions/{id}/bids/{bidId}/win [patch]
func (h *AuctionHandler) MarkBidWinner(c *gin.Context) {
	auctionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	bidID, err := strconv.Atoi(c.Param("bidId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid bid ID"})
		return
	}
	claims := middleware.GetClaims(c)
	req := &domain.FinalizeAuctionRequest{WinningBidID: bidID}
	if err := h.service.Finalize(c.Request.Context(), auctionID, claims.CompanyID, claims.UserID, req); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to mark bid as winner"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Bid marked as winner"})
}

// CancelAuction godoc
// @Summary Cancel an auction
// @Tags Auctions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Auction ID"
// @Param body body object false "Optional remarks"
// @Success 200 {object} domain.SuccessResponse
// @Router /auctions/{id}/cancel [patch]
func (h *AuctionHandler) Cancel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid auction ID"})
		return
	}
	claims := middleware.GetClaims(c)

	var body struct {
		Remarks string `json:"remarks"`
	}
	c.ShouldBindJSON(&body) // optional body

	if err := h.service.Cancel(c.Request.Context(), id, claims.CompanyID, claims.UserID, body.Remarks); err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to cancel auction",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Auction cancelled successfully"})
}

// Finalize is kept for backward compatibility
func (h *AuctionHandler) Finalize(c *gin.Context) {
	h.PostResult(c)
}

// PlaceBid is kept for backward compatibility
func (h *AuctionHandler) PlaceBid(c *gin.Context) {
	h.SubmitBid(c)
}
