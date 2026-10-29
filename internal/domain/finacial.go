package domain

import "time"

// ── Company Expense ───────────────────────────────────────────────────────────

type CompanyExpense struct {
	ExpenseID        int64      `json:"expense_id"         db:"expense_id"`
	CompanyID        int        `json:"company_id"         db:"company_id"`
	BranchID         *int       `json:"branch_id"          db:"branch_id"`
	ExpenseDate      time.Time  `json:"expense_date"       db:"expense_date"`
	Category         string     `json:"category"           db:"category"`
	SubCategory      *string    `json:"sub_category"       db:"sub_category"`
	Description      *string    `json:"description"        db:"description"`
	Amount           float64    `json:"amount"             db:"amount"`
	GSTAmount        float64    `json:"gst_amount"         db:"gst_amount"`
	TDSAmount        float64    `json:"tds_amount"         db:"tds_amount"`
	TotalAmount      float64    `json:"total_amount"       db:"total_amount"` // GENERATED: amount + gst_amount
	PaymentMode      *string    `json:"payment_mode"       db:"payment_mode"`
	PaymentReference *string    `json:"payment_reference"  db:"payment_reference"`
	PaymentStatus    string     `json:"payment_status"     db:"payment_status"`
	PaidAmount       float64    `json:"paid_amount"        db:"paid_amount"`
	DueDate          *time.Time `json:"due_date"           db:"due_date"`
	VendorName       *string    `json:"vendor_name"        db:"vendor_name"`
	VendorGSTIN      *string    `json:"vendor_gstin"       db:"vendor_gstin"`
	InvoiceNumber    *string    `json:"invoice_number"     db:"invoice_number"`
	InvoiceDate      *time.Time `json:"invoice_date"       db:"invoice_date"`
	RecordedBy       *int       `json:"recorded_by"        db:"recorded_by"`
	ApprovedBy       *int       `json:"approved_by"        db:"approved_by"`
	IsRecurring      bool       `json:"is_recurring"       db:"is_recurring"`
	RecurringDay     *int       `json:"recurring_day"      db:"recurring_day"`
	Notes            *string    `json:"notes"              db:"notes"`
	CreatedAt        time.Time  `json:"created_at"         db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"         db:"updated_at"`
}

// Expense category constants
const (
	ExpCatSalary             = "SALARY"
	ExpCatRent               = "RENT"
	ExpCatBankCharges        = "BANK_CHARGES"
	ExpCatLegalFees          = "LEGAL_FEES"
	ExpCatPrintingStationery = "PRINTING_STATIONERY"
	ExpCatAdvertisement      = "ADVERTISEMENT"
	ExpCatDepreciation       = "DEPRECIATION"
	ExpCatGSTPayable         = "GST_PAYABLE"
	ExpCatTDSPayable         = "TDS_PAYABLE"
	ExpCatProcessingFees     = "PROCESSING_FEES"
	ExpCatProfessionalFees   = "PROFESSIONAL_FEES"
	ExpCatRepairsMaintenance = "REPAIRS_MAINTENANCE"
	ExpCatTravel             = "TRAVEL"
	ExpCatMiscellaneous      = "MISCELLANEOUS"
	ExpCatIncome             = "INCOME"
)

// ── Company Asset ─────────────────────────────────────────────────────────────

type CompanyAsset struct {
	AssetID                 int64      `json:"asset_id"                   db:"asset_id"`
	CompanyID               int        `json:"company_id"                 db:"company_id"`
	BranchID                *int       `json:"branch_id"                  db:"branch_id"`
	AssetName               string     `json:"asset_name"                 db:"asset_name"`
	AssetCode               *string    `json:"asset_code"                 db:"asset_code"`
	Category                string     `json:"category"                   db:"category"`
	PurchaseDate            time.Time  `json:"purchase_date"              db:"purchase_date"`
	PurchaseCost            float64    `json:"purchase_cost"              db:"purchase_cost"`
	SalvageValue            float64    `json:"salvage_value"              db:"salvage_value"`
	UsefulLifeYears         int        `json:"useful_life_years"          db:"useful_life_years"`
	DepreciationMethod      string     `json:"depreciation_method"        db:"depreciation_method"`
	DepreciationRate        float64    `json:"depreciation_rate"          db:"depreciation_rate"`
	AccumulatedDepreciation float64    `json:"accumulated_depreciation"   db:"accumulated_depreciation"`
	BookValue               float64    `json:"book_value"                 db:"book_value"` // GENERATED
	Status                  string     `json:"status"                     db:"status"`
	DisposalDate            *time.Time `json:"disposal_date"              db:"disposal_date"`
	DisposalValue           *float64   `json:"disposal_value"             db:"disposal_value"`
	VendorName              *string    `json:"vendor_name"                db:"vendor_name"`
	InvoiceNumber           *string    `json:"invoice_number"             db:"invoice_number"`
	Notes                   *string    `json:"notes"                      db:"notes"`
	RecordedBy              *int       `json:"recorded_by"                db:"recorded_by"`
	CreatedAt               time.Time  `json:"created_at"                 db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"                 db:"updated_at"`
}

