package domain

// ── Inbound payment (pay-in) ───────────────────────────────────────────────────

type CreatePaymentOrderRequest struct {
	SubscriberInstallmentID int    `json:"subscriber_installment_id" binding:"required"`
	Purpose                 string `json:"purpose" binding:"required,oneof=INSTALLMENT PENALTY ENROLLMENT_FEE SECURITY_DEPOSIT ADVANCE OTHER"`
}

type VerifyPaymentRequest struct {
	PaymentID         int    `json:"payment_id"          binding:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id"   binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature"  binding:"required"`
}

// ── Refunds ───────────────────────────────────────────────────────────────────

type CreateRefundRequest struct {
	PaymentID int     `json:"payment_id" binding:"required"`
	Amount    float64 `json:"amount"     binding:"required,gt=0"`
	Reason    string  `json:"reason"     binding:"required,min=5"`
}

// ── Beneficiaries & Disbursements ─────────────────────────────────────────────

type CreateBeneficiaryRequest struct {
	SubscriberID    int     `json:"subscriber_id"    binding:"required"`
	BankID          *int    `json:"bank_id"`
	BeneficiaryName string  `json:"beneficiary_name" binding:"required,min=3"`
	IFSCCode        *string `json:"ifsc_code"`
	VPA             *string `json:"vpa"`
}

type InitiateDisbursementRequest struct {
	PayoutID      int    `json:"payout_id"      binding:"required"`
	BeneficiaryID int    `json:"beneficiary_id" binding:"required"`
	Mode          string `json:"mode"           binding:"required,oneof=IMPS NEFT RTGS UPI BANK_TRANSFER"`
}

// ── Gateway config management ─────────────────────────────────────────────────

type CreateGatewayConfigRequest struct {
	Provider          string `json:"provider"           binding:"required,oneof=RAZORPAY CASHFREE STRIPE PAYU PHONEPE BILLDESK OTHER"`
	DisplayName       string `json:"display_name"`
	Mode              string `json:"mode"               binding:"required,oneof=TEST LIVE"`
	MerchantID        string `json:"merchant_id"`
	PublicKey         string `json:"public_key"`
	APIKeySecretRef   string `json:"api_key_secret_ref"`
	WebhookSecretRef  string `json:"webhook_secret_ref"`
	SecretStorageMode string `json:"secret_storage_mode" binding:"required,oneof=DB_ENCRYPTED VAULT"`
	SupportsPayins    bool   `json:"supports_payins"`
	SupportsPayouts   bool   `json:"supports_payouts"`
	Priority          int    `json:"priority"`
}
