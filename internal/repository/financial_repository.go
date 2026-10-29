package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"
)

type FinancialRepository struct {
	db *database.PostgresPool
}

func NewFinancialRepository(db *database.PostgresPool) *FinancialRepository {
	return &FinancialRepository{db: db}
}

// ── Period helpers ────────────────────────────────────────────────────────────

func (r *FinancialRepository) ResolvePeriodPublic(req domain.FinancialStatementRequest) (from, to time.Time, label string, err error) {
	return resolvePeriod(req)
}

func resolvePeriod(req domain.FinancialStatementRequest) (from, to time.Time, label string, err error) {
	now := time.Now()
	switch req.Period {
	case "fy":
		yr := req.FYYear
		if yr == 0 {
			yr = now.Year()
			if now.Month() < 4 {
				yr--
			}
		}
		from = time.Date(yr, 4, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(yr+1, 3, 31, 23, 59, 59, 0, time.UTC)
		label = fmt.Sprintf("FY %d-%d", yr, yr+1)
	case "month":
		yr, mo := req.Year, req.Month
		if yr == 0 {
			yr = now.Year()
		}
		if mo == 0 {
			mo = int(now.Month())
		}
		from = time.Date(yr, time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, -1)
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)
		label = from.Format("January 2006")
	case "quarter":
		yr, q := req.Year, req.Quarter
		if yr == 0 {
			yr = now.Year()
		}
		if q == 0 {
			q = (int(now.Month())-1)/3 + 1
		}
		startMonth := time.Month((q-1)*3 + 1)
		from = time.Date(yr, startMonth, 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 3, -1)
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)
		label = fmt.Sprintf("Q%d %d", q, yr)
	default: // custom or fy (default)
		if req.FromDate != "" && req.ToDate != "" {
			from, err = time.Parse("2006-01-02", req.FromDate)
			if err != nil {
				return
			}
			to, err = time.Parse("2006-01-02", req.ToDate)
			if err != nil {
				return
			}
			to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)
			label = fmt.Sprintf("%s to %s", from.Format("02 Jan 2006"), to.Format("02 Jan 2006"))
		} else {
			// Default: current Indian FY
			yr := now.Year()
			if now.Month() < 4 {
				yr--
			}
			from = time.Date(yr, 4, 1, 0, 0, 0, 0, time.UTC)
			to = time.Date(yr+1, 3, 31, 23, 59, 59, 0, time.UTC)
			label = fmt.Sprintf("FY %d-%d", yr, yr+1)
		}
	}
	return
}

// ── P&L ───────────────────────────────────────────────────────────────────────

