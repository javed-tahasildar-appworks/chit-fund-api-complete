package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type PayoutRepository struct {
	db *database.PostgresPool
}

func NewPayoutRepository(db *database.PostgresPool) *PayoutRepository {
	return &PayoutRepository{db: db}
}

// payoutCols is the canonical SELECT column list for payouts.
// Using an explicit list + explicit Scan avoids pgx.RowToStructByName failures
// when new columns are added to the table by migrations.
const payoutCols = `
	p.payout_id, cg.company_id,
	p.chit_id, cg.chit_code, cg.chit_name,
	p.auction_id, p.chit_subscriber_id,
	p.subscriber_id, s.full_name AS subscriber_name,
	p.month_number, p.gross_chit_value, p.discount_amount, p.foreman_commission,
	p.net_payout, p.payout_date, p.payout_mode, p.transaction_reference,
	p.bank_id, p.paid_by, pu.full_name AS paid_by_name,
	p.receipt_number, p.status,
	p.approved_by, au.full_name AS approved_by_name,
	p.approval_date, p.remarks,
	p.created_at, p.updated_at`

const payoutJoins = `
	FROM payouts p
	INNER JOIN chit_groups cg  ON p.chit_id      = cg.chit_id
	INNER JOIN subscribers s   ON p.subscriber_id = s.subscriber_id
	LEFT  JOIN users pu        ON p.paid_by       = pu.user_id
	LEFT  JOIN users au        ON p.approved_by   = au.user_id`

func scanPayout(row pgx.Row, p *domain.Payout) error {
	return row.Scan(
		&p.PayoutID, &p.CompanyID,
		&p.ChitID, &p.ChitCode, &p.ChitName,
		&p.AuctionID, &p.ChitSubscriberID,
		&p.SubscriberID, &p.SubscriberName,
		&p.MonthNumber, &p.GrossChitValue, &p.DiscountAmount, &p.ForemanCommission,
		&p.NetPayout, &p.PayoutDate, &p.PayoutMode, &p.TransactionReference,
		&p.BankID, &p.PaidBy, &p.PaidByName,
		&p.ReceiptNumber, &p.Status,
		&p.ApprovedBy, &p.ApprovedByName,
		&p.ApprovalDate, &p.Remarks,
		&p.CreatedAt, &p.UpdatedAt,
	)
}

// ── Initiate ──────────────────────────────────────────────────────────────────

func (r *PayoutRepository) Initiate(ctx context.Context, payout *domain.Payout) error {
	receiptNo := fmt.Sprintf("PAY-%d-%d", payout.AuctionID, time.Now().UnixMilli())
	payout.ReceiptNumber = &receiptNo

	return r.db.QueryRow(ctx, `
		INSERT INTO payouts (
			chit_id, auction_id, chit_subscriber_id, subscriber_id,
			month_number, gross_chit_value, discount_amount, foreman_commission,
			net_payout, bank_id, receipt_number, status, remarks
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'PENDING',$12)
		RETURNING payout_id, created_at, updated_at
	`,
		payout.ChitID, payout.AuctionID, payout.ChitSubscriberID, payout.SubscriberID,
		payout.MonthNumber, payout.GrossChitValue, payout.DiscountAmount,
		payout.ForemanCommission, payout.NetPayout, payout.BankID,
		receiptNo, payout.Remarks,
	).Scan(&payout.PayoutID, &payout.CreatedAt, &payout.UpdatedAt)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (r *PayoutRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Payout, error) {
	var p domain.Payout
	err := scanPayout(r.db.QueryRow(ctx,
		`SELECT `+payoutCols+payoutJoins+`
		WHERE p.payout_id = $1 AND cg.company_id = $2`,
		id, companyID), &p)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *PayoutRepository) List(ctx context.Context, companyID int,
	filters map[string]interface{}) ([]*domain.Payout, int64, error) {

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND p.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
		where += fmt.Sprintf(" AND p.chit_id = $%d", argPos)
		args = append(args, chitID)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM payouts p
		 INNER JOIN chit_groups cg ON p.chit_id = cg.chit_id`+where,
		args...,
	).Scan(&total)

	limit, offset := 20, 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}

	query := `SELECT ` + payoutCols + payoutJoins +
		where +
		fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payouts []*domain.Payout
	for rows.Next() {
		var p domain.Payout
		if err := scanPayout(rows, &p); err != nil {
			return nil, 0, err
		}
		payouts = append(payouts, &p)
	}
	if payouts == nil {
		payouts = []*domain.Payout{}
	}
	return payouts, total, rows.Err()
}

// ── Approve ───────────────────────────────────────────────────────────────────

func (r *PayoutRepository) Approve(ctx context.Context, id, companyID, approvedBy int, remarks string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE payouts SET status = 'APPROVED', approved_by = $1, approval_date = NOW(),
			remarks = COALESCE(NULLIF($2,''), remarks), updated_at = NOW()
		WHERE payout_id = $3
		  AND status = 'PENDING'
		  AND EXISTS (
			SELECT 1 FROM chit_groups cg
			WHERE cg.chit_id = payouts.chit_id AND cg.company_id = $4
		  )
	`, approvedBy, remarks, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPayoutNotPending
	}
	return nil
}

