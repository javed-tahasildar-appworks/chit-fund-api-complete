package domain

import "time"

// Payout matches 'payouts' table exactly
type Payout struct {
	PayoutID             int        `json:"payout_id"`
	CompanyID            int        `json:"company_id"`
	ChitID               int        `json:"chit_id"`
	ChitCode             string     `json:"chit_code"`
	ChitName             string     `json:"chit_name"`
	AuctionID            int        `json:"auction_id"`
	ChitSubscriberID     int        `json:"chit_subscriber_id"`
	SubscriberID         int        `json:"subscriber_id"`
	SubscriberName       string     `json:"subscriber_name"`
	MonthNumber          int        `json:"month_number"`
	GrossChitValue       float64    `json:"gross_chit_value"`
	DiscountAmount       float64    `json:"discount_amount"`
	ForemanCommission    float64    `json:"foreman_commission"`
	NetPayout            float64    `json:"net_payout"`
	PayoutDate           *time.Time `json:"payout_date"`
	PayoutMode           *string    `json:"payout_mode"`
	TransactionReference *string    `json:"transaction_reference"`
	BankID               *int       `json:"bank_id"`
	PaidBy               *int       `json:"paid_by"`
	PaidByName           *string    `json:"paid_by_name"`
	ReceiptNumber        *string    `json:"receipt_number"`
	Status               string     `json:"status"`
	ApprovedBy           *int       `json:"approved_by"`
	ApprovedByName       *string    `json:"approved_by_name"`
	ApprovalDate         *time.Time `json:"approval_date"`
	Remarks              *string    `json:"remarks"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Dividend matches 'dividends' table exactly
type Dividend struct {
	DividendID            int        `json:"dividend_id"`
	AuctionID             int        `json:"auction_id"`
	ChitID                int        `json:"chit_id"`
	ChitCode              string     `json:"chit_code"`
	ChitName              string     `json:"chit_name"`
	MonthNumber           int        `json:"month_number"`
	TotalDiscountAmount   float64    `json:"total_discount_amount"`
	ForemanCommission     float64    `json:"foreman_commission"`
	DistributableAmount   float64    `json:"distributable_amount"`
	TotalSubscribers      int        `json:"total_subscribers"`
	DividendPerSubscriber float64    `json:"dividend_per_subscriber"`
	DistributionDate      *time.Time `json:"distribution_date"`
	Status                string     `json:"status"`
	CreatedAt             time.Time  `json:"created_at"`
}
