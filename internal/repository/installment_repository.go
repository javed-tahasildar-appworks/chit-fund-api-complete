package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type InstallmentRepository struct {
	db *database.PostgresPool
}

func NewInstallmentRepository(db *database.PostgresPool) *InstallmentRepository {
	return &InstallmentRepository{db: db}
}

func (r *InstallmentRepository) GenerateBatch(ctx context.Context, req *domain.GenerateInstallmentRequest, companyID int) (*domain.Installment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return nil, domain.NewAppError("INVALID_DATE", "Invalid due date format", err)
	}

	netPayable := req.BaseInstallmentAmount - req.DividendAmount
	if netPayable < 0 {
		netPayable = 0
	}

	inst := &domain.Installment{}
	err = tx.QueryRow(ctx, `
		INSERT INTO installments (
			chit_id, month_number, base_installment_amount,
			dividend_amount, net_payable_amount, due_date, status
		) VALUES ($1,$2,$3,$4,$5,$6,'PENDING')
		RETURNING installment_id, grace_period_days, penalty_rate_per_day, created_at
	`,
		req.ChitID, req.MonthNumber, req.BaseInstallmentAmount,
		req.DividendAmount, netPayable, dueDate,
	).Scan(&inst.InstallmentID, &inst.GracePeriodDays, &inst.PenaltyRatePerDay, &inst.CreatedAt)
	if err != nil {
		return nil, err
	}

	inst.ChitID = req.ChitID
	inst.MonthNumber = req.MonthNumber
	inst.BaseInstallmentAmount = req.BaseInstallmentAmount
	inst.DividendAmount = req.DividendAmount
	inst.NetPayableAmount = netPayable
	inst.DueDate = dueDate
	inst.Status = "PENDING"

	// Generate per-subscriber installment rows
	_, err = tx.Exec(ctx, `
		INSERT INTO subscriber_installments (
			installment_id, chit_subscriber_id, subscriber_id,
			amount_due, amount_paid, balance_amount, penalty_amount, total_payable, status
		)
		SELECT $1, cs.chit_subscriber_id, cs.subscriber_id,
			$2, 0, $2, 0, $2, 'PENDING'
		FROM chit_subscribers cs WHERE cs.chit_id = $3 AND cs.status = 'ACTIVE'
	`, inst.InstallmentID, netPayable, req.ChitID)
	if err != nil {
		return nil, err
	}

	return inst, tx.Commit(ctx)
}

func (r *InstallmentRepository) Create(ctx context.Context, inst *domain.Installment) error {
	query := `
		INSERT INTO installments (
			chit_id, month_number, base_installment_amount,
			dividend_amount, net_payable_amount, due_date, status
		) VALUES ($1,$2,$3,$4,$5,$6,'PENDING')
		RETURNING installment_id, grace_period_days, penalty_rate_per_day, created_at
	`
	return r.db.QueryRow(ctx, query,
		inst.ChitID, inst.MonthNumber, inst.BaseInstallmentAmount,
		inst.DividendAmount, inst.NetPayableAmount, inst.DueDate,
	).Scan(&inst.InstallmentID, &inst.GracePeriodDays, &inst.PenaltyRatePerDay, &inst.CreatedAt)
}

func (r *InstallmentRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Installment, error) {
	query := `
		SELECT i.installment_id, i.chit_id, cg.chit_name, i.month_number,
			i.base_installment_amount, i.dividend_amount, i.net_payable_amount,
			i.due_date, i.grace_period_days, i.penalty_rate_per_day,
			i.status, i.created_at
		FROM installments i
		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
		WHERE i.installment_id = $1 AND cg.company_id = $2
	`
	rows, err := r.db.Query(ctx, query, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inst, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Installment])
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &inst, err
}