func (r *FinancialRepository) GetProfitLoss(
	ctx context.Context, companyID int, req domain.FinancialStatementRequest,
) (*domain.ProfitLossStatement, error) {
	from, to, label, err := resolvePeriod(req)
	if err != nil {
		return nil, err
	}
	pl := &domain.ProfitLossStatement{
		Period: domain.FinancialPeriod{FromDate: from, ToDate: to, Label: label},
	}

	// ── Income from ledger (if available) ───────────────────────────────────
	var ledgerCommission, ledgerGST, ledgerPenalty float64
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN account = 'commission_income' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN account = 'gst_output' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN account = 'penalty_income' THEN amount ELSE 0 END), 0)
		FROM financial_ledger_entries
		WHERE company_id = $1 AND direction = 'CREDIT'
		  AND entry_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&ledgerCommission, &ledgerGST, &ledgerPenalty)

	// ── Fallback from operational tables ────────────────────────────────────
	var opCommission, opGST, opPenalty float64
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(foreman_commission), 0),
			COALESCE(SUM(gst_on_commission), 0)
		FROM payouts
		WHERE company_id = $1 AND status = 'PAID'
		  AND payout_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&opCommission, &opGST)

	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(si.penalty_amount), 0)
		FROM subscriber_installments si
		INNER JOIN chit_groups cg ON si.chit_id_ref = cg.chit_id
		WHERE cg.company_id = $1 AND si.status = 'PAID'
		  AND si.payment_date BETWEEN $2 AND $3
		  AND si.penalty_amount > 0
	`, companyID, from, to).Scan(&opPenalty)

	// Prefer ledger if it has data
	commission := ledgerCommission
	if commission == 0 {
		commission = opCommission
	}
	gstOnCommission := ledgerGST
	if gstOnCommission == 0 {
		gstOnCommission = opGST
	}
	penalty := ledgerPenalty
	if penalty == 0 {
		penalty = opPenalty
	}

	// Misc income from expenses table (category = INCOME)
	var miscIncome float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM company_expenses
		WHERE company_id = $1 AND category = 'INCOME'
		  AND payment_status = 'PAID'
		  AND expense_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&miscIncome)

	pl.Income.ForemanCommission = commission
	pl.Income.GSTOnCommission = gstOnCommission
	pl.Income.PenaltyIncome = penalty
	pl.Income.MiscellaneousIncome = miscIncome
	pl.Income.TotalIncome = commission + gstOnCommission + penalty + miscIncome

	// ── Discount pool reconciliation (informational for CA) ──────────────────
	// Read the total discount pool and distributable from payouts/dividends.
	// Distributable dividend is a LIABILITY REDUCTION, not income.
	var totalDiscountPool, totalDistributed float64
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(discount_amount), 0),
			COALESCE(SUM(distributable_dividend_amount), 0)
		FROM payouts
		WHERE company_id = $1 AND status = 'PAID'
		  AND payout_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&totalDiscountPool, &totalDistributed)

	pl.Income.TotalDiscountPool = totalDiscountPool
	pl.Income.CommissionFromPool = commission // same as ForemanCommission — deducted first
	pl.Income.DividendDistributed = totalDistributed
	pl.Income.NoteDiscountSplit = fmt.Sprintf(
		"Discount pool ₹%.2f = Commission ₹%.2f (income) + Dividend ₹%.2f (liability reduction to subscribers)",
		totalDiscountPool, commission, totalDistributed,
	)

	// ── Expenses from company_expenses ──────────────────────────────────────
	rows, err := r.db.Query(ctx, `
		SELECT category, COALESCE(SUM(amount), 0)
		FROM company_expenses
		WHERE company_id = $1 AND category != 'INCOME'
		  AND payment_status IN ('PAID', 'PARTIALLY_PAID')
		  AND expense_date BETWEEN $2 AND $3
		GROUP BY category
	`, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cat string
		var amt float64
		rows.Scan(&cat, &amt)
		switch cat {
		case domain.ExpCatSalary:
			pl.Expenses.Salary = amt
		case domain.ExpCatRent:
			pl.Expenses.Rent = amt
		case domain.ExpCatBankCharges:
			pl.Expenses.BankCharges = amt
		case domain.ExpCatLegalFees:
			pl.Expenses.LegalFees = amt
		case domain.ExpCatPrintingStationery:
			pl.Expenses.PrintingStationery = amt
		case domain.ExpCatAdvertisement:
			pl.Expenses.Advertisement = amt
		case domain.ExpCatDepreciation:
			pl.Expenses.Depreciation = amt
		case domain.ExpCatProcessingFees:
			pl.Expenses.ProcessingFees = amt
		case domain.ExpCatProfessionalFees:
			pl.Expenses.ProfessionalFees = amt
		case domain.ExpCatRepairsMaintenance:
			pl.Expenses.RepairsMaintenance = amt
		case domain.ExpCatTravel:
			pl.Expenses.Travel = amt
		case domain.ExpCatMiscellaneous:
			pl.Expenses.Miscellaneous = amt
		}
	}
	pl.Expenses.TotalExpenses = pl.Expenses.Salary + pl.Expenses.Rent +
		pl.Expenses.BankCharges + pl.Expenses.LegalFees +
		pl.Expenses.PrintingStationery + pl.Expenses.Advertisement +
		pl.Expenses.Depreciation + pl.Expenses.ProcessingFees +
		pl.Expenses.ProfessionalFees + pl.Expenses.RepairsMaintenance +
		pl.Expenses.Travel + pl.Expenses.Miscellaneous

	pl.Summary.GrossProfit = pl.Income.ForemanCommission + pl.Income.PenaltyIncome + pl.Income.MiscellaneousIncome
	pl.Summary.NetProfitBeforeTax = pl.Summary.GrossProfit - pl.Expenses.TotalExpenses
	pl.Summary.NetProfitAfterTax = pl.Summary.NetProfitBeforeTax // tax provision TBD
	return pl, nil
}

// ── Balance Sheet ─────────────────────────────────────────────────────────────

func (r *FinancialRepository) GetBalanceSheet(
	ctx context.Context, companyID int, req domain.FinancialStatementRequest,
) (*domain.BalanceSheet, error) {
	from, to, label, err := resolvePeriod(req)
	if err != nil {
		return nil, err
	}
	bs := &domain.BalanceSheet{
		Period: domain.FinancialPeriod{FromDate: from, ToDate: to, Label: label},
	}

	// ── ASSETS ──────────────────────────────────────────────────────────────

	// Cash & Bank: total collected - total disbursed (paid payouts) - paid expenses
	var totalCollected, totalDisbursed, totalExpensesPaid float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM payment_transactions
		WHERE company_id = $1 AND status = 'SUCCESS'
		  AND transaction_date <= $2
	`, companyID, to).Scan(&totalCollected)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(net_payout), 0) FROM payouts
		WHERE company_id = $1 AND status = 'PAID' AND payout_date <= $2
	`, companyID, to).Scan(&totalDisbursed)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(paid_amount), 0) FROM company_expenses
		WHERE company_id = $1 AND payment_status IN ('PAID','PARTIALLY_PAID')
		  AND expense_date <= $2
	`, companyID, to).Scan(&totalExpensesPaid)
	bs.Assets.CashAndBank = totalCollected - totalDisbursed - totalExpensesPaid

	// Debtors = prized subscribers with outstanding installments
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(si.balance_amount), 0)
		FROM subscriber_installments si
		INNER JOIN chit_subscribers cs ON si.chit_subscriber_id = cs.chit_subscriber_id
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		WHERE cg.company_id = $1
		  AND cs.is_prized = true
		  AND si.status IN ('PENDING','PARTIAL','OVERDUE')
		  AND si.created_at <= $2
	`, companyID, to).Scan(&bs.Assets.InstallmentReceivables)

	// Penalty receivable
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(si.penalty_amount), 0)
		FROM subscriber_installments si
		INNER JOIN chit_subscribers cs ON si.chit_subscriber_id = cs.chit_subscriber_id
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		WHERE cg.company_id = $1 AND si.status != 'PAID' AND si.penalty_amount > 0
		  AND si.created_at <= $2
	`, companyID, to).Scan(&bs.Assets.PenaltyReceivable)

	// TDS receivable (TDS deducted on commission)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(tds_on_commission), 0) FROM payouts
		WHERE company_id = $1 AND status = 'PAID' AND payout_date <= $2
	`, companyID, to).Scan(&bs.Assets.TDSReceivable)

	bs.Assets.TotalCurrentAssets = bs.Assets.CashAndBank +
		bs.Assets.InstallmentReceivables + bs.Assets.PenaltyReceivable +
		bs.Assets.TDSReceivable

	// Fixed assets
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(purchase_cost), 0),
			COALESCE(SUM(accumulated_depreciation), 0)
		FROM company_assets
		WHERE company_id = $1 AND status != 'DISPOSED'
		  AND purchase_date <= $2
	`, companyID, to).Scan(&bs.Assets.GrossFixedAssets, &bs.Assets.AccumulatedDepreciation)
	bs.Assets.NetFixedAssets = bs.Assets.GrossFixedAssets - bs.Assets.AccumulatedDepreciation

	// Security deposits (asset category)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(purchase_cost), 0) FROM company_assets
		WHERE company_id = $1 AND category = 'SECURITY_DEPOSIT'
		  AND status = 'ACTIVE' AND purchase_date <= $2
	`, companyID, to).Scan(&bs.Assets.SecurityDeposits)
	bs.Assets.TotalNonCurrentAssets = bs.Assets.NetFixedAssets + bs.Assets.SecurityDeposits
	bs.Assets.TotalAssets = bs.Assets.TotalCurrentAssets + bs.Assets.TotalNonCurrentAssets

	// ── LIABILITIES ─────────────────────────────────────────────────────────

	// Chit Fund Corpus = money collected from non-prized subscribers
	// MINUS dividends already distributed (which reduce what company owes them).
	// This is the net liability to non-prized subscribers.
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(pt.amount), 0)
		FROM payment_transactions pt
		INNER JOIN chit_subscribers cs
			ON pt.subscriber_id = cs.subscriber_id
			AND pt.chit_id = cs.chit_id
		INNER JOIN chit_groups cg ON pt.chit_id = cg.chit_id
		WHERE cg.company_id = $1
		  AND cs.is_prized = false
		  AND pt.status = 'SUCCESS'
		  AND pt.transaction_date <= $2
	`, companyID, to).Scan(&bs.Liabilities.ChitFundCorpus)

	// Subtract dividends distributed to reduce corpus liability
	var dividendsDistributed float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(distributable_amount), 0)
		FROM dividends
		WHERE company_id = $1 AND status = 'POSTED'
		  AND distribution_date <= $2
	`, companyID, to).Scan(&dividendsDistributed)
	bs.Liabilities.ChitFundCorpus -= dividendsDistributed
	if bs.Liabilities.ChitFundCorpus < 0 {
		bs.Liabilities.ChitFundCorpus = 0
	}

	// Payouts payable (approved but not paid)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(net_payout), 0) FROM payouts
		WHERE company_id = $1 AND status = 'APPROVED'
		  AND created_at <= $2
	`, companyID, to).Scan(&bs.Liabilities.PayoutsPayable)

	// Dividend payable
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(distributable_amount), 0) FROM dividends
		WHERE company_id = $1 AND status = 'CALCULATED'
		  AND created_at <= $2
	`, companyID, to).Scan(&bs.Liabilities.DividendPayable)

	// GST payable (collected on commission, not yet remitted)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gst_on_commission), 0) FROM payouts
		WHERE company_id = $1 AND status = 'PAID' AND payout_date <= $2
	`, companyID, to).Scan(&bs.Liabilities.GSTPayable)
	// Subtract already remitted
	var gstRemitted float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM company_expenses
		WHERE company_id = $1 AND category = 'GST_PAYABLE'
		  AND payment_status = 'PAID' AND expense_date <= $2
	`, companyID, to).Scan(&gstRemitted)
	bs.Liabilities.GSTPayable -= gstRemitted

	// Salary payable (pending salary expenses)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount - paid_amount), 0) FROM company_expenses
		WHERE company_id = $1 AND category = 'SALARY'
		  AND payment_status IN ('PENDING','PARTIALLY_PAID')
		  AND due_date <= $2
	`, companyID, to).Scan(&bs.Liabilities.SalaryPayable)

	// Other payables
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount - paid_amount), 0) FROM company_expenses
		WHERE company_id = $1 AND category NOT IN ('SALARY','GST_PAYABLE','TDS_PAYABLE','INCOME')
		  AND payment_status IN ('PENDING','PARTIALLY_PAID')
		  AND due_date <= $2
	`, companyID, to).Scan(&bs.Liabilities.OtherPayables)

	bs.Liabilities.TotalCurrentLiab = bs.Liabilities.ChitFundCorpus +
		bs.Liabilities.PayoutsPayable + bs.Liabilities.DividendPayable +
		bs.Liabilities.GSTPayable + bs.Liabilities.SalaryPayable +
		bs.Liabilities.OtherPayables
	bs.Liabilities.TotalLiabilities = bs.Liabilities.TotalCurrentLiab

	// ── EQUITY ──────────────────────────────────────────────────────────────

	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN type='INTRODUCTION' THEN amount ELSE -amount END), 0)
		FROM company_capital WHERE company_id = $1 AND transaction_date <= $2
	`, companyID, to).Scan(&bs.Equity.PaidUpCapital)

	// Retained earnings = cumulative P&L up to start of this period
	prevReq := req
	prevReq.Period = "custom"
	prevReq.ToDate = from.AddDate(0, 0, -1).Format("2006-01-02")
	prevReq.FromDate = "2000-01-01"
	prevPL, _ := r.GetProfitLoss(ctx, companyID, prevReq)
	if prevPL != nil {
		bs.Equity.RetainedEarnings = prevPL.Summary.NetProfitAfterTax
	}

	// Current year profit from P&L
	currPL, err := r.GetProfitLoss(ctx, companyID, req)
	if err != nil {
		return nil, err
	}
	bs.Equity.CurrentYearProfit = currPL.Summary.NetProfitAfterTax
	bs.Equity.TotalEquity = bs.Equity.PaidUpCapital + bs.Equity.RetainedEarnings + bs.Equity.CurrentYearProfit

	bs.IsBalanced = abs64(bs.Assets.TotalAssets-(bs.Liabilities.TotalLiabilities+bs.Equity.TotalEquity)) < 0.01
	return bs, nil
}