// ── Disburse ──────────────────────────────────────────────────────────────────

func (r *PayoutRepository) Disburse(ctx context.Context, id, companyID, paidBy int,
	req *domain.DisbursePayoutRequest) error {

	payoutDate, err := time.Parse("2006-01-02", req.PayoutDate)
	if err != nil {
		return domain.NewAppError("INVALID_DATE", "Invalid payout date", err)
	}
	result, err := r.db.Exec(ctx, `
		UPDATE payouts SET status = 'DISBURSED', paid_by = $1, payout_date = $2,
			payout_mode = $3, transaction_reference = $4,
			remarks = COALESCE(NULLIF($5,''), remarks), updated_at = NOW()
		WHERE payout_id = $6
		  AND status = 'APPROVED'
		  AND EXISTS (
			SELECT 1 FROM chit_groups cg
			WHERE cg.chit_id = payouts.chit_id AND cg.company_id = $7
		  )
	`, paidBy, payoutDate, req.PayoutMode, req.TransactionReference, req.Remarks, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("payout not found or not in approved state")
	}
	return nil
}

// ── CalculateDividend ─────────────────────────────────────────────────────────

func (r *PayoutRepository) CalculateDividend(ctx context.Context, auctionID, companyID int) (*domain.Dividend, error) {
	var d domain.Dividend
	var scanChitValue, scanForemanPct float64

	err := r.db.QueryRow(ctx, `
		SELECT a.auction_id, a.chit_id, a.month_number,
			cg.chit_value, cg.total_subscribers, cg.foreman_commission_percent
		FROM auctions a
		INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
		WHERE a.auction_id = $1 AND cg.company_id = $2 AND a.status = 'COMPLETED'
	`, auctionID, companyID).Scan(
		&d.AuctionID, &d.ChitID, &d.MonthNumber,
		&scanChitValue, &d.TotalSubscribers, &scanForemanPct,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var discountAmount float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(ab.bid_discount_amount, 0)
		FROM auctions a
		LEFT JOIN auction_bids ab ON ab.bid_id = a.winning_bid_id
		WHERE a.auction_id = $1
	`, auctionID).Scan(&discountAmount)

	foremanComm := discountAmount * scanForemanPct / 100
	distributable := discountAmount - foremanComm
	if distributable < 0 {
		distributable = 0
	}
	dividendPer := 0.0
	if d.TotalSubscribers > 0 {
		dividendPer = distributable / float64(d.TotalSubscribers)
	}

	d.TotalDiscountAmount = discountAmount
	d.ForemanCommission = foremanComm
	d.DistributableAmount = distributable
	d.DividendPerSubscriber = dividendPer
	d.Status = "CALCULATED"

	err = r.db.QueryRow(ctx, `
		INSERT INTO dividends (
			auction_id, chit_id, month_number,
			total_discount_amount, foreman_commission, distributable_amount,
			total_subscribers, dividend_per_subscriber, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'CALCULATED')
		ON CONFLICT (auction_id) DO UPDATE SET
			total_discount_amount   = EXCLUDED.total_discount_amount,
			foreman_commission      = EXCLUDED.foreman_commission,
			distributable_amount    = EXCLUDED.distributable_amount,
			dividend_per_subscriber = EXCLUDED.dividend_per_subscriber,
			status                  = 'CALCULATED'
		RETURNING dividend_id, created_at
	`, d.AuctionID, d.ChitID, d.MonthNumber,
		d.TotalDiscountAmount, d.ForemanCommission, d.DistributableAmount,
		d.TotalSubscribers, d.DividendPerSubscriber,
	).Scan(&d.DividendID, &d.CreatedAt)

	return &d, err
}

// ── PostDividend ──────────────────────────────────────────────────────────────

func (r *PayoutRepository) PostDividend(ctx context.Context, dividendID, companyID int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var d domain.Dividend
	err = tx.QueryRow(ctx, `
		SELECT dividend_id, chit_id, dividend_per_subscriber, status
		FROM dividends WHERE dividend_id = $1
	`, dividendID).Scan(&d.DividendID, &d.ChitID, &d.DividendPerSubscriber, &d.Status)
	if err != nil {
		return domain.ErrNotFound
	}
	if d.Status == "POSTED" {
		return fmt.Errorf("dividend already posted")
	}

	_, err = tx.Exec(ctx, `
		UPDATE subscriber_installments si SET
			amount_due     = GREATEST(0, si.amount_due     - $1),
			total_payable  = GREATEST(0, si.total_payable  - $1),
			balance_amount = GREATEST(0, si.balance_amount - $1)
		FROM chit_subscribers cs
		WHERE cs.chit_id = $2 AND cs.status = 'ACTIVE'
		  AND si.chit_subscriber_id = cs.chit_subscriber_id
		  AND si.status = 'PENDING'
	`, d.DividendPerSubscriber, d.ChitID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE dividends SET status = 'POSTED', distribution_date = NOW()
		WHERE dividend_id = $1
	`, dividendID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── ListDividends ─────────────────────────────────────────────────────────────

func (r *PayoutRepository) ListDividends(ctx context.Context, companyID int,
	filters map[string]interface{}) ([]*domain.Dividend, int64, error) {

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
		where += fmt.Sprintf(" AND d.chit_id = $%d", argPos)
		args = append(args, chitID)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM dividends d
		 INNER JOIN chit_groups cg ON d.chit_id = cg.chit_id`+where,
		args...,
	).Scan(&total)

	limit, offset := 20, 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}

	query := `
		SELECT d.dividend_id, d.auction_id, d.chit_id,
			cg.chit_code, cg.chit_name,
			d.month_number,
			d.total_discount_amount, d.foreman_commission, d.distributable_amount,
			d.total_subscribers, d.dividend_per_subscriber, d.distribution_date,
			d.status, d.created_at
		FROM dividends d
		INNER JOIN chit_groups cg ON d.chit_id = cg.chit_id` +
		where +
		fmt.Sprintf(" ORDER BY d.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var divs []*domain.Dividend
	for rows.Next() {
		var d domain.Dividend
		if err := rows.Scan(
			&d.DividendID, &d.AuctionID, &d.ChitID,
			&d.ChitCode, &d.ChitName,
			&d.MonthNumber,
			&d.TotalDiscountAmount, &d.ForemanCommission, &d.DistributableAmount,
			&d.TotalSubscribers, &d.DividendPerSubscriber, &d.DistributionDate,
			&d.Status, &d.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		divs = append(divs, &d)
	}
	if divs == nil {
		divs = []*domain.Dividend{}
	}
	return divs, total, rows.Err()
}