func (r *InstallmentRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Installment, int64, error) {
	base := `SELECT i.installment_id, i.chit_id, cg.chit_name, i.month_number,
		i.base_installment_amount, i.dividend_amount, i.net_payable_amount,
		i.due_date, i.grace_period_days, i.penalty_rate_per_day, i.status, i.created_at
		FROM installments i INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
		where += fmt.Sprintf(" AND i.chit_id = $%d", argPos)
		args = append(args, chitID)
		argPos++
	}
	if month, ok := filters["month_number"].(int); ok && month > 0 {
		where += fmt.Sprintf(" AND i.month_number = $%d", argPos)
		args = append(args, month)
		argPos++
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND i.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM installments i INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY i.due_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	insts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.Installment])
	return insts, total, err
}

// ─── Subscriber Installments ─────────────────────────────────────────────────

func (r *InstallmentRepository) ListSubscriberInstallments(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.SubscriberInstallment, int64, error) {
	base := `SELECT si.subscriber_installment_id, si.installment_id, si.chit_subscriber_id,
		si.subscriber_id, s.full_name AS subscriber_name, si.amount_due, si.amount_paid, si.balance_amount,
		si.penalty_amount, si.total_payable, si.payment_date, si.payment_mode,
		si.transaction_reference, si.payment_collected_by, si.receipt_number,
		si.status, si.remarks, si.created_at, si.updated_at
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
		INNER JOIN subscribers s ON si.subscriber_id = s.subscriber_id`

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
		where += fmt.Sprintf(" AND i.chit_id = $%d", argPos)
		args = append(args, chitID)
		argPos++
	}
	if subscriberID, ok := filters["subscriber_id"].(int); ok && subscriberID > 0 {
		where += fmt.Sprintf(" AND si.subscriber_id = $%d", argPos)
		args = append(args, subscriberID)
		argPos++
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND si.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	var total int64
	countQ := `SELECT COUNT(*) FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`
	r.db.QueryRow(ctx, countQ+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY si.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	sis, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.SubscriberInstallment])
	return sis, total, err
}

func (r *InstallmentRepository) GetSubscriberInstallmentByID(ctx context.Context, id, companyID int) (*domain.SubscriberInstallment, error) {
	query := `
		SELECT si.subscriber_installment_id, si.installment_id, si.chit_subscriber_id,
			si.subscriber_id, s.full_name AS subscriber_name, si.amount_due, si.amount_paid, si.balance_amount,
			si.penalty_amount, si.total_payable, si.payment_date, si.payment_mode,
			si.transaction_reference, si.payment_collected_by, si.receipt_number,
			si.status, si.remarks, si.created_at, si.updated_at
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
		INNER JOIN subscribers s ON si.subscriber_id = s.subscriber_id
		WHERE si.subscriber_installment_id = $1 AND cg.company_id = $2
	`
	rows, err := r.db.Query(ctx, query, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	si, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.SubscriberInstallment])
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &si, err
}

// ─── Payment Transactions ─────────────────────────────────────────────────────

func (r *InstallmentRepository) RecordPayment(ctx context.Context, siID, collectedBy int, req *domain.RecordPaymentRequest) (*domain.PaymentTransaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Get subscriber installment
	var si domain.SubscriberInstallment
	err = tx.QueryRow(ctx, `
		SELECT subscriber_installment_id, subscriber_id, chit_subscriber_id,
			amount_due, amount_paid, balance_amount, total_payable
		FROM subscriber_installments WHERE subscriber_installment_id = $1
	`, siID).Scan(
		&si.SubscriberInstallmentID, &si.SubscriberID, &si.ChitSubscriberID,
		&si.AmountDue, &si.AmountPaid, &si.BalanceAmount, &si.TotalPayable,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	receiptNo := fmt.Sprintf("RCP-%d-%d", siID, time.Now().UnixMilli())
	newPaid := si.AmountPaid + req.Amount
	newBalance := si.TotalPayable - newPaid
	status := "PARTIAL"
	if newBalance <= 0 {
		status = "PAID"
		newBalance = 0
	}

	_, err = tx.Exec(ctx, `
		UPDATE subscriber_installments SET
			amount_paid = $1, balance_amount = $2, status = $3,
			payment_date = NOW(), payment_mode = $4,
			transaction_reference = $5, payment_collected_by = $6,
			receipt_number = $7, updated_at = NOW()
		WHERE subscriber_installment_id = $8
	`, newPaid, newBalance, status, req.PaymentMode, req.TransactionReference,
		collectedBy, receiptNo, siID)
	if err != nil {
		return nil, err
	}

	var pt domain.PaymentTransaction
	err = tx.QueryRow(ctx, `
		INSERT INTO payment_transactions (
			subscriber_installment_id, subscriber_id, transaction_type,
			amount, payment_mode, transaction_reference,
			transaction_date, collected_by, receipt_number, remarks, status
		) VALUES ($1,$2,'INSTALLMENT',$3,$4,$5,NOW(),$6,$7,$8,'COMPLETED')
		RETURNING transaction_id, created_at
	`,
		siID, si.SubscriberID, req.Amount, req.PaymentMode, req.TransactionReference,
		collectedBy, receiptNo, req.Remarks,
	).Scan(&pt.TransactionID, &pt.CreatedAt)
	if err != nil {
		return nil, err
	}

	pt.Amount = req.Amount
	pt.ReceiptNumber = &receiptNo
	pt.Status = "COMPLETED"

	return &pt, tx.Commit(ctx)
}

func (r *InstallmentRepository) ListPaymentTransactions(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.PaymentTransaction, int64, error) {
	base := `SELECT pt.transaction_id, pt.subscriber_installment_id, pt.subscriber_id,
		s.full_name AS subscriber_name,
		pt.chit_id, COALESCE(cg.chit_name, '') AS chit_name,
		pt.transaction_type, pt.amount, pt.payment_mode,
		pt.transaction_reference, pt.transaction_date, pt.collected_by,
		pt.branch_id, COALESCE(b.branch_name, '') AS branch_name,
		pt.receipt_number, pt.remarks, pt.status, pt.created_at
		FROM payment_transactions pt
		INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
		LEFT JOIN chit_groups cg ON pt.chit_id = cg.chit_id
		LEFT JOIN branches b ON pt.branch_id = b.branch_id`

	args := []interface{}{companyID}
	where := ` WHERE s.company_id = $1`
	argPos := 2

	if subscriberID, ok := filters["subscriber_id"].(int); ok && subscriberID > 0 {
		where += fmt.Sprintf(" AND pt.subscriber_id = $%d", argPos)
		args = append(args, subscriberID)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM payment_transactions pt INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id`+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY pt.transaction_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	pts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.PaymentTransaction])
	return pts, total, err
}