// ── Cash Flow ─────────────────────────────────────────────────────────────────

func (r *FinancialRepository) GetCashFlow(
	ctx context.Context, companyID int, req domain.FinancialStatementRequest,
) (*domain.CashFlowStatement, error) {
	from, to, label, err := resolvePeriod(req)
	if err != nil {
		return nil, err
	}
	cf := &domain.CashFlowStatement{
		Period: domain.FinancialPeriod{FromDate: from, ToDate: to, Label: label},
	}

	// Operating: Collections from subscribers
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM payment_transactions
		WHERE company_id = $1 AND status = 'SUCCESS'
		  AND transaction_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Operating.CollectionsFromSubscribers)

	// Operating: Payouts to winners
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(net_payout), 0) FROM payouts
		WHERE company_id = $1 AND status = 'PAID'
		  AND payout_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Operating.PayoutsToWinners)

	// Operating: Commission received
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(foreman_commission), 0) FROM payouts
		WHERE company_id = $1 AND status = 'PAID'
		  AND payout_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Operating.CommissionReceived)

	// Operating: Penalties collected
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(si.penalty_amount), 0)
		FROM subscriber_installments si
		INNER JOIN chit_groups cg ON si.chit_id_ref = cg.chit_id
		WHERE cg.company_id = $1 AND si.status = 'PAID'
		  AND si.payment_date BETWEEN $2 AND $3
		  AND si.penalty_amount > 0
	`, companyID, from, to).Scan(&cf.Operating.PenaltiesCollected)

	// Operating: Expenses paid
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(paid_amount), 0) FROM company_expenses
		WHERE company_id = $1 AND category NOT IN ('GST_PAYABLE','TDS_PAYABLE')
		  AND payment_status IN ('PAID','PARTIALLY_PAID')
		  AND expense_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Operating.ExpensesPaid)

	// GST remitted
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(paid_amount), 0) FROM company_expenses
		WHERE company_id = $1 AND category = 'GST_PAYABLE'
		  AND payment_status = 'PAID' AND expense_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Operating.GSTRemitted)

	cf.Operating.NetOperatingCashFlow = cf.Operating.CollectionsFromSubscribers +
		cf.Operating.CommissionReceived + cf.Operating.PenaltiesCollected -
		cf.Operating.PayoutsToWinners - cf.Operating.ExpensesPaid -
		cf.Operating.GSTRemitted - cf.Operating.TDSRemitted

	// Investing: Asset purchases
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(purchase_cost), 0) FROM company_assets
		WHERE company_id = $1 AND purchase_date BETWEEN $2 AND $3
		  AND category != 'SECURITY_DEPOSIT'
	`, companyID, from, to).Scan(&cf.Investing.AssetPurchases)

	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(disposal_value), 0) FROM company_assets
		WHERE company_id = $1 AND disposal_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Investing.AssetDisposals)

	cf.Investing.NetInvestingCashFlow = cf.Investing.AssetDisposals - cf.Investing.AssetPurchases

	// Financing: Capital movements
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type='INTRODUCTION' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type='WITHDRAWAL' THEN amount ELSE 0 END), 0)
		FROM company_capital WHERE company_id = $1
		  AND transaction_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&cf.Financing.CapitalIntroduced, &cf.Financing.CapitalWithdrawn)
	cf.Financing.NetFinancingCashFlow = cf.Financing.CapitalIntroduced - cf.Financing.CapitalWithdrawn

	cf.NetCashFlow = cf.Operating.NetOperatingCashFlow +
		cf.Investing.NetInvestingCashFlow + cf.Financing.NetFinancingCashFlow

	// Opening and closing balance
	openReq := req
	openReq.Period = "custom"
	openReq.ToDate = from.AddDate(0, 0, -1).Format("2006-01-02")
	openReq.FromDate = "2000-01-01"
	openBS, _ := r.GetBalanceSheet(ctx, companyID, openReq)
	if openBS != nil {
		cf.OpeningBalance = openBS.Assets.CashAndBank
	}
	cf.ClosingBalance = cf.OpeningBalance + cf.NetCashFlow
	return cf, nil
}

