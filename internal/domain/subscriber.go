package domain

import "time"

// Subscriber matches the 'subscribers' table exactly (branch_id added)
type Subscriber struct {
	SubscriberID    int        `json:"subscriber_id"`
	CompanyID       int        `json:"company_id"`
	BranchID        *int       `json:"branch_id"`
	SubscriberCode  string     `json:"subscriber_code"`
	FullName        string     `json:"full_name"`
	FatherName      *string    `json:"father_name"`
	DOB             *time.Time `json:"dob"`
	Gender          *string    `json:"gender"`
	AadhaarNumber   *string    `json:"aadhaar_number"`
	PANNumber       *string    `json:"pan_number"`
	Address         string     `json:"address"`
	City            *string    `json:"city"`
	StateID         *int       `json:"state_id"`
	Pincode         *string    `json:"pincode"`
	Mobile          string     `json:"mobile"`
	AlternateMobile *string    `json:"alternate_mobile"`
	Email           *string    `json:"email"`
	Occupation      *string    `json:"occupation"`
	AnnualIncome    *float64   `json:"annual_income"`
	KYCStatus       string     `json:"kyc_status"`
	KYCVerifiedDate *time.Time `json:"kyc_verified_date"`
	KYCVerifiedBy   *int       `json:"kyc_verified_by"`
	PhotoPath       *string    `json:"photo_path"`
	SignaturePath   *string    `json:"signature_path"`
	IsBlacklisted   bool       `json:"is_blacklisted"`
	BlacklistReason *string    `json:"blacklist_reason"`
	CreditScore     *int       `json:"credit_score"`
	CreatedBy       *int       `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// SubscriberBank matches 'subscriber_banks' table
type SubscriberBank struct {
	BankID            int        `json:"bank_id"`
	SubscriberID      int        `json:"subscriber_id"`
	BankName          string     `json:"bank_name"`
	AccountNumber     string     `json:"account_number"`
	IFSCCode          string     `json:"ifsc_code"`
	AccountHolderName string     `json:"account_holder_name"`
	AccountType       *string    `json:"account_type"`
	IsPrimary         bool       `json:"is_primary"`
	IsVerified        bool       `json:"is_verified"`
	VerifiedDate      *time.Time `json:"verified_date"`
	CreatedAt         time.Time  `json:"created_at"`
}

// SubscriberListItem is the list-view response struct.
// Field names match exactly what the Flutter SubscriberListItemModel.fromJson() expects.
// All fields are value types — JSON never returns null.
type SubscriberListItem struct {
	// Core identity — Flutter uses string "id", not int "subscriber_id"
	ID             string `json:"id"`
	SubscriberCode string `json:"subscriber_code"`
	CompanyID      string `json:"company_id"`
	BranchID       string `json:"branch_id"`

	// Personal
	FullName        string  `json:"full_name"`
	DateOfBirth     string  `json:"date_of_birth"`
	Gender          string  `json:"gender"`
	Mobile          string  `json:"mobile"`
	AlternateMobile string  `json:"alternate_mobile"`
	Email           string  `json:"email"`
	Address         string  `json:"address"`
	City            string  `json:"city"`
	State           string  `json:"state"`
	Pincode         string  `json:"pincode"`
	Occupation      string  `json:"occupation"`
	AnnualIncome    float64 `json:"annual_income"`

	// KYC / Identity
	AadhaarNumber string `json:"aadhaar_number"`
	PANNumber     string `json:"pan_number"`
	CreditScore   int    `json:"credit_score"`

	// Status — Flutter _parseStatus expects "active"/"inactive"/"blacklisted"
	Status          string `json:"status"`
	IsBlacklisted   bool   `json:"is_blacklisted"`
	BlacklistReason string `json:"blacklist_reason"`

	// Computed KYC flags — Flutter SubscriberListItemModel.fromJson uses these
	IsKycComplete         bool `json:"is_kyc_complete"`
	KycDocumentCount      int  `json:"kyc_document_count"`
	VerifiedDocumentCount int  `json:"verified_document_count"`

	// Computed chit count
	ActiveChitCount int `json:"active_chit_count"`

	// Timestamps as ISO strings
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PaginatedSubscribers wraps the list with pagination — matches Flutter PaginatedSubscribersModel.fromJson
type PaginatedSubscribers struct {
	Items      []*SubscriberListItem `json:"items"`
	TotalCount int64                 `json:"total_count"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
}