func (r *InstallmentRepository) GetPaymentTransactionByID(ctx context.Context, id, companyID int) (*domain.PaymentTransaction, error) {
	query := `
		SELECT pt.transaction_id, pt.subscriber_installment_id, pt.subscriber_id,
			s.full_name AS subscriber_name,
			pt.chit_id, COALESCE(cg.chit_name, '') AS chit_name,
			pt.transaction_type, pt.amount, pt.payment_mode,
			pt.transaction_reference, pt.transaction_date, pt.collected_by,
			pt.branch_id, COALESCE(b.branch_name, '') AS branch_name,
			pt.receipt_number, pt.remarks, pt.status, pt.created_at
		FROM payment_transactions pt
		INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
		LEFT JOIN chit_groups cg ON pt.chit_id = cg.chit_id
		LEFT JOIN branches b ON pt.branch_id = b.branch_id
		WHERE pt.transaction_id = $1 AND s.company_id = $2
	`
	rows, err := r.db.Query(ctx, query, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pt, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.PaymentTransaction])
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &pt, err
}

// package repository

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"chit-fund-api/internal/domain"
// 	"chit-fund-api/pkg/database"

// 	"github.com/jackc/pgx/v5"
// )

// type InstallmentRepository struct {
// 	db *database.PostgresPool
// }

// func NewInstallmentRepository(db *database.PostgresPool) *InstallmentRepository {
// 	return &InstallmentRepository{db: db}
// }

// func (r *InstallmentRepository) GenerateBatch(ctx context.Context, req *domain.GenerateInstallmentRequest, companyID int) (*domain.Installment, error) {
// 	tx, err := r.db.Begin(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer tx.Rollback(ctx)

// 	dueDate, err := time.Parse("2006-01-02", req.DueDate)
// 	if err != nil {
// 		return nil, domain.NewAppError("INVALID_DATE", "Invalid due date format", err)
// 	}

// 	netPayable := req.BaseInstallmentAmount - req.DividendAmount
// 	if netPayable < 0 {
// 		netPayable = 0
// 	}

// 	inst := &domain.Installment{}
// 	err = tx.QueryRow(ctx, `
// 		INSERT INTO installments (
// 			chit_id, month_number, base_installment_amount,
// 			dividend_amount, net_payable_amount, due_date, status
// 		) VALUES ($1,$2,$3,$4,$5,$6,'PENDING')
// 		RETURNING installment_id, grace_period_days, penalty_rate_per_day, created_at
// 	`,
// 		req.ChitID, req.MonthNumber, req.BaseInstallmentAmount,
// 		req.DividendAmount, netPayable, dueDate,
// 	).Scan(&inst.InstallmentID, &inst.GracePeriodDays, &inst.PenaltyRatePerDay, &inst.CreatedAt)
// 	if err != nil {
// 		return nil, err
// 	}

