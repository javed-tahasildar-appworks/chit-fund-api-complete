package domain

import "time"

// ── Subscriber App Auth ───────────────────────────────────────────────────────

// AppTokenClaims is stored in JWT for subscriber app users.
// Separate from staff TokenClaims to avoid mixing the two auth systems.
type AppTokenClaims struct {
	AppUserID    int    `json:"app_user_id"`
	SubscriberID int    `json:"subscriber_id"`
	CompanyID    int    `json:"company_id"`
	AppType      string `json:"app_type"` // "subscriber"
}

// AgentTokenClaims wraps the regular staff TokenClaims but is used by agent app.
// The agent uses the existing users table + JWT — same as staff login but role is
// COLLECTION_AGENT. We reuse TokenClaims directly for agents.

// ── Subscriber App request/response types ─────────────────────────────────────

type AppLoginRequest struct {
	Mobile   string `json:"mobile"   binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type AppLoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	Subscriber   *AppSubscriberProfile `json:"subscriber"`
}

type AppSendOTPRequest struct {
	Mobile string `json:"mobile" binding:"required"`
}

type AppVerifyOTPRequest struct {
	Mobile string `json:"mobile" binding:"required"`
	OTP    string `json:"otp"    binding:"required,len=6"`
}

type AppRegisterRequest struct {
	SubscriberID int    `json:"subscriber_id" binding:"required"`
	Mobile       string `json:"mobile"        binding:"required"`
	Password     string `json:"password"      binding:"required,min=6"`
	DeviceToken  string `json:"device_token"`
	DeviceType   string `json:"device_type"   binding:"required,oneof=ANDROID IOS"`
	AppVersion   string `json:"app_version"`
}

type AppUpdateDeviceRequest struct {
	DeviceToken string `json:"device_token" binding:"required"`
	DeviceType  string `json:"device_type"  binding:"required,oneof=ANDROID IOS"`
	AppVersion  string `json:"app_version"`
}

// ── Subscriber App read models ─────────────────────────────────────────────────

// AppSubscriberProfile is the slim profile shown on app home / "My Account".
type AppSubscriberProfile struct {
	AppUserID      int        `json:"app_user_id"`
	SubscriberID   int        `json:"subscriber_id"`
	SubscriberCode string     `json:"subscriber_code"`
	FullName       string     `json:"full_name"`
	Mobile         string     `json:"mobile"`
	Email          *string    `json:"email"`
	KYCStatus      string     `json:"kyc_status"`
	IsVerified     bool       `json:"is_verified"`
	LastLogin      *time.Time `json:"last_login"`
}

// AppDashboard is returned by GET /app/subscriber/dashboard
type AppDashboard struct {
	Profile          *AppSubscriberProfile `json:"profile"`
	TotalChits       int                   `json:"total_chits"`
	ActiveChits      int                   `json:"active_chits"`
	TotalPaid        float64               `json:"total_paid"`
	TotalPending     float64               `json:"total_pending"`
	OverdueAmount    float64               `json:"overdue_amount"`
	UnreadNotifications int                `json:"unread_notifications"`
	UpcomingDueDate  *string               `json:"upcoming_due_date"`
	UpcomingDueAmount float64              `json:"upcoming_due_amount"`
}

// AppChitSummary is one row in GET /app/subscriber/chits
type AppChitSummary struct {
	ChitID            int     `json:"chit_id"`
	ChitCode          string  `json:"chit_code"`
	ChitName          string  `json:"chit_name"`
	ChitValue         float64 `json:"chit_value"`
	InstallmentAmount float64 `json:"installment_amount"`
	CurrentMonth      int     `json:"current_month"`
	TotalMonths       int     `json:"total_months"`
	Status            string  `json:"status"`
	IsPrized          bool    `json:"is_prized"`
	PrizedMonth       *int    `json:"prized_month"`
	SlotNumber        *int    `json:"slot_number"`
	NextDueDate       *string `json:"next_due_date"`
	NextDueAmount     float64 `json:"next_due_amount"`
}

// AppInstallmentDue is returned in GET /app/subscriber/dues
type AppInstallmentDue struct {
	SubscriberInstallmentID int     `json:"subscriber_installment_id"`
	ChitID                  int     `json:"chit_id"`
	ChitName                string  `json:"chit_name"`
	ChitCode                string  `json:"chit_code"`
	MonthNumber             int     `json:"month_number"`
	DueDate                 string  `json:"due_date"`
	AmountDue               float64 `json:"amount_due"`
	AmountPaid              float64 `json:"amount_paid"`
	BalanceAmount           float64 `json:"balance_amount"`
	PenaltyAmount           float64 `json:"penalty_amount"`
	TotalPayable            float64 `json:"total_payable"`
	Status                  string  `json:"status"` // PENDING | PARTIAL | OVERDUE
	IsOverdue               bool    `json:"is_overdue"`
}

// AppPaymentHistoryItem — one entry in GET /app/subscriber/payments
type AppPaymentHistoryItem struct {
	TransactionID int     `json:"transaction_id"`
	ChitName      string  `json:"chit_name"`
	MonthNumber   int     `json:"month_number"`
	Amount        float64 `json:"amount"`
	PaymentMode   string  `json:"payment_mode"`
	Reference     *string `json:"transaction_reference"`
	ReceiptNumber *string `json:"receipt_number"`
	PaymentDate   string  `json:"payment_date"`
	Status        string  `json:"status"`
	IsOnline      bool    `json:"is_online"`
}

// ── Agent App request/response types ─────────────────────────────────────────

// AgentLoginRequest reuses the staff login — agent logs in with username/password.
// Response is the standard LoginResponse with role COLLECTION_AGENT.

// AgentCollectionSummary — GET /app/agent/dashboard
type AgentCollectionSummary struct {
	AgentName           string                   `json:"agent_name"`
	TodayDate           string                   `json:"today_date"`
	TotalDueToday       float64                  `json:"total_due_today"`
	TotalCollectedToday float64                  `json:"total_collected_today"`
	PendingToCollect    float64                  `json:"pending_to_collect"`
	SubscribersVisited  int                      `json:"subscribers_visited"`
	SubscribersPending  int                      `json:"subscribers_pending"`
	RecentCollections   []*AgentRecentCollection `json:"recent_collections"`
}

// AgentRecentCollection — last N cash payments recorded by this agent today
type AgentRecentCollection struct {
	TransactionID   int     `json:"transaction_id"`
	SubscriberName  string  `json:"subscriber_name"`
	ChitName        string  `json:"chit_name"`
	Amount          float64 `json:"amount"`
	PaymentMode     string  `json:"payment_mode"`
	RecordedAt      string  `json:"recorded_at"`
}

// AgentSubscriberListItem — row in GET /app/agent/subscribers
type AgentSubscriberListItem struct {
	SubscriberID   int     `json:"subscriber_id"`
	SubscriberCode string  `json:"subscriber_code"`
	FullName       string  `json:"full_name"`
	Mobile         string  `json:"mobile"`
	Address        string  `json:"address"`
	TotalPending   float64 `json:"total_pending"`
	IsOverdue      bool    `json:"is_overdue"`
}

// AgentRecordCashPaymentRequest — POST /app/agent/collect
type AgentRecordCashPaymentRequest struct {
	SubscriberInstallmentID int     `json:"subscriber_installment_id" binding:"required"`
	Amount                  float64 `json:"amount"                    binding:"required,gt=0"`
	PaymentMode             string  `json:"payment_mode"              binding:"required,oneof=CASH CHEQUE"`
	TransactionReference    string  `json:"transaction_reference"`
	Remarks                 string  `json:"remarks"`
}
