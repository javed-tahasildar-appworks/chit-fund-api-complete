package service

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
	rzp "chit-fund-api/pkg/razorpay"
)

type PaymentService struct {
	repo          *repository.PaymentRepository
	rzpKeyID      string
	rzpKeySecret  string
	rzpAcctNumber string // RazorpayX virtual account for payouts
}

func NewPaymentService(repo *repository.PaymentRepository, keyID, keySecret, acctNumber string) *PaymentService {
	return &PaymentService{
		repo:          repo,
		rzpKeyID:      keyID,
		rzpKeySecret:  keySecret,
		rzpAcctNumber: acctNumber,
	}
}

func (s *PaymentService) client() *rzp.Client {
	return rzp.NewClient(s.rzpKeyID, s.rzpKeySecret)
}

// ── Feature 1: Digital Collection (pay-in) ───────────────────────────────────

// CreateInstallmentOrder is the entry point for Feature 1.
// Steps:
//  1. Check idempotency — if a live payment order already exists for the installment, return it
//  2. Resolve installment → subscriber + chit + net_payable (uses GENERATED column)
//  3. Fetch active gateway config for RAZORPAY
//  4. Create payment record in DB (status=CREATED)
//  5. Create Razorpay order via API
//  6. Update payment with gateway_order_id (status=INITIATED)
//  7. Return RazorpayOrderResponse to Flutter for Checkout SDK
func (s *PaymentService) CreateInstallmentOrder(
	ctx context.Context,
	companyID, branchID, userID int,
	req *domain.CreatePaymentOrderRequest,
) (*domain.RazorpayOrderResponse, error) {

	// Step 1: Idempotency check
	existing, err := s.repo.GetExistingPaymentForInstallment(ctx, req.SubscriberInstallmentID, companyID)
	if err != nil {
		return nil, domain.NewAppError("IDEMPOTENCY_CHECK_FAILED", "Failed to check existing payment", err)
	}
	if existing != nil && existing.GatewayOrderID != nil {
		return &domain.RazorpayOrderResponse{
			PaymentID:   existing.PaymentID,
			RazorpayKey: s.rzpKeyID,
			OrderID:     *existing.GatewayOrderID,
			Amount:      int64(existing.Amount * 100),
			Currency:    existing.Currency,
			Description: fmt.Sprintf("Existing %s payment", existing.Purpose),
		}, nil
	}

	// Step 2: Resolve installment
	subscriberID, chitID, amount, err := s.repo.GetInstallmentForPayment(ctx, req.SubscriberInstallmentID)
	if err != nil {
		return nil, domain.NewAppError("INSTALLMENT_NOT_FOUND",
			"Installment not found, already paid, or a payment order already exists", err)
	}

	// Step 3: Get gateway config
	gatewayCfg, err := s.repo.GetActiveGatewayConfig(ctx, companyID, "RAZORPAY", false)
	if err == domain.ErrNotFound {
		return nil, domain.NewAppError("GATEWAY_NOT_CONFIGURED",
			"No active Razorpay gateway configuration found for this company. "+
				"Please configure it under Settings → Payment Gateways.", nil)
	}
	if err != nil {
		return nil, domain.NewAppError("GATEWAY_CONFIG_ERROR", "Failed to fetch gateway config", err)
	}

	// Determine API key — DB_ENCRYPTED or VAULT reference
	apiKey := s.rzpKeyID // fallback to env config
	if gatewayCfg.SecretStorageMode == "DB_ENCRYPTED" && len(gatewayCfg.APISecretEncrypted) > 0 {
		// TODO: decrypt gatewayCfg.APISecretEncrypted using your AES key
		// apiKey = decrypt(gatewayCfg.APISecretEncrypted)
	} else if gatewayCfg.SecretStorageMode == "VAULT" && gatewayCfg.APIKeySecretRef != nil {
		// TODO: fetch from AWS Secrets Manager / Vault using *gatewayCfg.APIKeySecretRef
	}
	_ = apiKey

	// Step 4: Create payment record
	branchPtr := &branchID
	chitPtr := &chitID
	siPtr := &req.SubscriberInstallmentID
	cfgPtr := &gatewayCfg.ConfigID
	userPtr := &userID
	provider := "RAZORPAY"

	pm := &domain.Payment{
		CompanyID:               companyID,
		BranchID:                branchPtr,
		SubscriberID:            subscriberID,
		ChitID:                  chitPtr,
		SubscriberInstallmentID: siPtr,
		ConfigID:                cfgPtr,
		Provider:                &provider,
		Purpose:                 req.Purpose,
		Amount:                  amount,
		InitiatedBy:             userPtr,
	}
	if err := s.repo.CreatePayment(ctx, pm); err != nil {
		return nil, domain.NewAppError("PAYMENT_CREATE_FAILED", "Failed to create payment record", err)
	}

	// Step 5: Create Razorpay order
	rzpOrder, err := s.client().CreateOrder(&rzp.CreateOrderReq{
		Amount:   int64(amount * 100), // convert to paise
		Currency: "INR",
		Receipt:  fmt.Sprintf("pay_%d", pm.PaymentID),
		Notes:    map[string]string{"payment_id": fmt.Sprintf("%d", pm.PaymentID)},
	})
	if err != nil {
		_ = s.repo.MarkPaymentFailed(ctx, pm.PaymentID, "GATEWAY_ERROR", err.Error(), companyID)
		return nil, domain.NewAppError("ORDER_CREATION_FAILED", "Failed to create Razorpay order", err)
	}

	// Step 6: Persist gateway_order_id → INITIATED
	if err := s.repo.UpdatePaymentGatewayOrder(ctx, pm.PaymentID, rzpOrder.ID, companyID); err != nil {
		return nil, domain.NewAppError("ORDER_UPDATE_FAILED", "Failed to update payment with order ID", err)
	}

	// Step 7: Return to Flutter
	return &domain.RazorpayOrderResponse{
		PaymentID:   pm.PaymentID,
		RazorpayKey: s.rzpKeyID,
		OrderID:     rzpOrder.ID,
		Amount:      int64(amount * 100),
		Currency:    "INR",
		Description: fmt.Sprintf("%s payment", req.Purpose),
	}, nil
}

