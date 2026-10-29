package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type DefaultRepository struct {
	db *database.PostgresPool
}

func NewDefaultRepository(db *database.PostgresPool) *DefaultRepository {
	return &DefaultRepository{db: db}
}

func (r *DefaultRepository) DB() *database.PostgresPool {
	return r.db
}

func (r *DefaultRepository) Detect(ctx context.Context, companyID int) (int, error) {
	result, err := r.db.Exec(ctx, `
		INSERT INTO defaults (
			subscriber_installment_id, chit_subscriber_id, subscriber_id, chit_id,
			default_date, installment_due_date, days_overdue,
			outstanding_principal, penalty_amount, total_outstanding, status
		)
		SELECT
			si.subscriber_installment_id, si.chit_subscriber_id, si.subscriber_id,
			cs.chit_id,
			CURRENT_DATE,
			i.due_date,
			(CURRENT_DATE - i.due_date::date),
			si.amount_due,
			si.penalty_amount,
			si.total_payable,
			'ACTIVE'
		FROM subscriber_installments si
		INNER JOIN chit_subscribers cs ON si.chit_subscriber_id = cs.chit_subscriber_id
		INNER JOIN chit_groups cg      ON cs.chit_id = cg.chit_id
		INNER JOIN installments i      ON si.installment_id = i.installment_id
		WHERE cg.company_id = $1
		  AND si.status IN ('PENDING', 'PARTIAL')
		  AND i.due_date < NOW() - INTERVAL '7 days'
		  AND NOT EXISTS (
			SELECT 1 FROM defaults d
			WHERE d.subscriber_installment_id = si.subscriber_installment_id
			  AND d.status = 'ACTIVE'
		  )
	`, companyID)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func (r *DefaultRepository) FlagManual(ctx context.Context, def *domain.Default) error {
	query := `
		INSERT INTO defaults (
			subscriber_installment_id, chit_subscriber_id, subscriber_id, chit_id,
			default_date, installment_due_date, days_overdue,
			outstanding_principal, penalty_amount, total_outstanding,
			remarks, status
		) VALUES ($1,$2,$3,$4,CURRENT_DATE,$5,0,$6,$7,$8,$9,'ACTIVE')
		RETURNING default_id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		def.SubscriberInstallmentID, def.ChitSubscriberID, def.SubscriberID, def.ChitID,
		def.InstallmentDueDate,
		def.OutstandingPrincipal, def.PenaltyAmount, def.TotalOutstanding,
		def.Remarks,
	).Scan(&def.DefaultID, &def.CreatedAt, &def.UpdatedAt)
}

func (r *DefaultRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Default, error) {
	query := `
		SELECT d.default_id, d.subscriber_installment_id, d.chit_subscriber_id,
			d.subscriber_id, d.chit_id,
			d.default_date, d.installment_due_date, d.days_overdue,
			d.outstanding_principal, d.penalty_amount, d.total_outstanding,
			d.reminder_sent_count, d.last_reminder_date,
			d.legal_notice_sent, d.legal_notice_date,
			d.status, d.cleared_date, d.remarks,
			d.created_at, d.updated_at,
			s.full_name AS subscriber_name, cg.chit_code
		FROM defaults d
		INNER JOIN subscribers s  ON d.subscriber_id = s.subscriber_id
		INNER JOIN chit_groups cg ON d.chit_id = cg.chit_id
		WHERE d.default_id = $1 AND cg.company_id = $2
	`
	var def domain.Default
	err := r.db.QueryRow(ctx, query, id, companyID).Scan(
		&def.DefaultID, &def.SubscriberInstallmentID, &def.ChitSubscriberID,
		&def.SubscriberID, &def.ChitID,
		&def.DefaultDate, &def.InstallmentDueDate, &def.DaysOverdue,
		&def.OutstandingPrincipal, &def.PenaltyAmount, &def.TotalOutstanding,
		&def.ReminderSentCount, &def.LastReminderDate,
		&def.LegalNoticeSent, &def.LegalNoticeDate,
		&def.Status, &def.ClearedDate, &def.Remarks,
		&def.CreatedAt, &def.UpdatedAt,
		&def.SubscriberName, &def.ChitCode,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &def, err
}

func (r *DefaultRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Default, int64, error) {
	base := `
		SELECT d.default_id, d.subscriber_installment_id, d.chit_subscriber_id,
			d.subscriber_id, d.chit_id,
			d.default_date, d.installment_due_date, d.days_overdue,
			d.outstanding_principal, d.penalty_amount, d.total_outstanding,
			d.reminder_sent_count, d.last_reminder_date,
			d.legal_notice_sent, d.legal_notice_date,
			d.status, d.cleared_date, d.remarks,
			d.created_at, d.updated_at,
			s.full_name AS subscriber_name, cg.chit_code
		FROM defaults d
		INNER JOIN subscribers s  ON d.subscriber_id = s.subscriber_id
		INNER JOIN chit_groups cg ON d.chit_id = cg.chit_id`

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND d.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if branchID, ok := filters["branch_id"].(int); ok && branchID > 0 {
		where += fmt.Sprintf(" AND cg.branch_id = $%d", argPos)
		args = append(args, branchID)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM defaults d INNER JOIN chit_groups cg ON d.chit_id = cg.chit_id`+where,
		args...,
	).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY d.default_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var defs []*domain.Default
	for rows.Next() {
		var def domain.Default
		if err := rows.Scan(
			&def.DefaultID, &def.SubscriberInstallmentID, &def.ChitSubscriberID,
			&def.SubscriberID, &def.ChitID,
			&def.DefaultDate, &def.InstallmentDueDate, &def.DaysOverdue,
			&def.OutstandingPrincipal, &def.PenaltyAmount, &def.TotalOutstanding,
			&def.ReminderSentCount, &def.LastReminderDate,
			&def.LegalNoticeSent, &def.LegalNoticeDate,
			&def.Status, &def.ClearedDate, &def.Remarks,
			&def.CreatedAt, &def.UpdatedAt,
			&def.SubscriberName, &def.ChitCode,
		); err != nil {
			return nil, 0, err
		}
		defs = append(defs, &def)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return defs, total, nil
}

func (r *DefaultRepository) Resolve(ctx context.Context, id, companyID, resolvedBy int, notes string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE defaults SET
			status = 'CLEARED',
			cleared_date = CURRENT_DATE,
			remarks = $1,
			updated_at = NOW()
		WHERE default_id = $2
		  AND status = 'ACTIVE'
		  AND EXISTS (SELECT 1 FROM chit_groups cg WHERE cg.chit_id = defaults.chit_id AND cg.company_id = $3)
	`, notes, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DefaultRepository) SendReminder(ctx context.Context, id, companyID int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE defaults SET
			reminder_sent_count = reminder_sent_count + 1,
			last_reminder_date  = CURRENT_DATE,
			updated_at          = NOW()
		WHERE default_id = $1
		  AND EXISTS (SELECT 1 FROM chit_groups cg WHERE cg.chit_id = defaults.chit_id AND cg.company_id = $2)
	`, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ─── Legal Notices ────────────────────────────────────────────────────────────

type LegalNoticeRepository struct {
	db *database.PostgresPool
}

func NewLegalNoticeRepository(db *database.PostgresPool) *LegalNoticeRepository {
	return &LegalNoticeRepository{db: db}
}

func (r *LegalNoticeRepository) Issue(ctx context.Context, notice *domain.LegalNotice) error {
	if notice.SubscriberID == 0 || notice.ChitID == 0 {
		row := r.db.QueryRow(ctx,
			`SELECT subscriber_id, chit_id FROM defaults WHERE default_id = $1`,
			notice.DefaultID,
		)
		_ = row.Scan(&notice.SubscriberID, &notice.ChitID)
	}

	var deadline interface{}
	if notice.ResponseDeadline != nil {
		deadline = notice.ResponseDeadline
	}

	// Map virtual fields to actual DB columns
	deliveryMode := notice.DeliveryMode
	if deliveryMode == nil && notice.NoticeSentVia != nil {
		deliveryMode = notice.NoticeSentVia
	}
	responseDetails := notice.ResponseDetails
	if responseDetails == nil && notice.NoticeContent != nil {
		responseDetails = notice.NoticeContent
	}

	query := `
		INSERT INTO legal_notices (
			default_id, subscriber_id, chit_id,
			notice_type, notice_date, delivery_mode,
			outstanding_amount, response_deadline, issued_by,
			response_details, status
		) VALUES ($1,$2,$3,$4,CURRENT_DATE,$5,0,$6,$7,$8,'SENT')
		RETURNING notice_id, created_at
	`
	return r.db.QueryRow(ctx, query,
		notice.DefaultID, notice.SubscriberID, notice.ChitID,
		notice.NoticeType, deliveryMode,
		deadline, notice.IssuedBy,
		responseDetails,
	).Scan(&notice.NoticeID, &notice.CreatedAt)
}

func (r *LegalNoticeRepository) GetByID(ctx context.Context, id, companyID int) (*domain.LegalNotice, error) {
	query := `
		SELECT ln.notice_id, ln.default_id, ln.subscriber_id, ln.chit_id,
			ln.notice_date, ln.notice_type, ln.notice_number, ln.outstanding_amount,
			ln.response_deadline, ln.delivery_mode, ln.tracking_number, ln.delivered_date,
			ln.response_received, ln.response_date, ln.response_details,
			ln.document_path, ln.issued_by, ln.status, ln.remarks, ln.created_at
		FROM legal_notices ln
		INNER JOIN chit_groups cg ON ln.chit_id = cg.chit_id
		WHERE ln.notice_id = $1 AND cg.company_id = $2
	`
	var n domain.LegalNotice
	err := r.db.QueryRow(ctx, query, id, companyID).Scan(
		&n.NoticeID, &n.DefaultID, &n.SubscriberID, &n.ChitID,
		&n.NoticeDate, &n.NoticeType, &n.NoticeNumber, &n.OutstandingAmount,
		&n.ResponseDeadline, &n.DeliveryMode, &n.TrackingNumber, &n.DeliveredDate,
		&n.ResponseReceived, &n.ResponseDate, &n.ResponseDetails,
		&n.DocumentPath, &n.IssuedBy, &n.Status, &n.Remarks, &n.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &n, err
}

func (r *LegalNoticeRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.LegalNotice, int64, error) {
	base := `
		SELECT ln.notice_id, ln.default_id, ln.subscriber_id, ln.chit_id,
			ln.notice_date, ln.notice_type, ln.notice_number, ln.outstanding_amount,
			ln.response_deadline, ln.delivery_mode, ln.tracking_number, ln.delivered_date,
			ln.response_received, ln.response_date, ln.response_details,
			ln.document_path, ln.issued_by, ln.status, ln.remarks, ln.created_at
		FROM legal_notices ln
		INNER JOIN chit_groups cg ON ln.chit_id = cg.chit_id`

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND ln.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM legal_notices ln INNER JOIN chit_groups cg ON ln.chit_id = cg.chit_id`+where,
		args...,
	).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY ln.notice_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notices []*domain.LegalNotice
	for rows.Next() {
		var n domain.LegalNotice
		if err := rows.Scan(
			&n.NoticeID, &n.DefaultID, &n.SubscriberID, &n.ChitID,
			&n.NoticeDate, &n.NoticeType, &n.NoticeNumber, &n.OutstandingAmount,
			&n.ResponseDeadline, &n.DeliveryMode, &n.TrackingNumber, &n.DeliveredDate,
			&n.ResponseReceived, &n.ResponseDate, &n.ResponseDetails,
			&n.DocumentPath, &n.IssuedBy, &n.Status, &n.Remarks, &n.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		notices = append(notices, &n)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return notices, total, nil
}

func (r *LegalNoticeRepository) Escalate(ctx context.Context, id, companyID int, req *domain.EscalateLegalNoticeRequest) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE legal_notices SET
			notice_type      = 'COURT_SUMMON',
			status           = 'ESCALATED',
			remarks          = CASE WHEN $1::text <> '' THEN $1::text ELSE remarks END
		WHERE notice_id = $2
		  AND EXISTS (SELECT 1 FROM chit_groups cg WHERE cg.chit_id = legal_notices.chit_id AND cg.company_id = $3)
	`, req.Remarks, id, companyID)
	_ = now
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