// 	inst.ChitID = req.ChitID
// 	inst.MonthNumber = req.MonthNumber
// 	inst.BaseInstallmentAmount = req.BaseInstallmentAmount
// 	inst.DividendAmount = req.DividendAmount
// 	inst.NetPayableAmount = netPayable
// 	inst.DueDate = dueDate
// 	inst.Status = "PENDING"

// 	// Generate per-subscriber installment rows
// 	_, err = tx.Exec(ctx, `
// 		INSERT INTO subscriber_installments (
// 			installment_id, chit_subscriber_id, subscriber_id,
// 			amount_due, amount_paid, balance_amount, penalty_amount, total_payable, status
// 		)
// 		SELECT $1, cs.chit_subscriber_id, cs.subscriber_id,
// 			$2, 0, $2, 0, $2, 'PENDING'
// 		FROM chit_subscribers cs WHERE cs.chit_id = $3 AND cs.status = 'ACTIVE'
// 	`, inst.InstallmentID, netPayable, req.ChitID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return inst, tx.Commit(ctx)
// }

// func (r *InstallmentRepository) Create(ctx context.Context, inst *domain.Installment) error {
// 	query := `
// 		INSERT INTO installments (
// 			chit_id, month_number, base_installment_amount,
// 			dividend_amount, net_payable_amount, due_date, status
// 		) VALUES ($1,$2,$3,$4,$5,$6,'PENDING')
// 		RETURNING installment_id, grace_period_days, penalty_rate_per_day, created_at
// 	`
// 	return r.db.QueryRow(ctx, query,
// 		inst.ChitID, inst.MonthNumber, inst.BaseInstallmentAmount,
// 		inst.DividendAmount, inst.NetPayableAmount, inst.DueDate,
// 	).Scan(&inst.InstallmentID, &inst.GracePeriodDays, &inst.PenaltyRatePerDay, &inst.CreatedAt)
// }

// func (r *InstallmentRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Installment, error) {
// 	query := `
// 		SELECT i.installment_id, i.chit_id, i.month_number,
// 			i.base_installment_amount, i.dividend_amount, i.net_payable_amount,
// 			i.due_date, i.grace_period_days, i.penalty_rate_per_day,
// 			i.status, i.created_at
// 		FROM installments i
// 		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
// 		WHERE i.installment_id = $1 AND cg.company_id = $2
// 	`
// 	rows, err := r.db.Query(ctx, query, id, companyID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	inst, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Installment])
// 	if err == pgx.ErrNoRows {
// 		return nil, domain.ErrNotFound
// 	}
// 	return &inst, err
// }

// func (r *InstallmentRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Installment, int64, error) {
// 	base := `SELECT i.installment_id, i.chit_id, i.month_number,
// 		i.base_installment_amount, i.dividend_amount, i.net_payable_amount,
// 		i.due_date, i.grace_period_days, i.penalty_rate_per_day, i.status, i.created_at
// 		FROM installments i INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`

// 	args := []interface{}{companyID}
// 	where := " WHERE cg.company_id = $1"
// 	argPos := 2

// 	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
// 		where += fmt.Sprintf(" AND i.chit_id = $%d", argPos)
// 		args = append(args, chitID)
// 		argPos++
// 	}
// 	if month, ok := filters["month_number"].(int); ok && month > 0 {
// 		where += fmt.Sprintf(" AND i.month_number = $%d", argPos)
// 		args = append(args, month)
// 		argPos++
// 	}
// 	if status, ok := filters["status"].(string); ok && status != "" {
// 		where += fmt.Sprintf(" AND i.status = $%d", argPos)
// 		args = append(args, status)
// 		argPos++
// 	}

// 	var total int64
// 	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM installments i INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`+where, args...).Scan(&total)

// 	limit := 20
// 	offset := 0
// 	if l, ok := filters["limit"].(int); ok && l > 0 {
// 		limit = l
// 	}
// 	if o, ok := filters["offset"].(int); ok {
// 		offset = o
// 	}
// 	query := base + where + fmt.Sprintf(" ORDER BY i.due_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
// 	args = append(args, limit, offset)

// 	rows, err := r.db.Query(ctx, query, args...)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	defer rows.Close()

// 	insts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.Installment])
// 	return insts, total, err
// }

// // ─── Subscriber Installments ─────────────────────────────────────────────────