// ── Company Capital ───────────────────────────────────────────────────────────

type CompanyCapital struct {
	CapitalID       int64     `json:"capital_id"       db:"capital_id"`
	CompanyID       int       `json:"company_id"       db:"company_id"`
	TransactionDate time.Time `json:"transaction_date" db:"transaction_date"`
	Type            string    `json:"type"             db:"type"`
	Amount          float64   `json:"amount"           db:"amount"`
	Description     *string   `json:"description"      db:"description"`
	RecordedBy      *int      `json:"recorded_by"      db:"recorded_by"`
	CreatedAt       time.Time `json:"created_at"       db:"created_at"`
}

// ── Financial Period ──────────────────────────────────────────────────────────

type FinancialPeriod struct {
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
	Label    string    `json:"label"` // e.g. "FY 2025-26" or "Apr 2026"
}

// ── P&L Statement ─────────────────────────────────────────────────────────────

type ProfitLossStatement struct {
	Period   FinancialPeriod  `json:"period"`
	Income   PLIncomeSection  `json:"income"`
	Expenses PLExpenseSection `json:"expenses"`
	Summary  PLSummary        `json:"summary"`
}

type PLIncomeSection struct {
	// ── Revenue (actual company income) ──────────────────────────────────────
	ForemanCommission   float64 `json:"foreman_commission"` // income FROM discount pool
	GSTOnCommission     float64 `json:"gst_on_commission"`  // GST collected on commission
	PenaltyIncome       float64 `json:"penalty_income"`     // late payment penalties
	ProcessingFeeIncome float64 `json:"processing_fee_income"`
	MiscellaneousIncome float64 `json:"miscellaneous_income"`
	TotalIncome         float64 `json:"total_income"`

	// ── Discount pool reconciliation (informational — NOT income) ────────────
	// Shows the CA how each auction's discount pool was split.
	// DividendDistributed is a liability reduction, NOT company income.
	TotalDiscountPool   float64 `json:"total_discount_pool"`  // sum of all winning bid discounts
	CommissionFromPool  float64 `json:"commission_from_pool"` // = ForemanCommission (deducted first)
	DividendDistributed float64 `json:"dividend_distributed"` // pool - commission → paid to subscribers
	NoteDiscountSplit   string  `json:"note_discount_split"`
}

type PLExpenseSection struct {
	Salary             float64 `json:"salary"`
	Rent               float64 `json:"rent"`
	BankCharges        float64 `json:"bank_charges"`
	LegalFees          float64 `json:"legal_fees"`
	PrintingStationery float64 `json:"printing_stationery"`
	Advertisement      float64 `json:"advertisement"`
	Depreciation       float64 `json:"depreciation"`
	ProcessingFees     float64 `json:"processing_fees"`
	ProfessionalFees   float64 `json:"professional_fees"`
	RepairsMaintenance float64 `json:"repairs_maintenance"`
	Travel             float64 `json:"travel"`
	Miscellaneous      float64 `json:"miscellaneous"`
	TotalExpenses      float64 `json:"total_expenses"`
}

type PLSummary struct {
	GrossProfit        float64 `json:"gross_profit"`
	NetProfitBeforeTax float64 `json:"net_profit_before_tax"`
	TaxProvision       float64 `json:"tax_provision"`
	NetProfitAfterTax  float64 `json:"net_profit_after_tax"`
}

// ── Balance Sheet ─────────────────────────────────────────────────────────────

type BalanceSheet struct {
	Period      FinancialPeriod         `json:"period"`
	Assets      BalanceSheetAssets      `json:"assets"`
	Liabilities BalanceSheetLiabilities `json:"liabilities"`
	Equity      BalanceSheetEquity      `json:"equity"`
	IsBalanced  bool                    `json:"is_balanced"` // assets == liabilities + equity
}

