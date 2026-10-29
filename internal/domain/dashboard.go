package domain

import "time"

// DashboardBundle is the full dashboard response
type DashboardBundle struct {
	KPIs               *DashboardKPIs          `json:"kpis"`
	MonthlyCollections []MonthlyCollectionItem `json:"monthly_collections"`
	SubscriberStats    *SubscriberStats        `json:"subscriber_stats"`
	UpcomingAuctions   []UpcomingAuction       `json:"upcoming_auctions"`
	BranchCollections  []BranchCollection      `json:"branch_collections"`
	RecentActivity     []RecentActivity        `json:"recent_activity"`
	LastRefreshed      time.Time               `json:"last_refreshed"`
}

// DashboardKPIs matches the exact kpis block
type DashboardKPIs struct {
	TotalCollectionLakhs    float64 `json:"total_collection_lakhs"`
	CollectionChangePercent float64 `json:"collection_change_percent"`
	ActiveChitGroups        int     `json:"active_chit_groups"`
	NewChitGroupsThisMonth  int     `json:"new_chit_groups_this_month"`
	PendingDefaults         int     `json:"pending_defaults"`
	EscalatedToLegal        int     `json:"escalated_to_legal"`
	AuctionsThisMonth       int     `json:"auctions_this_month"`
	UpcomingAuctions        int     `json:"upcoming_auctions"`
	CompletedAuctions       int     `json:"completed_auctions"`
}

// MonthlyCollectionItem matches the monthly_collections array
type MonthlyCollectionItem struct {
	Month      string  `json:"month"`
	CollectedL float64 `json:"collected_l"`
	TargetL    float64 `json:"target_l"`
	IsFuture   bool    `json:"is_future"`
}

// SubscriberStats matches the subscriber_stats block
type SubscriberStats struct {
	Total      int `json:"total"`
	Active     int `json:"active"`
	Prized     int `json:"prized"`
	Defaulters int `json:"defaulters"`
	KYCPending int `json:"kyc_pending"`
}

// UpcomingAuction matches each item in upcoming_auctions array
type UpcomingAuction struct {
	ID             string    `json:"id"`
	GroupName      string    `json:"group_name"`
	ChitValueLakhs float64   `json:"chit_value_lakhs"`
	AuctionDate    time.Time `json:"auction_date"`
	MemberCount    int       `json:"member_count"`
	Method         string    `json:"method"`
	BranchName     string    `json:"branch_name"`
}

// BranchCollection matches each item in branch_collections array
type BranchCollection struct {
	BranchID   string  `json:"branch_id"`
	BranchName string  `json:"branch_name"`
	CollectedL float64 `json:"collected_l"`
	TargetL    float64 `json:"target_l"`
}

// RecentActivity matches each item in recent_activity array
type RecentActivity struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	Timestamp time.Time `json:"timestamp"`
}

// ── Legacy types kept for report endpoints ───────────────────────────────────

type ChitPerformanceReport struct {
	ChitID           int     `json:"chit_id"`
	ChitCode         string  `json:"chit_code"`
	ChitName         string  `json:"chit_name"`
	ChitValue        float64 `json:"chit_value"`
	TotalSubscribers int     `json:"total_subscribers"`
	CurrentMonth     int     `json:"current_month"`
	TotalMonths      int     `json:"total_months"`
	TotalCollected   float64 `json:"total_collected"`
	TotalPending     float64 `json:"total_pending"`
	DefaultersCount  int     `json:"defaulters_count"`
	Status           string  `json:"status"`
}

type SubscriberLedgerEntry struct {
	Date          string  `json:"date"`
	ChitCode      string  `json:"chit_code"`
	Description   string  `json:"description"`
	MonthNumber   int     `json:"month_number"`
	AmountDue     float64 `json:"amount_due"`
	AmountPaid    float64 `json:"amount_paid"`
	Balance       float64 `json:"balance"`
	PaymentMode   *string `json:"payment_mode"`
	ReceiptNumber *string `json:"receipt_number"`
	Status        string  `json:"status"`
}

type DefaulterReport struct {
	SubscriberID    int     `json:"subscriber_id"`
	SubscriberCode  string  `json:"subscriber_code"`
	FullName        string  `json:"full_name"`
	Mobile          string  `json:"mobile"`
	ChitCode        string  `json:"chit_code"`
	ChitName        string  `json:"chit_name"`
	OverdueMonths   int     `json:"days_overdue"`
	TotalOverdue    float64 `json:"total_overdue"`
	LastPaymentDate *string `json:"last_payment_date"`
	DefaultStatus   string  `json:"default_status"`
}
