package domain

import "time"

// Installment matches 'installments' table exactly
type Installment struct {
	InstallmentID         int       `json:"installment_id"`
	ChitID                int       `json:"chit_id"`
	ChitName              string    `json:"chit_name"`
	MonthNumber           int       `json:"month_number"`
	BaseInstallmentAmount float64   `json:"base_installment_amount"`
	DividendAmount        float64   `json:"dividend_amount"`
	NetPayableAmount      float64   `json:"net_payable_amount"`
	DueDate               time.Time `json:"due_date"`
	GracePeriodDays       int       `json:"grace_period_days"`
	PenaltyRatePerDay     float64   `json:"penalty_rate_per_day"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
}

// SubscriberInstallment matches 'subscriber_installments' table exactly
type SubscriberInstallment struct {
	SubscriberInstallmentID int        `json:"subscriber_installment_id"`
	InstallmentID           int        `json:"installment_id"`
	ChitSubscriberID        int        `json:"chit_subscriber_id"`
	SubscriberID            int        `json:"subscriber_id"`
	SubscriberName          string     `json:"subscriber_name"`
	AmountDue               float64    `json:"amount_due"`
	AmountPaid              float64    `json:"amount_paid"`
	BalanceAmount           float64    `json:"balance_amount"`
	PenaltyAmount           float64    `json:"penalty_amount"`
	TotalPayable            float64    `json:"total_payable"`
	PaymentDate             *time.Time `json:"payment_date"`
	PaymentMode             *string    `json:"payment_mode"`
	TransactionReference    *string    `json:"transaction_reference"`
	PaymentCollectedBy      *int       `json:"payment_collected_by"`
	ReceiptNumber           *string    `json:"receipt_number"`
	Status                  string     `json:"status"`
	Remarks                 *string    `json:"remarks"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// PaymentTransaction matches 'payment_transactions' table
type PaymentTransaction struct {
	TransactionID           int       `json:"transaction_id"`
	SubscriberInstallmentID *int      `json:"subscriber_installment_id"`
	SubscriberID            int       `json:"subscriber_id"`
	SubscriberName          string    `json:"subscriber_name"`
	ChitID                  *int      `json:"chit_id"`
	ChitName                string    `json:"chit_name"`
	TransactionType         *string   `json:"transaction_type"`
	Amount                  float64   `json:"amount"`
	PaymentMode             *string   `json:"payment_mode"`
	TransactionReference    *string   `json:"transaction_reference"`
	TransactionDate         time.Time `json:"transaction_date"`
	CollectedBy             *int      `json:"collected_by"`
	BranchID                *int      `json:"branch_id"`
	BranchName              string    `json:"branch_name"`
	ReceiptNumber           *string   `json:"receipt_number"`
	Remarks                 *string   `json:"remarks"`
	Status                  string    `json:"status"`
	CreatedAt               time.Time `json:"created_at"`
}