type BalanceSheetAssets struct {
	// Current Assets
	CashAndBank            float64 `json:"cash_and_bank"`
	InstallmentReceivables float64 `json:"installment_receivables"` // Debtors (prized subscribers)
	PenaltyReceivable      float64 `json:"penalty_receivable"`
	TDSReceivable          float64 `json:"tds_receivable"`
	PrepaidExpenses        float64 `json:"prepaid_expenses"`
	OtherCurrentAssets     float64 `json:"other_current_assets"`
	TotalCurrentAssets     float64 `json:"total_current_assets"`

	// Fixed Assets
	GrossFixedAssets        float64 `json:"gross_fixed_assets"`
	AccumulatedDepreciation float64 `json:"accumulated_depreciation"`
	NetFixedAssets          float64 `json:"net_fixed_assets"`

	// Other Non-Current Assets
	SecurityDeposits      float64 `json:"security_deposits"`
	TotalNonCurrentAssets float64 `json:"total_non_current_assets"`

	TotalAssets float64 `json:"total_assets"`
}

type BalanceSheetLiabilities struct {
	// Current Liabilities
	ChitFundCorpus   float64 `json:"chit_fund_corpus"` // Creditors (non-prized subscribers)
	PayoutsPayable   float64 `json:"payouts_payable"`  // Approved but not yet paid
	DividendPayable  float64 `json:"dividend_payable"` // Calculated but not posted
	GSTPayable       float64 `json:"gst_payable"`      // GST on commission to be remitted
	TDSPayable       float64 `json:"tds_payable"`      // TDS to be remitted
	SalaryPayable    float64 `json:"salary_payable"`   // Accrued unpaid salary
	OtherPayables    float64 `json:"other_payables"`   // Other pending expenses
	TotalCurrentLiab float64 `json:"total_current_liabilities"`

	// Non-Current Liabilities (loans etc)
	TotalNonCurrentLiab float64 `json:"total_non_current_liabilities"`

	TotalLiabilities float64 `json:"total_liabilities"`
}

type BalanceSheetEquity struct {
	PaidUpCapital     float64 `json:"paid_up_capital"`
	RetainedEarnings  float64 `json:"retained_earnings"`   // Prior years cumulative profit
	CurrentYearProfit float64 `json:"current_year_profit"` // From P&L
	TotalEquity       float64 `json:"total_equity"`
}

// ── Cash Flow Statement ───────────────────────────────────────────────────────

type CashFlowStatement struct {
	Period         FinancialPeriod   `json:"period"`
	Operating      CashFlowOperating `json:"operating_activities"`
	Investing      CashFlowInvesting `json:"investing_activities"`
	Financing      CashFlowFinancing `json:"financing_activities"`
	NetCashFlow    float64           `json:"net_cash_flow"`
	OpeningBalance float64           `json:"opening_cash_balance"`
	ClosingBalance float64           `json:"closing_cash_balance"`
}

type CashFlowOperating struct {
	CollectionsFromSubscribers float64 `json:"collections_from_subscribers"`
	PayoutsToWinners           float64 `json:"payouts_to_winners"`
	CommissionReceived         float64 `json:"commission_received"`
	PenaltiesCollected         float64 `json:"penalties_collected"`
	ExpensesPaid               float64 `json:"expenses_paid"`
	GSTRemitted                float64 `json:"gst_remitted"`
	TDSRemitted                float64 `json:"tds_remitted"`
	NetOperatingCashFlow       float64 `json:"net_operating_cash_flow"`
}

type CashFlowInvesting struct {
	AssetPurchases       float64 `json:"asset_purchases"`
	AssetDisposals       float64 `json:"asset_disposals"`
	NetInvestingCashFlow float64 `json:"net_investing_cash_flow"`
}

type CashFlowFinancing struct {
	CapitalIntroduced    float64 `json:"capital_introduced"`
	CapitalWithdrawn     float64 `json:"capital_withdrawn"`
	NetFinancingCashFlow float64 `json:"net_financing_cash_flow"`
}

// ── Creditor / Debtor Schedule ────────────────────────────────────────────────

type CreditorDebtorSchedule struct {
	Period FinancialPeriod `json:"period"`
	// Creditors = non-prized subscribers (company owes them future prize)
	Creditors      []SubscriberLedgerLine `json:"creditors"`
	TotalCreditors float64                `json:"total_creditors"`
	// Debtors = prized subscribers (they owe company remaining installments)
	Debtors      []SubscriberLedgerLine `json:"debtors"`
	TotalDebtors float64                `json:"total_debtors"`
}