// func (r *InstallmentRepository) ListSubscriberInstallments(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.SubscriberInstallment, int64, error) {
// 	base := `SELECT si.subscriber_installment_id, si.installment_id, si.chit_subscriber_id,
// 		si.subscriber_id, si.amount_due, si.amount_paid, si.balance_amount,
// 		si.penalty_amount, si.total_payable, si.payment_date, si.payment_mode,
// 		si.transaction_reference, si.payment_collected_by, si.receipt_number,
// 		si.status, si.remarks, si.created_at, si.updated_at
// 		FROM subscriber_installments si
// 		INNER JOIN installments i ON si.installment_id = i.installment_id
// 		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`

// 	args := []interface{}{companyID}
// 	where := " WHERE cg.company_id = $1"
// 	argPos := 2

// 	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
// 		where += fmt.Sprintf(" AND i.chit_id = $%d", argPos)
// 		args = append(args, chitID)
// 		argPos++
// 	}
// 	if subscriberID, ok := filters["subscriber_id"].(int); ok && subscriberID > 0 {
// 		where += fmt.Sprintf(" AND si.subscriber_id = $%d", argPos)
// 		args = append(args, subscriberID)
// 		argPos++
// 	}
// 	if status, ok := filters["status"].(string); ok && status != "" {
// 		where += fmt.Sprintf(" AND si.status = $%d", argPos)
// 		args = append(args, status)
// 		argPos++
// 	}

// 	var total int64
// 	countQ := `SELECT COUNT(*) FROM subscriber_installments si
// 		INNER JOIN installments i ON si.installment_id = i.installment_id
// 		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id`
// 	r.db.QueryRow(ctx, countQ+where, args...).Scan(&total)

// 	limit := 20
// 	offset := 0
// 	if l, ok := filters["limit"].(int); ok && l > 0 {
// 		limit = l
// 	}
// 	if o, ok := filters["offset"].(int); ok {
// 		offset = o
// 	}
// 	query := base + where + fmt.Sprintf(" ORDER BY si.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
// 	args = append(args, limit, offset)

// 	rows, err := r.db.Query(ctx, query, args...)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	defer rows.Close()

// 	sis, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.SubscriberInstallment])
// 	return sis, total, err
// }

// func (r *InstallmentRepository) GetSubscriberInstallmentByID(ctx context.Context, id, companyID int) (*domain.SubscriberInstallment, error) {
// 	query := `
// 		SELECT si.subscriber_installment_id, si.installment_id, si.chit_subscriber_id,
// 			si.subscriber_id, si.amount_due, si.amount_paid, si.balance_amount,
// 			si.penalty_amount, si.total_payable, si.payment_date, si.payment_mode,
// 			si.transaction_reference, si.payment_collected_by, si.receipt_number,
// 			si.status, si.remarks, si.created_at, si.updated_at
// 		FROM subscriber_installments si
// 		INNER JOIN installments i ON si.installment_id = i.installment_id
// 		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
// 		WHERE si.subscriber_installment_id = $1 AND cg.company_id = $2
// 	`
// 	rows, err := r.db.Query(ctx, query, id, companyID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	si, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.SubscriberInstallment])
// 	if err == pgx.ErrNoRows {
// 		return nil, domain.ErrNotFound
// 	}
// 	return &si, err
// }

// // ─── Payment Transactions ─────────────────────────────────────────────────────

// func (r *InstallmentRepository) RecordPayment(ctx context.Context, siID, collectedBy int, req *domain.RecordPaymentRequest) (*domain.PaymentTransaction, error) {
// 	tx, err := r.db.Begin(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer tx.Rollback(ctx)

// 	// Get subscriber installment
// 	var si domain.SubscriberInstallment
// 	err = tx.QueryRow(ctx, `
// 		SELECT subscriber_installment_id, subscriber_id, chit_subscriber_id,
// 			amount_due, amount_paid, balance_amount, total_payable
// 		FROM subscriber_installments WHERE subscriber_installment_id = $1
// 	`, siID).Scan(
// 		&si.SubscriberInstallmentID, &si.SubscriberID, &si.ChitSubscriberID,
// 		&si.AmountDue, &si.AmountPaid, &si.BalanceAmount, &si.TotalPayable,
// 	)
// 	if err != nil {
// 		return nil, domain.ErrNotFound
// 	}

