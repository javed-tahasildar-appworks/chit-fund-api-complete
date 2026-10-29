package domain

import (
	"encoding/json"
	"time"
)

// ── payments ──────────────────────────────────────────────────────────────────
// Matches the payments table exactly. RLS enforced on company_id.

type Payment struct {
	PaymentID               int             `json:"payment_id"                 db:"payment_id"`
	PaymentUID              string          `json:"payment_uid"                db:"payment_uid"`
	CompanyID               int             `json:"company_id"                 db:"company_id"`
	BranchID                *int            `json:"branch_id"                  db:"branch_id"`
	SubscriberID            int             `json:"subscriber_id"              db:"subscriber_id"`
	ChitID                  *int            `json:"chit_id"                    db:"chit_id"`
	SubscriberInstallmentID *int            `json:"subscriber_installment_id"  db:"subscriber_installment_id"`
	ConfigID                *int            `json:"config_id"                  db:"config_id"`
	Provider                *string         `json:"provider"                   db:"provider"`
	Purpose                 string          `json:"purpose"                    db:"purpose"`
	Amount                  float64         `json:"amount"                     db:"amount"`
	Currency                string          `json:"currency"                   db:"currency"`
	AmountCaptured          float64         `json:"amount_captured"            db:"amount_captured"`
	AmountRefunded          float64         `json:"amount_refunded"            db:"amount_refunded"`
	GatewayOrderID          *string         `json:"gateway_order_id"           db:"gateway_order_id"`
	GatewayPaymentID        *string         `json:"gateway_payment_id"         db:"gateway_payment_id"`
	Status                  string          `json:"status"                     db:"status"`
	IdempotencyKey          string          `json:"idempotency_key"            db:"idempotency_key"`
	ExpiresAt               *time.Time      `json:"expires_at"                 db:"expires_at"`
	FailureCode             *string         `json:"failure_code"               db:"failure_code"`
	FailureReason           *string         `json:"failure_reason"             db:"failure_reason"`
	Notes                   json.RawMessage `json:"notes"                      db:"notes"`
	InitiatedBy             *int            `json:"initiated_by"               db:"initiated_by"`
	CreatedAt               time.Time       `json:"created_at"                 db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"                 db:"updated_at"`

	// Joined display fields — not stored in payments table
	SubscriberName string `json:"subscriber_name,omitempty" db:"-"`
	ChitName       string `json:"chit_name,omitempty"       db:"-"`
}

// Payment status values (payments_status_check constraint)
const (
	PaymentStatusCreated           = "CREATED"
	PaymentStatusInitiated         = "INITIATED"
	PaymentStatusPending           = "PENDING"
	PaymentStatusAuthorized        = "AUTHORIZED"
	PaymentStatusCaptured          = "CAPTURED"
	PaymentStatusSuccess           = "SUCCESS"
	PaymentStatusFailed            = "FAILED"
	PaymentStatusCancelled         = "CANCELLED"
	PaymentStatusExpired           = "EXPIRED"
	PaymentStatusRefunded          = "REFUNDED"
	PaymentStatusPartiallyRefunded = "PARTIALLY_REFUNDED"
)

// ── payment_attempts ──────────────────────────────────────────────────────────

type PaymentAttempt struct {
	AttemptID        int64           `json:"attempt_id"         db:"attempt_id"`
	CompanyID        int             `json:"company_id"         db:"company_id"`
	PaymentID        int             `json:"payment_id"         db:"payment_id"`
	AttemptNumber    int             `json:"attempt_number"     db:"attempt_number"`
	Provider         *string         `json:"provider"           db:"provider"`
	PaymentMethod    *string         `json:"payment_method"     db:"payment_method"`
	GatewayAttemptID *string         `json:"gateway_attempt_id" db:"gateway_attempt_id"`
	Amount           float64         `json:"amount"             db:"amount"`
	Status           string          `json:"status"             db:"status"`
	ErrorCode        *string         `json:"error_code"         db:"error_code"`
	ErrorDescription *string         `json:"error_description"  db:"error_description"`
	RequestPayload   json.RawMessage `json:"request_payload"    db:"request_payload"`
	ResponsePayload  json.RawMessage `json:"response_payload"   db:"response_payload"`
	StartedAt        time.Time       `json:"started_at"         db:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at"       db:"completed_at"`
	CreatedAt        time.Time       `json:"created_at"         db:"created_at"`
}

