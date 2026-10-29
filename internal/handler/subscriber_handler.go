package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/service"
)

type SubscriberHandler struct {
	service *service.SubscriberService
}

func NewSubscriberHandler(svc *service.SubscriberService) *SubscriberHandler {
	return &SubscriberHandler{service: svc}
}

// List godoc
// @Summary List subscribers (paginated + filter)
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Produce json
// @Param search query string false "Search by name/mobile/code"
// @Param kyc_status query string false "Filter by KYC status"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} domain.PaginatedResponse
// @Router /subscribers [get]
func (h *SubscriberHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	filters := map[string]interface{}{}
	if s := c.Query("search"); s != "" {
		filters["search"] = s
	}
	if k := c.Query("kyc_status"); k != "" {
		filters["kyc_status"] = k
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filters["limit"] = limit
	filters["offset"] = offset

	subs, total, err := h.service.List(c.Request.Context(), claims.CompanyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to fetch subscribers",
			Details: err.Error(), // exposes exact DB error — remove in production
		})
		return
	}
	// Response shaped to match Flutter PaginatedSubscribersModel.fromJson
	c.JSON(http.StatusOK, domain.PaginatedSubscribers{
		Items:      subs,
		TotalCount: total,
		Page:       offset/limit + 1,
		Limit:      limit,
	})
}

// Create godoc
// @Summary Create subscriber
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateSubscriberRequest true "Subscriber details"
// @Success 201 {object} domain.Subscriber
// @Router /subscribers [post]
func (h *SubscriberHandler) Create(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req domain.CreateSubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	sub, err := h.service.Create(c.Request.Context(), claims.CompanyID, claims.UserID, &req)
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
			Error:   "Failed to create subscriber",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

// GetByID godoc
// @Summary Get subscriber profile
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber ID"
// @Success 200 {object} domain.Subscriber
// @Router /subscribers/{id} [get]
func (h *SubscriberHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	sub, err := h.service.GetByID(c.Request.Context(), id, claims.CompanyID)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Subscriber not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to fetch subscriber",
			Details: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// Update godoc
// @Summary Update subscriber
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Subscriber ID"
// @Param body body domain.UpdateSubscriberRequest true "Update fields"
// @Success 200 {object} domain.SuccessResponse
// @Router /subscribers/{id} [put]
func (h *SubscriberHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UpdateSubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.Update(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to update subscriber"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Subscriber updated successfully"})
}

// Blacklist godoc
// @Summary Blacklist subscriber
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Subscriber ID"
// @Param body body domain.BlacklistRequest true "Reason"
// @Success 200 {object} domain.SuccessResponse
// @Router /subscribers/{id}/blacklist [patch]
func (h *SubscriberHandler) Blacklist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.BlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Reason is required", Details: err.Error()})
		return
	}
	if err := h.service.Blacklist(c.Request.Context(), id, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to blacklist subscriber"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Subscriber blacklisted"})
}

// Unblacklist godoc
// @Summary Remove blacklist
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /subscribers/{id}/unblacklist [patch]
func (h *SubscriberHandler) Unblacklist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.Unblacklist(c.Request.Context(), id, claims.CompanyID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to remove blacklist"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Blacklist removed"})
}

// ListDocuments godoc
// @Summary List KYC documents
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber ID"
// @Success 200 {array} domain.SubscriberDocument
// @Router /subscribers/{id}/documents [get]
func (h *SubscriberHandler) ListDocuments(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	docs, err := h.service.ListDocuments(c.Request.Context(), id, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch documents", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs})
}

// UploadDocument godoc
// @Summary Upload KYC document
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Subscriber ID"
// @Param body body domain.SubscriberDocument true "Document metadata"
// @Success 201 {object} domain.SubscriberDocument
// @Router /subscribers/{id}/documents [post]
func (h *SubscriberHandler) UploadDocument(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.UploadDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}
	doc, err := h.service.UploadDocument(c.Request.Context(), id, claims.CompanyID, &req)
	if err == domain.ErrNotFound {
		c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "Subscriber not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Error:   "Failed to upload document",
			Details: err.Error(),
		})
		return
	}
	// Null-free response — all string fields default to ""
	c.JSON(http.StatusCreated, gin.H{
		"document_id":         doc.DocumentID,
		"subscriber_id":       doc.SubscriberID,
		"document_type":       doc.DocumentType,
		"document_path":       doc.FilePath,
		"document_number":     doc.DocumentNumber,
		"expiry_date":         doc.ExpiryDate,
		"verification_status": doc.VerificationStatus,
		"verified_by":         doc.VerifiedByText,
		"uploaded_date":       "",
		"created_at":          doc.CreatedAt,
	})
}

// VerifyDocument godoc
// @Summary Verify KYC document
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Subscriber ID"
// @Param docId path int true "Document ID"
// @Param body body domain.VerifyDocumentRequest true "Verification result"
// @Success 200 {object} domain.SuccessResponse
// @Router /subscribers/{id}/documents/{docId}/verify [patch]
func (h *SubscriberHandler) VerifyDocument(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	docID, err := strconv.Atoi(c.Param("docId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid document ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.VerifyDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	if err := h.service.VerifyDocument(c.Request.Context(), docID, subscriberID, claims.UserID, claims.CompanyID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to verify document"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Document " + req.VerificationStatus})
}

// ListBanks godoc
// @Summary List bank accounts
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber ID"
// @Success 200 {array} domain.SubscriberBank
// @Router /subscribers/{id}/banks [get]
func (h *SubscriberHandler) ListBanks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	banks, err := h.service.ListBankAccounts(c.Request.Context(), id, claims.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to fetch bank accounts", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": banks})
}

// AddBank godoc
// @Summary Add bank account
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Subscriber ID"
// @Param body body domain.AddBankAccountRequest true "Bank details"
// @Success 201 {object} domain.SubscriberBank
// @Router /subscribers/{id}/banks [post]
func (h *SubscriberHandler) AddBank(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	claims := middleware.GetClaims(c)
	var req domain.AddBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body", Details: err.Error()})
		return
	}
	bank, err := h.service.AddBankAccount(c.Request.Context(), id, claims.CompanyID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to add bank account"})
		return
	}
	c.JSON(http.StatusCreated, bank)
}

// DeleteBank godoc
// @Summary Remove bank account
// @Tags Subscribers & KYC
// @Security BearerAuth
// @Produce json
// @Param id path int true "Subscriber ID"
// @Param bankId path int true "Bank ID"
// @Success 200 {object} domain.SuccessResponse
// @Router /subscribers/{id}/banks/{bankId} [delete]
func (h *SubscriberHandler) DeleteBank(c *gin.Context) {
	subscriberID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid subscriber ID"})
		return
	}
	bankID, err := strconv.Atoi(c.Param("bankId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid bank ID"})
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.service.DeleteBankAccount(c.Request.Context(), bankID, subscriberID, claims.CompanyID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "Failed to delete bank account"})
		return
	}
	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Bank account removed"})
}
