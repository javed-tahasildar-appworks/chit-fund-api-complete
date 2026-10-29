package domain

import (
	"time"
)

// ── Staff Notification ────────────────────────────────────────────────────────
// Matches staff_notifications table. Separate from AppNotification (subscriber app).

type StaffNotification struct {
	NotificationID    int64      `json:"notification_id"     db:"notification_id"`
	NotificationUID   string     `json:"notification_uid"    db:"notification_uid"`
	CompanyID         int        `json:"company_id"          db:"company_id"`
	Type              string     `json:"type"                db:"type"`
	Severity          string     `json:"severity"            db:"severity"`
	Title             string     `json:"title"               db:"title"`
	Body              *string    `json:"body"                db:"body"`
	IsRead            bool       `json:"is_read"             db:"-"` // computed per user
	ActionRoute       *string    `json:"action_route"        db:"action_route"`
	RelatedEntityType *string    `json:"related_entity_type" db:"related_entity_type"`
	RelatedEntityID   *int       `json:"related_entity_id"   db:"related_entity_id"`
	BranchID          *int       `json:"branch_id"           db:"branch_id"`
	CreatedAt         time.Time  `json:"created_at"          db:"created_at"`
	ReadAt            *time.Time `json:"read_at"             db:"read_at"` // from notification_reads join
}

// Notification type constants — match the contract §5
const (
	NotifTypeInstallmentOverdue     = "INSTALLMENT_OVERDUE"
	NotifTypeInstallmentDueSoon     = "INSTALLMENT_DUE_SOON"
	NotifTypeAuctionScheduled       = "AUCTION_SCHEDULED"
	NotifTypeAuctionDueToday        = "AUCTION_DUE_TODAY"
	NotifTypePayoutAwaitingApproval = "PAYOUT_AWAITING_APPROVAL"
	NotifTypePayoutDisbursed        = "PAYOUT_DISBURSED"
	NotifTypePaymentReceived        = "PAYMENT_RECEIVED"
	NotifTypePaymentFailed          = "PAYMENT_FAILED"
	NotifTypeDefaultFlagged         = "DEFAULT_FLAGGED"
	NotifTypeLegalEscalated         = "LEGAL_ESCALATED"
	NotifTypeKYCPending             = "KYC_PENDING"
	NotifTypeSystem                 = "SYSTEM"
)

// Default severity per type — used when creating notifications server-side
var NotifDefaultSeverity = map[string]string{
	NotifTypeInstallmentOverdue:     "WARNING",
	NotifTypeInstallmentDueSoon:     "INFO",
	NotifTypeAuctionScheduled:       "INFO",
	NotifTypeAuctionDueToday:        "WARNING",
	NotifTypePayoutAwaitingApproval: "WARNING",
	NotifTypePayoutDisbursed:        "INFO",
	NotifTypePaymentReceived:        "INFO",
	NotifTypePaymentFailed:          "WARNING",
	NotifTypeDefaultFlagged:         "CRITICAL",
	NotifTypeLegalEscalated:         "CRITICAL",
	NotifTypeKYCPending:             "INFO",
	NotifTypeSystem:                 "INFO",
}

// Default action route per type
var NotifDefaultActionRoute = map[string]string{
	NotifTypeInstallmentOverdue:     "/defaults",
	NotifTypeInstallmentDueSoon:     "/installments",
	NotifTypeAuctionScheduled:       "/auctions",
	NotifTypeAuctionDueToday:        "/auctions",
	NotifTypePayoutAwaitingApproval: "/payouts",
	NotifTypePayoutDisbursed:        "/payouts",
	NotifTypePaymentReceived:        "/installments",
	NotifTypePaymentFailed:          "/installments",
	NotifTypeDefaultFlagged:         "/defaults",
	NotifTypeLegalEscalated:         "/legal",
	NotifTypeKYCPending:             "/subscribers",
	NotifTypeSystem:                 "",
}

// ── Request/response types ────────────────────────────────────────────────────

type NotificationListFilters struct {
	Status string // "unread" | "read" | "all" (default)
	Type   string // filter by notification type
	Limit  int
	Offset int
}

type NotificationUnreadCount struct {
	Unread int `json:"unread"`
}

type ReadAllRequest struct {
	Type string `json:"type"` // optional — scope to one type
}

type ReadAllResponse struct {
	MarkedRead int `json:"marked_read"`
}

// ── Internal create helper (used by domain event publishers) ─────────────────
// Not exposed as an HTTP endpoint in v1.

type CreateStaffNotificationRequest struct {
	CompanyID         int
	UserID            *int    // target specific user
	Role              *string // target by role name
	BranchID          *int    // target by branch
	Type              string
	Severity          string // defaults to NotifDefaultSeverity[Type] if empty
	Title             string
	Body              string
	ActionRoute       string
	RelatedEntityType string
	RelatedEntityID   *int
}

// AppNotification lives in app_user.go — subscriber mobile app only.
// Do not redeclare it here.