// ── payment_gateway_configs ───────────────────────────────────────────────────

type PaymentGatewayConfig struct {
	ConfigID           int             `json:"config_id"                db:"config_id"`
	CompanyID          int             `json:"company_id"               db:"company_id"`
	Provider           string          `json:"provider"                 db:"provider"`
	DisplayName        *string         `json:"display_name"             db:"display_name"`
	Mode               string          `json:"mode"                     db:"mode"` // TEST | LIVE
	MerchantID         *string         `json:"merchant_id"              db:"merchant_id"`
	PublicKey          *string         `json:"public_key"               db:"public_key"`
	APIKeySecretRef    *string         `json:"api_key_secret_ref"       db:"api_key_secret_ref"`
	APISecretSecretRef *string         `json:"api_secret_secret_ref"    db:"api_secret_secret_ref"`
	WebhookSecretRef   *string         `json:"webhook_secret_ref"       db:"webhook_secret_ref"`
	SecretStorageMode  string          `json:"secret_storage_mode"      db:"secret_storage_mode"` // DB_ENCRYPTED | VAULT
	SupportsPayins     bool            `json:"supports_payins"          db:"supports_payins"`
	SupportsPayouts    bool            `json:"supports_payouts"         db:"supports_payouts"`
	Priority           int             `json:"priority"                 db:"priority"`
	IsActive           bool            `json:"is_active"                db:"is_active"`
	Metadata           json.RawMessage `json:"metadata"                 db:"metadata"`
	CreatedBy          *int            `json:"created_by"               db:"created_by"`
	CreatedAt          time.Time       `json:"created_at"               db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"               db:"updated_at"`

	// api_secret_encrypted and webhook_secret_encrypted are bytea columns.
	// Never expose these in JSON responses — omit always.
	APISecretEncrypted     []byte `json:"-" db:"api_secret_encrypted"`
	WebhookSecretEncrypted []byte `json:"-" db:"webhook_secret_encrypted"`
}

// ── payment_refunds ───────────────────────────────────────────────────────────

type PaymentRefund struct {
	RefundID        int             `json:"refund_id"         db:"refund_id"`
	RefundUID       string          `json:"refund_uid"        db:"refund_uid"`
	CompanyID       int             `json:"company_id"        db:"company_id"`
	PaymentID       int             `json:"payment_id"        db:"payment_id"`
	Amount          float64         `json:"amount"            db:"amount"`
	Currency        string          `json:"currency"          db:"currency"`
	Reason          *string         `json:"reason"            db:"reason"`
	GatewayRefundID *string         `json:"gateway_refund_id" db:"gateway_refund_id"`
	Status          string          `json:"status"            db:"status"`
	IdempotencyKey  string          `json:"idempotency_key"   db:"idempotency_key"`
	FailureReason   *string         `json:"failure_reason"    db:"failure_reason"`
	RequestedBy     *int            `json:"requested_by"      db:"requested_by"`
	ApprovedBy      *int            `json:"approved_by"       db:"approved_by"`
	Notes           json.RawMessage `json:"notes"             db:"notes"`
	CreatedAt       time.Time       `json:"created_at"        db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"        db:"updated_at"`
	ProcessedAt     *time.Time      `json:"processed_at"      db:"processed_at"`
}

// ── payout_beneficiaries ──────────────────────────────────────────────────────

