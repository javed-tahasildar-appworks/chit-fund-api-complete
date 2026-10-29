package domain

import "time"

// ============================================================
// AUTHENTICATION REQUESTS & RESPONSES
// ============================================================

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin@chitfund.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
	ExpiresIn    int    `json:"expires_in"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ============================================================
// COMPANY REQUESTS
// ============================================================

type CreateCompanyRequest struct {
	CompanyName              string  `json:"company_name" binding:"required,min=3,max=200"`
	RegistrationNumber       string  `json:"registration_number" binding:"required"`
	PANNumber                string  `json:"pan_number,omitempty"`
	GSTIN                    string  `json:"gstin,omitempty"`
	StateID                  int     `json:"state_id" binding:"required"`
	RegisteredAddress        string  `json:"registered_address" binding:"required"`
	ContactEmail             string  `json:"contact_email" binding:"required,email"`
	ContactPhone             string  `json:"contact_phone" binding:"required"`
	ForemanCommissionPercent float64 `json:"foreman_commission_percent" binding:"max=5"`
}

type UpdateCompanyRequest struct {
	CompanyName              *string  `json:"company_name,omitempty"`
	ContactEmail             *string  `json:"contact_email,omitempty"`
	ContactPhone             *string  `json:"contact_phone,omitempty"`
	OfficeAddress            *string  `json:"office_address,omitempty"`
	City                     *string  `json:"city,omitempty"`
	Pincode                  *string  `json:"pincode,omitempty"`
	ForemanCommissionPercent *float64 `json:"foreman_commission_percent,omitempty"`
	Status                   *string  `json:"status,omitempty"`
}

// ============================================================
// BRANCH REQUESTS
// ============================================================

// CreateBranchRequest — fields match actual branches table columns
// DB columns: branch_code, branch_name, address, city, pincode, contact_phone
type CreateBranchRequest struct {
	BranchCode string  `json:"branch_code"  binding:"required"`
	BranchName string  `json:"branch_name"  binding:"required,min=3"`
	Address    *string `json:"address,omitempty"`
	City       *string `json:"city,omitempty"`
	Pincode    *string `json:"pincode,omitempty"`
	Phone      *string `json:"phone,omitempty"` // maps to contact_phone column
}

// UpdateBranchRequest — only fields that exist in actual branches table
type UpdateBranchRequest struct {
	BranchName    *string `json:"branch_name,omitempty"`
	Address       *string `json:"address,omitempty"`
	City          *string `json:"city,omitempty"`
	Phone         *string `json:"phone,omitempty"` // maps to contact_phone column
	ManagerUserID *int    `json:"manager_user_id,omitempty"`
}

// ============================================================
// USER REQUESTS
// ============================================================

type CreateUserRequest struct {
	Username     string  `json:"username" binding:"required,min=3,max=50"`
	Email        string  `json:"email" binding:"required,email"`
	Mobile       string  `json:"mobile" binding:"required"`
	FullName     string  `json:"full_name" binding:"required,min=3"`
	Password     string  `json:"password" binding:"required,min=8"`
	RoleID       int     `json:"role_id" binding:"required"`
	BranchID     *int    `json:"branch_id,omitempty"`
	EmployeeCode *string `json:"employee_code,omitempty"`
}

type UpdateUserRequest struct {
	FullName     *string `json:"full_name,omitempty"`
	Email        *string `json:"email,omitempty"`
	Mobile       *string `json:"mobile,omitempty"`
	RoleID       *int    `json:"role_id,omitempty"`
	BranchID     *int    `json:"branch_id,omitempty"`
	EmployeeCode *string `json:"employee_code,omitempty"`
}

// ============================================================
// SUBSCRIBER REQUESTS
// ============================================================

// CreateSubscriberRequest covers every insertable column in the subscribers table.
// Required by DB: full_name, address, mobile.
// gender must be MALE | FEMALE | OTHER (DB CHECK constraint).
// kyc_status must be PENDING | VERIFIED | REJECTED (default: PENDING).
// credit_score must be 300–900 if provided (DB CHECK constraint).
// state_id references the states table (1=Karnataka, 2=Tamil Nadu, 3=Kerala, 4=Andhra Pradesh, 5=Telangana).
// branch_id references the branches table.
type CreateSubscriberRequest struct {
	// Required
	FullName string `json:"full_name"      binding:"required,min=3,max=200"`
	Address  string `json:"address"        binding:"required"`
	Mobile   string `json:"mobile"         binding:"required,max=15"`

	// Organisation
	BranchID *int `json:"branch_id"` // FK → branches.branch_id

	// Personal details
	FatherName      string `json:"father_name"`
	DOB             string `json:"dob"`    // YYYY-MM-DD
	Gender          string `json:"gender"` // MALE | FEMALE | OTHER
	AlternateMobile string `json:"alternate_mobile"`
	Email           string `json:"email"`
	City            string `json:"city"`
	StateID         *int   `json:"state_id"` // FK → states.state_id
	Pincode         string `json:"pincode"`

	// Identity / KYC
	AadhaarNumber string `json:"aadhaar_number"` // exactly 12 digits
	PANNumber     string `json:"pan_number"`     // exactly 10 chars
	PhotoPath     string `json:"photo_path"`     // file path / URL
	SignaturePath string `json:"signature_path"` // file path / URL
	KYCStatus     string `json:"kyc_status"`     // PENDING | VERIFIED | REJECTED (default PENDING)

	// Financial
	Occupation   string  `json:"occupation"`
	AnnualIncome float64 `json:"annual_income"`
	CreditScore  *int    `json:"credit_score"` // 300–900
}

type UpdateSubscriberRequest struct {
	BranchID        *int     `json:"branch_id,omitempty"`
	FullName        *string  `json:"full_name,omitempty"`
	Mobile          *string  `json:"mobile,omitempty"`
	AlternateMobile *string  `json:"alternate_mobile,omitempty"`
	Email           *string  `json:"email,omitempty"`
	Address         *string  `json:"address,omitempty"`
	City            *string  `json:"city,omitempty"`
	StateID         *int     `json:"state_id,omitempty"`
	Pincode         *string  `json:"pincode,omitempty"`
	Occupation      *string  `json:"occupation,omitempty"`
	AnnualIncome    *float64 `json:"annual_income,omitempty"`
}

type BlacklistRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

// UploadDocumentRequest — fields map to subscriber_documents table columns exactly.
// document_type must be one of the DB CHECK constraint values.
// document_path is the file path or URL of the uploaded document.
// expiry_date is optional — format YYYY-MM-DD.
type UploadDocumentRequest struct {
	DocumentType   string `json:"document_type"   binding:"required,oneof=AADHAAR PAN PHOTO SIGNATURE ADDRESS_PROOF INCOME_PROOF"`
	DocumentPath   string `json:"document_path"   binding:"required"`
	DocumentNumber string `json:"document_number"`
	ExpiryDate     string `json:"expiry_date"` // YYYY-MM-DD, optional
}

type VerifyDocumentRequest struct {
	VerificationStatus string  `json:"verification_status" binding:"required,oneof=VERIFIED REJECTED"`
	RejectionReason    *string `json:"rejection_reason,omitempty"`
}

type AddBankAccountRequest struct {
	BankName          string  `json:"bank_name" binding:"required"`
	AccountNumber     string  `json:"account_number" binding:"required"`
	IFSCCode          string  `json:"ifsc_code" binding:"required"`
	AccountHolderName string  `json:"account_holder_name" binding:"required"`
	AccountType       *string `json:"account_type,omitempty"`
	IsPrimary         bool    `json:"is_primary"`
}

// ============================================================
// CHIT GROUP REQUESTS
// ============================================================

type CreateChitGroupRequest struct {
	ChitName                 string  `json:"chit_name" binding:"required,min=5,max=200"`
	ChitValue                float64 `json:"chit_value" binding:"required,gt=0"`
	TotalSubscribers         int     `json:"total_subscribers" binding:"required,gt=0"`
	TotalMonths              int     `json:"total_months" binding:"required,gt=0"`
	StartDate                string  `json:"start_date" binding:"required"`
	AuctionDayOfMonth        int     `json:"auction_day_of_month" binding:"required,min=1,max=31"`
	AuctionTime              string  `json:"auction_time" binding:"required"`
	AuctionType              string  `json:"auction_type" binding:"required,oneof=AUCTION DRAW FIXED"`
	ForemanCommissionPercent float64 `json:"foreman_commission_percent" binding:"max=5"`
	MaxDiscountPercent       float64 `json:"max_discount_percent" binding:"max=40"`
}

type UpdateChitGroupRequest struct {
	ChitName     *string `json:"chit_name,omitempty"`
	Status       *string `json:"status,omitempty"`
	CurrentMonth *int    `json:"current_month,omitempty"`
}

type AddSubscriberToChitRequest struct {
	SubscriberID   int  `json:"subscriber_id" binding:"required"`
	TicketCount    int  `json:"ticket_count" binding:"required,min=1"`
	SuretyRequired bool `json:"surety_required"`
}

// ============================================================
// AUCTION REQUESTS
// ============================================================

type CreateAuctionRequest struct {
	ChitID      int    `json:"chit_id" binding:"required"`
	AuctionDate string `json:"auction_date" binding:"required"`
	AuctionTime string `json:"auction_time,omitempty"`
	Venue       string `json:"venue,omitempty"`
}

// UpdateAuctionRequest — all fields optional, only provided fields are updated.
// auction_date format: YYYY-MM-DD
// auction_time format: HH:MM:SS
// Only SCHEDULED or IN_PROGRESS auctions can be updated.
type UpdateAuctionRequest struct {
	AuctionDate *string `json:"auction_date,omitempty"`
	AuctionTime *string `json:"auction_time,omitempty"`
	Venue       *string `json:"venue,omitempty"`
}

type PlaceBidRequest struct {
	ChitSubscriberID   int     `json:"chit_subscriber_id" binding:"required"`
	BidDiscountAmount  float64 `json:"bid_discount_amount" binding:"required,gt=0"`
	BidDiscountPercent float64 `json:"bid_discount_percent" binding:"required,gt=0,lte=40"`
}

type FinalizeAuctionRequest struct {
	WinningBidID int    `json:"winning_bid_id" binding:"required"`
	Remarks      string `json:"remarks,omitempty"`
}

type PostResultRequest struct {
	WinningBidID   int     `json:"winning_bid_id" binding:"required"`
	DiscountAmount float64 `json:"discount_amount" binding:"required,gt=0"`
	Remarks        string  `json:"remarks,omitempty"`
}

type MarkBidWinnerRequest struct {
	Remarks string `json:"remarks,omitempty"`
}

// ============================================================
// INSTALLMENT REQUESTS
// ============================================================

type GenerateInstallmentRequest struct {
	ChitID                int     `json:"chit_id" binding:"required"`
	MonthNumber           int     `json:"month_number" binding:"required,min=1"`
	BaseInstallmentAmount float64 `json:"base_installment_amount" binding:"required,gt=0"`
	DividendAmount        float64 `json:"dividend_amount"`
	DueDate               string  `json:"due_date" binding:"required"`
}

type CreateInstallmentRequest struct {
	ChitID                int     `json:"chit_id" binding:"required"`
	MonthNumber           int     `json:"month_number" binding:"required"`
	BaseInstallmentAmount float64 `json:"base_installment_amount" binding:"required,gt=0"`
	DividendAmount        float64 `json:"dividend_amount"`
	DueDate               string  `json:"due_date" binding:"required"`
}

type RecordPaymentRequest struct {
	Amount               float64 `json:"amount" binding:"required,gt=0"`
	PaymentMode          string  `json:"payment_mode" binding:"required,oneof=CASH CHEQUE NEFT RTGS UPI CARD ONLINE"`
	TransactionReference string  `json:"transaction_reference,omitempty"`
	Remarks              string  `json:"remarks,omitempty"`
}

// ============================================================
// PAYOUT REQUESTS
// ============================================================

type InitiatePayoutRequest struct {
	AuctionID int    `json:"auction_id" binding:"required"`
	BankID    *int   `json:"bank_id,omitempty"`
	Remarks   string `json:"remarks,omitempty"`
}

type ApprovePayoutRequest struct {
	Remarks string `json:"remarks,omitempty"`
}

type DisbursePayoutRequest struct {
	PayoutMode           string  `json:"payout_mode" binding:"required,oneof=NEFT RTGS CASH CHEQUE"`
	TransactionReference *string `json:"transaction_reference,omitempty"`
	PayoutDate           string  `json:"payout_date" binding:"required"`
	Remarks              string  `json:"remarks,omitempty"`
}

// ============================================================
// DIVIDEND REQUESTS
// ============================================================

type CalculateDividendRequest struct {
	AuctionID int `json:"auction_id" binding:"required"`
}

type PostDividendRequest struct {
	DividendID int `json:"dividend_id" binding:"required"`
}

// ============================================================
// DEFAULTS & LEGAL REQUESTS
// ============================================================

type FlagDefaultRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

type ResolveDefaultRequest struct {
	ResolutionNotes string `json:"resolution_notes" binding:"required,min=5"`
}

type SendReminderRequest struct {
	ReminderType string `json:"reminder_type" binding:"required,oneof=SMS EMAIL BOTH"`
	Message      string `json:"message,omitempty"`
}

type IssueLegalNoticeRequest struct {
	DefaultID        int     `json:"default_id" binding:"required"`
	NoticeType       string  `json:"notice_type" binding:"required,oneof=REMINDER LEGAL_NOTICE COURT_SUMMON"`
	NoticeContent    string  `json:"notice_content,omitempty"`
	NoticeSentVia    string  `json:"notice_sent_via" binding:"required,oneof=EMAIL SMS POST HAND_DELIVERY"`
	ResponseDeadline *string `json:"response_deadline,omitempty"`
}

type EscalateLegalNoticeRequest struct {
	EscalateTo string `json:"escalate_to" binding:"required"`
	Remarks    string `json:"remarks,omitempty"`
}

// ============================================================
// ROLE REQUESTS
// ============================================================

type UpdateRolePermissionsRequest struct {
	Permissions interface{} `json:"permissions" binding:"required"`
}

// ============================================================
// COMMON RESPONSES
// ============================================================

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// ============================================================
// LIST FILTERS
// ============================================================

type ListFilters struct {
	Status   string `form:"status"`
	BranchID int    `form:"branch_id"`
	Search   string `form:"search"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type ChitGroupFilters struct {
	Status      string `form:"status"`
	BranchID    int    `form:"branch_id"`
	AuctionType string `form:"auction_type"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type AuctionFilters struct {
	ChitID      int    `form:"chit_id"`
	Status      string `form:"status"`
	MonthNumber int    `form:"month_number"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type InstallmentFilters struct {
	ChitID      int    `form:"chit_id"`
	MonthNumber int    `form:"month_number"`
	Status      string `form:"status"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type DefaultFilters struct {
	Status   string `form:"status"`
	BranchID int    `form:"branch_id"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ============================================================
// SUBSCRIBER SUMMARY
// ============================================================

type SubscriberChitSummary struct {
	ChitID        int       `json:"chit_id"`
	ChitName      string    `json:"chit_name"`
	ChitValue     float64   `json:"chit_value"`
	CurrentMonth  int       `json:"current_month"`
	TotalMonths   int       `json:"total_months"`
	IsPrized      bool      `json:"is_prized"`
	NextDueDate   time.Time `json:"next_due_date"`
	NextDueAmount float64   `json:"next_due_amount"`
	TotalPaid     float64   `json:"total_paid"`
	PendingAmount float64   `json:"pending_amount"`
	Status        string    `json:"status"`
}

type PaymentHistoryItem struct {
	PaymentID     int       `json:"payment_id"`
	ChitName      string    `json:"chit_name"`
	MonthNumber   int       `json:"month_number"`
	AmountPaid    float64   `json:"amount_paid"`
	PaymentDate   time.Time `json:"payment_date"`
	PaymentMode   string    `json:"payment_mode"`
	ReceiptNumber string    `json:"receipt_number"`
	Status        string    `json:"status"`
}
