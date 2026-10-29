package domain

import "time"

// ChitGroup matches your 'chit_groups' table exactly
type ChitGroup struct {
	ChitID                   int        `json:"chit_id"`
	CompanyID                int        `json:"company_id"`
	BranchID                 *int       `json:"branch_id"`
	ChitCode                 string     `json:"chit_code"`
	ChitName                 string     `json:"chit_name"`
	ChitValue                float64    `json:"chit_value"`
	TotalSubscribers         int        `json:"total_subscribers"`
	InstallmentAmount        float64    `json:"installment_amount"`
	TotalMonths              int        `json:"total_months"`
	StartDate                time.Time  `json:"start_date"`
	EndDate                  *time.Time `json:"end_date"`
	AuctionDayOfMonth        *int       `json:"auction_day_of_month"`
	AuctionTime              *string    `json:"auction_time"` // TIME type stored as string
	AuctionType              string     `json:"auction_type"`
	InstallmentType          string     `json:"installment_type"`
	ForemanCommissionPercent float64    `json:"foreman_commission_percent"`
	MaxDiscountPercent       float64    `json:"max_discount_percent"`
	SecurityDeposit          float64    `json:"security_deposit"`
	Status                   string     `json:"status"`
	CurrentMonth             int        `json:"current_month"`
	ForemanUserID            *int       `json:"foreman_user_id"`
	CreatedBy                *int       `json:"created_by"`
	CreatedByName            string     `json:"created_by_name"` // joined from users table
	ForemanName              string     `json:"foreman_name"`    // joined from users table
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// ChitSubscriber matches 'chit_subscribers' table
type ChitSubscriber struct {
	ChitSubscriberID       int        `json:"chit_subscriber_id"`
	ChitID                 int        `json:"chit_id"`
	ChitName               string     `json:"chit_name"`
	SubscriberID           int        `json:"subscriber_id"`
	SubscriberName         string     `json:"subscriber_name"`
	JoinDate               time.Time  `json:"join_date"`
	SlotNumber             *int       `json:"slot_number"`
	TicketCount            int        `json:"ticket_count"`
	IsPrized               bool       `json:"is_prized"`
	PrizedMonth            *int       `json:"prized_month"`
	PrizedDate             *time.Time `json:"prized_date"`
	SuretyRequired         bool       `json:"surety_required"`
	Status                 string     `json:"status"`
	DroppedDate            *time.Time `json:"dropped_date"`
	DropReason             *string    `json:"drop_reason"`
	TransferToSubscriberID *int       `json:"transfer_to_subscriber_id"`
	TransferDate           *time.Time `json:"transfer_date"`
	EnrollmentFee          float64    `json:"enrollment_fee"`
	CreatedBy              *int       `json:"created_by"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}