// 	receiptNo := fmt.Sprintf("RCP-%d-%d", siID, time.Now().UnixMilli())
// 	newPaid := si.AmountPaid + req.Amount
// 	newBalance := si.TotalPayable - newPaid
// 	status := "PARTIAL"
// 	if newBalance <= 0 {
// 		status = "PAID"
// 		newBalance = 0
// 	}

// 	_, err = tx.Exec(ctx, `
// 		UPDATE subscriber_installments SET
// 			amount_paid = $1, balance_amount = $2, status = $3,
// 			payment_date = NOW(), payment_mode = $4,
// 			transaction_reference = $5, payment_collected_by = $6,
// 			receipt_number = $7, updated_at = NOW()
// 		WHERE subscriber_installment_id = $8
// 	`, newPaid, newBalance, status, req.PaymentMode, req.TransactionReference,
// 		collectedBy, receiptNo, siID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var pt domain.PaymentTransaction
// 	err = tx.QueryRow(ctx, `
// 		INSERT INTO payment_transactions (
// 			subscriber_installment_id, subscriber_id, transaction_type,
// 			amount, payment_mode, transaction_reference,
// 			transaction_date, collected_by, receipt_number, remarks, status
// 		) VALUES ($1,$2,'INSTALLMENT',$3,$4,$5,NOW(),$6,$7,$8,'COMPLETED')
// 		RETURNING transaction_id, created_at
// 	`,
// 		siID, si.SubscriberID, req.Amount, req.PaymentMode, req.TransactionReference,
// 		collectedBy, receiptNo, req.Remarks,
// 	).Scan(&pt.TransactionID, &pt.CreatedAt)
// 	if err != nil {
// 		return nil, err
// 	}

// 	pt.Amount = req.Amount
// 	pt.ReceiptNumber = &receiptNo
// 	pt.Status = "COMPLETED"

// 	return &pt, tx.Commit(ctx)
// }

// func (r *InstallmentRepository) ListPaymentTransactions(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.PaymentTransaction, int64, error) {
// 	base := `SELECT pt.transaction_id, pt.subscriber_installment_id, pt.subscriber_id,
// 		pt.chit_id, pt.transaction_type, pt.amount, pt.payment_mode,
// 		pt.transaction_reference, pt.transaction_date, pt.collected_by,
// 		pt.branch_id, pt.receipt_number, pt.remarks, pt.status, pt.created_at
// 		FROM payment_transactions pt`

// 	args := []interface{}{companyID}
// 	where := ` WHERE EXISTS (
// 		SELECT 1 FROM subscribers s WHERE s.subscriber_id = pt.subscriber_id AND s.company_id = $1
// 	)`
// 	argPos := 2

// 	if subscriberID, ok := filters["subscriber_id"].(int); ok && subscriberID > 0 {
// 		where += fmt.Sprintf(" AND pt.subscriber_id = $%d", argPos)
// 		args = append(args, subscriberID)
// 		argPos++
// 	}

// 	var total int64
// 	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM payment_transactions pt`+where, args...).Scan(&total)

// 	limit := 20
// 	offset := 0
// 	if l, ok := filters["limit"].(int); ok && l > 0 {
// 		limit = l
// 	}
// 	if o, ok := filters["offset"].(int); ok {
// 		offset = o
// 	}
// 	query := base + where + fmt.Sprintf(" ORDER BY pt.transaction_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
// 	args = append(args, limit, offset)

// 	rows, err := r.db.Query(ctx, query, args...)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	defer rows.Close()

// 	pts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.PaymentTransaction])
// 	return pts, total, err
// }

// func (r *InstallmentRepository) GetPaymentTransactionByID(ctx context.Context, id, companyID int) (*domain.PaymentTransaction, error) {
// 	query := `
// 		SELECT pt.transaction_id, pt.subscriber_installment_id, pt.subscriber_id,
// 			pt.chit_id, pt.transaction_type, pt.amount, pt.payment_mode,
// 			pt.transaction_reference, pt.transaction_date, pt.collected_by,
// 			pt.branch_id, pt.receipt_number, pt.remarks, pt.status, pt.created_at
// 		FROM payment_transactions pt
// 		INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
// 		WHERE pt.transaction_id = $1 AND s.company_id = $2
// 	`
// 	rows, err := r.db.Query(ctx, query, id, companyID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	pt, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.PaymentTransaction])
// 	if err == pgx.ErrNoRows {
// 		return nil, domain.ErrNotFound
// 	}
// 	return &pt, err
// }