type SubscriberLedgerLine struct {
	SubscriberID   int     `json:"subscriber_id"`
	SubscriberCode string  `json:"subscriber_code"`
	FullName       string  `json:"full_name"`
	Mobile         string  `json:"mobile"`
	ChitCode       string  `json:"chit_code"`
	ChitName       string  `json:"chit_name"`
	PrizedMonth    *int    `json:"prized_month,omitempty"` // Debtors only
	PrizedDate     *string `json:"prized_date,omitempty"`  // Debtors only
	TotalPaid      float64 `json:"total_paid"`
	TotalDue       float64 `json:"total_due"`
	Balance        float64 `json:"balance"`
}

// ── GST Summary ───────────────────────────────────────────────────────────────

type GSTSummary struct {
	Period FinancialPeriod `json:"period"`
	// Output Tax (GST collected on commission — liability)
	OutputGST GSTBreakdown `json:"output_gst"`
	// Input Tax Credit (GST paid on expenses — asset)
	InputGST      GSTBreakdown `json:"input_gst"`
	NetGSTPayable float64      `json:"net_gst_payable"`
	// Monthly breakdown for GSTR-1 filing
	MonthlyBreakdown []GSTMonthlyLine `json:"monthly_breakdown"`
}

type GSTBreakdown struct {
	Taxable float64 `json:"taxable_amount"`
	CGST    float64 `json:"cgst"` // 9%
	SGST    float64 `json:"sgst"` // 9%
	IGST    float64 `json:"igst"` // 18% (inter-state)
	Total   float64 `json:"total_gst"`
}

type GSTMonthlyLine struct {
	Month      string  `json:"month"`
	Commission float64 `json:"commission"`
	GSTAmount  float64 `json:"gst_amount"`
	Remitted   bool    `json:"remitted"`
}

// ── Expense Request types ─────────────────────────────────────────────────────

type CreateExpenseRequest struct {
	BranchID         *int    `json:"branch_id"`
	ExpenseDate      string  `json:"expense_date"       binding:"required"`
	Category         string  `json:"category"           binding:"required"`
	SubCategory      string  `json:"sub_category"`
	Description      string  `json:"description"`
	Amount           float64 `json:"amount"             binding:"required,gt=0"`
	GSTAmount        float64 `json:"gst_amount"`
	TDSAmount        float64 `json:"tds_amount"`
	PaymentMode      string  `json:"payment_mode"`
	PaymentReference string  `json:"payment_reference"`
	PaymentStatus    string  `json:"payment_status"`
	PaidAmount       float64 `json:"paid_amount"`
	DueDate          string  `json:"due_date"`
	VendorName       string  `json:"vendor_name"`
	VendorGSTIN      string  `json:"vendor_gstin"`
	InvoiceNumber    string  `json:"invoice_number"`
	IsRecurring      bool    `json:"is_recurring"`
	RecurringDay     *int    `json:"recurring_day"`
	Notes            string  `json:"notes"`
}

type CreateAssetRequest struct {
	BranchID           *int    `json:"branch_id"`
	AssetName          string  `json:"asset_name"           binding:"required"`
	AssetCode          string  `json:"asset_code"`
	Category           string  `json:"category"             binding:"required"`
	PurchaseDate       string  `json:"purchase_date"        binding:"required"`
	PurchaseCost       float64 `json:"purchase_cost"        binding:"required,gt=0"`
	SalvageValue       float64 `json:"salvage_value"`
	UsefulLifeYears    int     `json:"useful_life_years"`
	DepreciationMethod string  `json:"depreciation_method"`
	DepreciationRate   float64 `json:"depreciation_rate"`
	VendorName         string  `json:"vendor_name"`
	InvoiceNumber      string  `json:"invoice_number"`
	Notes              string  `json:"notes"`
}

type CreateCapitalRequest struct {
	TransactionDate string  `json:"transaction_date" binding:"required"`
	Type            string  `json:"type"             binding:"required,oneof=INTRODUCTION WITHDRAWAL PROFIT_RETAINED LOSS_ADJUSTED"`
	Amount          float64 `json:"amount"           binding:"required,gt=0"`
	Description     string  `json:"description"`
}

type FinancialStatementRequest struct {
	FromDate string `form:"from_date"` // YYYY-MM-DD
	ToDate   string `form:"to_date"`   // YYYY-MM-DD
	Period   string `form:"period"`    // "fy" | "month" | "quarter" | "custom"
	FYYear   int    `form:"fy_year"`   // e.g. 2026 means FY 2025-26
	Month    int    `form:"month"`     // 1-12
	Year     int    `form:"year"`      // calendar year
	Quarter  int    `form:"quarter"`   // 1-4
}