// ── Creditor / Debtor Schedule ────────────────────────────────────────────────

func (r *FinancialRepository) GetCreditorDebtorSchedule(
	ctx context.Context, companyID int, req domain.FinancialStatementRequest,
) (*domain.CreditorDebtorSchedule, error) {
	from, to, label, err := resolvePeriod(req)
	if err != nil {
		return nil, err
	}
	sched := &domain.CreditorDebtorSchedule{
		Period:    domain.FinancialPeriod{FromDate: from, ToDate: to, Label: label},
		Creditors: []domain.SubscriberLedgerLine{},
		Debtors:   []domain.SubscriberLedgerLine{},
	}

	// Creditors = non-prized subscribers
	credRows, err := r.db.Query(ctx, `
		SELECT s.subscriber_id, s.subscriber_code, s.full_name, s.mobile,
			cg.chit_code, cg.chit_name,
			COALESCE(SUM(si.amount_paid), 0) AS total_paid,
			COALESCE(SUM(si.balance_amount), 0) AS total_due,
			COALESCE(SUM(si.amount_paid), 0) AS balance
		FROM chit_subscribers cs
		INNER JOIN subscribers s ON cs.subscriber_id = s.subscriber_id
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		LEFT JOIN subscriber_installments si ON cs.chit_subscriber_id = si.chit_subscriber_id
		WHERE cg.company_id = $1 AND cs.is_prized = false AND cs.status = 'ACTIVE'
		  AND cg.status = 'ACTIVE'
		GROUP BY s.subscriber_id, s.subscriber_code, s.full_name, s.mobile, cg.chit_code, cg.chit_name
		ORDER BY s.full_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer credRows.Close()
	for credRows.Next() {
		var line domain.SubscriberLedgerLine
		credRows.Scan(&line.SubscriberID, &line.SubscriberCode, &line.FullName,
			&line.Mobile, &line.ChitCode, &line.ChitName,
			&line.TotalPaid, &line.TotalDue, &line.Balance)
		sched.Creditors = append(sched.Creditors, line)
		sched.TotalCreditors += line.TotalPaid
	}

	// Debtors = prized subscribers
	debtRows, err := r.db.Query(ctx, `
		SELECT s.subscriber_id, s.subscriber_code, s.full_name, s.mobile,
			cg.chit_code, cg.chit_name,
			cs.prized_month,
			TO_CHAR(cs.prized_date, 'YYYY-MM-DD'),
			COALESCE(SUM(si.amount_paid), 0) AS total_paid,
			COALESCE(SUM(si.balance_amount), 0) AS total_due,
			COALESCE(SUM(si.balance_amount), 0) AS balance
		FROM chit_subscribers cs
		INNER JOIN subscribers s ON cs.subscriber_id = s.subscriber_id
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		LEFT JOIN subscriber_installments si ON cs.chit_subscriber_id = si.chit_subscriber_id
			AND si.status IN ('PENDING','PARTIAL','OVERDUE')
		WHERE cg.company_id = $1 AND cs.is_prized = true AND cs.status = 'ACTIVE'
		GROUP BY s.subscriber_id, s.subscriber_code, s.full_name, s.mobile,
			cg.chit_code, cg.chit_name, cs.prized_month, cs.prized_date
		ORDER BY s.full_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer debtRows.Close()
	for debtRows.Next() {
		var line domain.SubscriberLedgerLine
		debtRows.Scan(&line.SubscriberID, &line.SubscriberCode, &line.FullName,
			&line.Mobile, &line.ChitCode, &line.ChitName,
			&line.PrizedMonth, &line.PrizedDate,
			&line.TotalPaid, &line.TotalDue, &line.Balance)
		sched.Debtors = append(sched.Debtors, line)
		sched.TotalDebtors += line.Balance
	}
	_ = from
	_ = to
	return sched, nil
}

// ── GST Summary ───────────────────────────────────────────────────────────────

func (r *FinancialRepository) GetGSTSummary(
	ctx context.Context, companyID int, req domain.FinancialStatementRequest,
) (*domain.GSTSummary, error) {
	from, to, label, err := resolvePeriod(req)
	if err != nil {
		return nil, err
	}
	gst := &domain.GSTSummary{
		Period: domain.FinancialPeriod{FromDate: from, ToDate: to, Label: label},
	}

	// Output GST (collected on commission)
	var outputGSTTotal, commissionBase float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(foreman_commission), 0), COALESCE(SUM(gst_on_commission), 0)
		FROM payouts WHERE company_id = $1 AND status = 'PAID'
		  AND payout_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&commissionBase, &outputGSTTotal)

	gst.OutputGST.Taxable = commissionBase
	gst.OutputGST.CGST = outputGSTTotal / 2 // 9% CGST
	gst.OutputGST.SGST = outputGSTTotal / 2 // 9% SGST
	gst.OutputGST.Total = outputGSTTotal

	// Input GST (paid on expenses)
	var inputGSTTotal float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gst_amount), 0) FROM company_expenses
		WHERE company_id = $1 AND payment_status IN ('PAID','PARTIALLY_PAID')
		  AND expense_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&inputGSTTotal)
	gst.InputGST.Total = inputGSTTotal
	gst.InputGST.CGST = inputGSTTotal / 2
	gst.InputGST.SGST = inputGSTTotal / 2

	gst.NetGSTPayable = gst.OutputGST.Total - gst.InputGST.Total

	// Monthly breakdown
	monthRows, err := r.db.Query(ctx, `
		SELECT
			TO_CHAR(DATE_TRUNC('month', payout_date), 'Mon YYYY') AS month,
			COALESCE(SUM(foreman_commission), 0),
			COALESCE(SUM(gst_on_commission), 0)
		FROM payouts
		WHERE company_id = $1 AND status = 'PAID'
		  AND payout_date BETWEEN $2 AND $3
		GROUP BY DATE_TRUNC('month', payout_date)
		ORDER BY DATE_TRUNC('month', payout_date)
	`, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer monthRows.Close()
	for monthRows.Next() {
		var line domain.GSTMonthlyLine
		monthRows.Scan(&line.Month, &line.Commission, &line.GSTAmount)
		// Check if remitted
		var remitted float64
		r.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(paid_amount), 0) FROM company_expenses
			WHERE company_id = $1 AND category = 'GST_PAYABLE'
			  AND TO_CHAR(expense_date, 'Mon YYYY') = $2
			  AND payment_status = 'PAID'
		`, companyID, line.Month).Scan(&remitted)
		line.Remitted = remitted >= line.GSTAmount
		gst.MonthlyBreakdown = append(gst.MonthlyBreakdown, line)
	}
	return gst, nil
}

// ── Expense CRUD ──────────────────────────────────────────────────────────────

func (r *FinancialRepository) CreateExpense(ctx context.Context, companyID, userID int,
	req *domain.CreateExpenseRequest) (*domain.CompanyExpense, error) {
	expDate, _ := time.Parse("2006-01-02", req.ExpenseDate)
	payStatus := req.PaymentStatus
	if payStatus == "" {
		payStatus = "PAID"
	}
	paidAmt := req.PaidAmount
	if payStatus == "PAID" {
		paidAmt = req.Amount
	}
	var e domain.CompanyExpense
	err := r.db.QueryRow(ctx, `
		INSERT INTO company_expenses (
			company_id, branch_id, expense_date, category, sub_category,
			description, amount, gst_amount, tds_amount, payment_mode,
			payment_reference, payment_status, paid_amount,
			vendor_name, vendor_gstin, invoice_number,
			is_recurring, recurring_day, notes, recorded_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		RETURNING expense_id, total_amount, created_at, updated_at
	`, companyID, req.BranchID, expDate, req.Category,
		nullStr(req.SubCategory), nullStr(req.Description),
		req.Amount, req.GSTAmount, req.TDSAmount,
		nullStr(req.PaymentMode), nullStr(req.PaymentReference),
		payStatus, paidAmt,
		nullStr(req.VendorName), nullStr(req.VendorGSTIN), nullStr(req.InvoiceNumber),
		req.IsRecurring, req.RecurringDay, nullStr(req.Notes), userID,
	).Scan(&e.ExpenseID, &e.TotalAmount, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.CompanyID = companyID
	e.Amount = req.Amount
	e.Category = req.Category
	e.ExpenseDate = expDate
	return &e, nil
}

func (r *FinancialRepository) ListExpenses(ctx context.Context, companyID int,
	from, to time.Time, category string, limit, offset int) ([]*domain.CompanyExpense, int64, error) {
	args := []interface{}{companyID, from, to}
	where := " WHERE company_id = $1 AND expense_date BETWEEN $2 AND $3"
	if category != "" {
		where += " AND category = $4"
		args = append(args, category)
	}
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM company_expenses`+where, args...).Scan(&total)

	argPos := len(args) + 1
	query := `SELECT expense_id, company_id, branch_id, expense_date, category,
		sub_category, description, amount, gst_amount, tds_amount, total_amount,
		payment_mode, payment_reference, payment_status, paid_amount,
		vendor_name, is_recurring, notes, recorded_by, created_at, updated_at
		FROM company_expenses` + where +
		fmt.Sprintf(" ORDER BY expense_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var expenses []*domain.CompanyExpense
	for rows.Next() {
		var e domain.CompanyExpense
		rows.Scan(&e.ExpenseID, &e.CompanyID, &e.BranchID, &e.ExpenseDate, &e.Category,
			&e.SubCategory, &e.Description, &e.Amount, &e.GSTAmount, &e.TDSAmount,
			&e.TotalAmount, &e.PaymentMode, &e.PaymentReference, &e.PaymentStatus,
			&e.PaidAmount, &e.VendorName, &e.IsRecurring, &e.Notes,
			&e.RecordedBy, &e.CreatedAt, &e.UpdatedAt)
		expenses = append(expenses, &e)
	}
	if expenses == nil {
		expenses = []*domain.CompanyExpense{}
	}
	return expenses, total, rows.Err()
}

// ── Asset CRUD ────────────────────────────────────────────────────────────────

func (r *FinancialRepository) CreateAsset(ctx context.Context, companyID, userID int,
	req *domain.CreateAssetRequest) (*domain.CompanyAsset, error) {
	purchDate, _ := time.Parse("2006-01-02", req.PurchaseDate)
	method := req.DepreciationMethod
	if method == "" {
		method = "SLM"
	}
	rate := req.DepreciationRate
	if rate == 0 {
		rate = 20.0
	}
	life := req.UsefulLifeYears
	if life == 0 {
		life = 5
	}
	var a domain.CompanyAsset
	err := r.db.QueryRow(ctx, `
		INSERT INTO company_assets (
			company_id, branch_id, asset_name, asset_code, category,
			purchase_date, purchase_cost, salvage_value, useful_life_years,
			depreciation_method, depreciation_rate, vendor_name, invoice_number,
			notes, recorded_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING asset_id, book_value, created_at, updated_at
	`, companyID, req.BranchID, req.AssetName, nullStr(req.AssetCode), req.Category,
		purchDate, req.PurchaseCost, req.SalvageValue, life,
		method, rate, nullStr(req.VendorName), nullStr(req.InvoiceNumber),
		nullStr(req.Notes), userID,
	).Scan(&a.AssetID, &a.BookValue, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.CompanyID = companyID
	a.AssetName = req.AssetName
	a.Category = req.Category
	a.PurchaseDate = purchDate
	a.PurchaseCost = req.PurchaseCost
	return &a, nil
}

// ── Capital CRUD ──────────────────────────────────────────────────────────────

func (r *FinancialRepository) CreateCapital(ctx context.Context, companyID, userID int,
	req *domain.CreateCapitalRequest) (*domain.CompanyCapital, error) {
	txDate, _ := time.Parse("2006-01-02", req.TransactionDate)
	var c domain.CompanyCapital
	err := r.db.QueryRow(ctx, `
		INSERT INTO company_capital (company_id, transaction_date, type, amount, description, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING capital_id, created_at
	`, companyID, txDate, req.Type, req.Amount, nullStr(req.Description), userID,
	).Scan(&c.CapitalID, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.CompanyID = companyID
	c.Amount = req.Amount
	c.Type = req.Type
	return &c, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