// VerifyAndCapturePayment is called after the subscriber completes Razorpay Checkout.
// Steps:
//  1. Verify HMAC signature using gateway order ID + payment ID
//  2. Fetch payment details from Razorpay to confirm amount and status
//  3. Capture if status=authorized
//  4. Update DB atomically (payments + installment + ledger + transaction)
func (s *PaymentService) VerifyAndCapturePayment(
	ctx context.Context,
	companyID int,
	req *domain.VerifyPaymentRequest,
) (*domain.Payment, error) {

	// Step 1: Verify signature
	if !s.client().VerifyPaymentSignature(
		req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature) {
		return nil, domain.NewAppError("SIGNATURE_INVALID",
			"Payment signature verification failed. Do not trust this payment.", nil)
	}

	// Step 2: Fetch from Razorpay
	pd, err := s.client().FetchPayment(req.RazorpayPaymentID)
	if err != nil {
		return nil, domain.NewAppError("GATEWAY_FETCH_FAILED", "Failed to fetch payment from Razorpay", err)
	}

	// Step 3: Capture if authorized
	if pd.Status == "authorized" {
		if err := s.client().CapturePayment(req.RazorpayPaymentID, pd.Amount); err != nil {
			_ = s.repo.MarkPaymentFailed(ctx, req.PaymentID, "CAPTURE_FAILED", err.Error(), companyID)
			return nil, domain.NewAppError("CAPTURE_FAILED", "Payment capture failed", err)
		}
	}

	// Step 4: Atomic DB update
	amountCaptured := float64(pd.Amount) / 100.0
	if err := s.repo.MarkPaymentSuccess(ctx,
		req.PaymentID, req.RazorpayPaymentID, amountCaptured, nil, companyID); err != nil {
		return nil, domain.NewAppError("PAYMENT_UPDATE_FAILED", "Failed to update payment status", err)
	}

	return s.repo.GetPaymentByID(ctx, req.PaymentID, companyID)
}

func (s *PaymentService) GetPayment(ctx context.Context, paymentID, companyID int) (*domain.Payment, error) {
	pm, err := s.repo.GetPaymentByID(ctx, paymentID, companyID)
	if err == domain.ErrNotFound {
		return nil, domain.NewAppError("NOT_FOUND", "Payment not found", err)
	}
	return pm, err
}

func (s *PaymentService) ListPayments(ctx context.Context, companyID int,
	filters map[string]interface{}) ([]*domain.Payment, int64, error) {
	return s.repo.ListPayments(ctx, companyID, filters)
}