type PayoutBeneficiary struct {
	BeneficiaryID        int        `json:"beneficiary_id"          db:"beneficiary_id"`
	CompanyID            int        `json:"company_id"              db:"company_id"`
	SubscriberID         int        `json:"subscriber_id"           db:"subscriber_id"`
	BankID               *int       `json:"bank_id"                 db:"bank_id"`
	Provider             string     `json:"provider"                db:"provider"`
	GatewayContactID     *string    `json:"gateway_contact_id"      db:"gateway_contact_id"`
	GatewayFundAccountID *string    `json:"gateway_fund_account_id" db:"gateway_fund_account_id"`
	BeneficiaryName      string     `json:"beneficiary_name"        db:"beneficiary_name"`
	AccountNumberMasked  *string    `json:"account_number_masked"   db:"account_number_masked"`
	IFSCCode             *string    `json:"ifsc_code"               db:"ifsc_code"`
	VPA                  *string    `json:"vpa"                     db:"vpa"`
	Status               string     `json:"status"                  db:"status"`
	IsVerified           bool       `json:"is_verified"             db:"is_verified"`
	VerifiedAt           *time.Time `json:"verified_at"             db:"verified_at"`
	CreatedAt            time.Time  `json:"created_at"              db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"              db:"updated_at"`
}

// ── payout_disbursements ──────────────────────────────────────────────────────

type PayoutDisbursement struct {
	DisbursementID  int             `json:"disbursement_id"  db:"disbursement_id"`
	DisbursementUID string          `json:"disbursement_uid" db:"disbursement_uid"`
	CompanyID       int             `json:"company_id"       db:"company_id"`
	BranchID        *int            `json:"branch_id"        db:"branch_id"`
	PayoutID        int             `json:"payout_id"        db:"payout_id"`
	SubscriberID    int             `json:"subscriber_id"    db:"subscriber_id"`
	BeneficiaryID   int             `json:"beneficiary_id"   db:"beneficiary_id"`
	ConfigID        *int            `json:"config_id"        db:"config_id"`
	Provider        *string         `json:"provider"         db:"provider"`
	Amount          float64         `json:"amount"           db:"amount"`
	Currency        string          `json:"currency"         db:"currency"`
	Mode            string          `json:"mode"             db:"mode"`
	GatewayPayoutID *string         `json:"gateway_payout_id" db:"gateway_payout_id"`
	UTR             *string         `json:"utr"              db:"utr"`
	Status          string          `json:"status"           db:"status"`
	IdempotencyKey  string          `json:"idempotency_key"  db:"idempotency_key"`
	FailureCode     *string         `json:"failure_code"     db:"failure_code"`
	FailureReason   *string         `json:"failure_reason"   db:"failure_reason"`
	InitiatedBy     *int            `json:"initiated_by"     db:"initiated_by"`
	ApprovedBy      *int            `json:"approved_by"      db:"approved_by"`
	Notes           json.RawMessage `json:"notes"            db:"notes"`
	CreatedAt       time.Time       `json:"created_at"       db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"       db:"updated_at"`
	ProcessedAt     *time.Time      `json:"processed_at"     db:"processed_at"`
}

// ── payment_webhook_events ────────────────────────────────────────────────────

type PaymentWebhookEvent struct {
	ID                int64           `json:"id"                 db:"id"`
	CompanyID         *int            `json:"company_id"         db:"company_id"`
	Provider          string          `json:"provider"           db:"provider"`
	ProviderEventID   string          `json:"provider_event_id"  db:"provider_event_id"`
	EventType         *string         `json:"event_type"         db:"event_type"`
	PaymentID         *int            `json:"payment_id"         db:"payment_id"`
	DisbursementID    *int            `json:"disbursement_id"    db:"disbursement_id"`
	Signature         *string         `json:"signature"          db:"signature"`
	SignatureVerified bool            `json:"signature_verified" db:"signature_verified"`
	Payload           json.RawMessage `json:"payload"            db:"payload"`
	ProcessingStatus  string          `json:"processing_status"  db:"processing_status"`
	ProcessingError   *string         `json:"processing_error"   db:"processing_error"`
	ReceivedAt        time.Time       `json:"received_at"        db:"received_at"`
	ProcessedAt       *time.Time      `json:"processed_at"       db:"processed_at"`
}

// ── Razorpay checkout response (not stored — returned to Flutter) ─────────────

type RazorpayOrderResponse struct {
	PaymentID      int    `json:"payment_id"`
	RazorpayKey    string `json:"razorpay_key"`
	OrderID        string `json:"order_id"`
	Amount         int64  `json:"amount"` // paise
	Currency       string `json:"currency"`
	Description    string `json:"description"`
	SubscriberName string `json:"subscriber_name"`
}
