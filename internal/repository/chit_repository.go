package repository

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type ChitGroupRepository struct {
	db *database.PostgresPool
}

func NewChitGroupRepository(db *database.PostgresPool) *ChitGroupRepository {
	return &ChitGroupRepository{db: db}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *ChitGroupRepository) Create(ctx context.Context, chit *domain.ChitGroup) error {
	query := `
		INSERT INTO chit_groups (
			company_id, branch_id, chit_code, chit_name, chit_value,
			total_subscribers, installment_amount, total_months, start_date,
			auction_day_of_month, auction_time, auction_type, installment_type,
			foreman_commission_percent, max_discount_percent, security_deposit,
			status, current_month, foreman_user_id, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20
		) RETURNING chit_id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		chit.CompanyID, chit.BranchID, chit.ChitCode, chit.ChitName, chit.ChitValue,
		chit.TotalSubscribers, chit.InstallmentAmount, chit.TotalMonths, chit.StartDate,
		chit.AuctionDayOfMonth, chit.AuctionTime, chit.AuctionType, chit.InstallmentType,
		chit.ForemanCommissionPercent, chit.MaxDiscountPercent, chit.SecurityDeposit,
		chit.Status, chit.CurrentMonth, chit.ForemanUserID, chit.CreatedBy,
	).Scan(&chit.ChitID, &chit.CreatedAt, &chit.UpdatedAt)
}

// ─── GetByID — full record, manual scan ───────────────────────────────────────

func (r *ChitGroupRepository) GetByID(ctx context.Context, id, companyID int) (*domain.ChitGroup, error) {
	query := `
		SELECT
			cg.chit_id, cg.company_id, cg.branch_id, cg.chit_code, cg.chit_name, cg.chit_value,
			cg.total_subscribers, cg.installment_amount, cg.total_months, cg.start_date, cg.end_date,
			cg.auction_day_of_month, cg.auction_time, cg.auction_type, cg.installment_type,
			cg.foreman_commission_percent, cg.max_discount_percent, cg.security_deposit,
			cg.status, cg.current_month, cg.foreman_user_id, cg.created_by,
			COALESCE(creator.full_name, '')  AS created_by_name,
			COALESCE(foreman.full_name, '')  AS foreman_name,
			cg.created_at, cg.updated_at
		FROM chit_groups cg
		LEFT JOIN users creator ON cg.created_by   = creator.user_id
		LEFT JOIN users foreman ON cg.foreman_user_id = foreman.user_id
		WHERE cg.chit_id = $1 AND cg.company_id = $2
	`
	var c domain.ChitGroup
	err := r.db.QueryRow(ctx, query, id, companyID).Scan(
		&c.ChitID, &c.CompanyID, &c.BranchID, &c.ChitCode, &c.ChitName, &c.ChitValue,
		&c.TotalSubscribers, &c.InstallmentAmount, &c.TotalMonths, &c.StartDate, &c.EndDate,
		&c.AuctionDayOfMonth, &c.AuctionTime, &c.AuctionType, &c.InstallmentType,
		&c.ForemanCommissionPercent, &c.MaxDiscountPercent, &c.SecurityDeposit,
		&c.Status, &c.CurrentMonth, &c.ForemanUserID, &c.CreatedBy,
		&c.CreatedByName, &c.ForemanName,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ─── List — manual scan matching the 14 selected columns exactly ──────────────

func (r *ChitGroupRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.ChitGroup, error) {
	base := `
		SELECT
			chit_id, company_id, branch_id, chit_code, chit_name, chit_value,
			total_subscribers, installment_amount, total_months, start_date,
			status, current_month, created_at, updated_at
		FROM chit_groups
		WHERE company_id = $1`

	args := []interface{}{companyID}
	argPos := 2

	if status, ok := filters["status"].(string); ok && status != "" {
		base += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if branchID, ok := filters["branch_id"].(int); ok && branchID > 0 {
		base += fmt.Sprintf(" AND branch_id = $%d", argPos)
		args = append(args, branchID)
		argPos++
	}

	base += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		base += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
		argPos++
	}
	if offset, ok := filters["offset"].(int); ok && offset >= 0 {
		base += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, offset)
	}

	rows, err := r.db.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chits := []*domain.ChitGroup{}
	for rows.Next() {
		var c domain.ChitGroup
		if err := rows.Scan(
			&c.ChitID, &c.CompanyID, &c.BranchID, &c.ChitCode, &c.ChitName, &c.ChitValue,
			&c.TotalSubscribers, &c.InstallmentAmount, &c.TotalMonths, &c.StartDate,
			&c.Status, &c.CurrentMonth, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chits = append(chits, &c)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return chits, nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *ChitGroupRepository) Update(ctx context.Context, chit *domain.ChitGroup) error {
	// Use COALESCE so only non-empty values overwrite existing DB values.
	// This prevents a stale in-memory status from overwriting a DB-updated status.
	var statusPtr *string
	if chit.Status != "" {
		statusPtr = &chit.Status
	}
	var namePtr *string
	if chit.ChitName != "" {
		namePtr = &chit.ChitName
	}
	var monthPtr *int
	if chit.CurrentMonth >= 0 {
		monthPtr = &chit.CurrentMonth
	}

	query := `
		UPDATE chit_groups SET
			chit_name     = COALESCE($1, chit_name),
			status        = COALESCE($2, status),
			current_month = COALESCE($3, current_month),
			updated_at    = NOW()
		WHERE chit_id = $4 AND company_id = $5
	`
	result, err := r.db.Exec(ctx, query,
		namePtr, statusPtr, monthPtr, chit.ChitID, chit.CompanyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ─── AddSubscriber ────────────────────────────────────────────────────────────

func (r *ChitGroupRepository) AddSubscriber(ctx context.Context, cs *domain.ChitSubscriber) error {
	// Auto-assign slot_number = MAX(slot_number) + 1 for this chit group.
	// If no subscribers yet, slot starts at 1.
	query := `
		INSERT INTO chit_subscribers (
			chit_id, subscriber_id, join_date, slot_number, ticket_count,
			surety_required, status, enrollment_fee
		) VALUES (
			$1, $2, $3,
			COALESCE((SELECT MAX(slot_number) FROM chit_subscribers WHERE chit_id = $1), 0) + 1,
			$4, $5, $6, $7
		)
		RETURNING chit_subscriber_id, slot_number, created_at
	`
	return r.db.QueryRow(ctx, query,
		cs.ChitID, cs.SubscriberID, cs.JoinDate,
		cs.TicketCount, cs.SuretyRequired, cs.Status, cs.EnrollmentFee,
	).Scan(&cs.ChitSubscriberID, &cs.SlotNumber, &cs.CreatedAt)
}

// ─── GetSubscribers — manual scan matching 13 selected columns ───────────────

func (r *ChitGroupRepository) GetSubscribers(ctx context.Context, chitID, companyID int) ([]*domain.ChitSubscriber, error) {
	query := `
		SELECT
			cs.chit_subscriber_id, cs.chit_id, cg.chit_name,
			cs.subscriber_id, s.full_name AS subscriber_name,
			cs.join_date, cs.slot_number, cs.ticket_count,
			cs.is_prized, cs.prized_month, cs.prized_date,
			cs.surety_required, cs.status, cs.enrollment_fee,
			cs.created_at, cs.updated_at
		FROM chit_subscribers cs
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		INNER JOIN subscribers s ON cs.subscriber_id = s.subscriber_id
		WHERE cs.chit_id = $1 AND cg.company_id = $2 AND cs.status = 'ACTIVE'
		ORDER BY cs.slot_number
	`
	rows, err := r.db.Query(ctx, query, chitID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscribers := []*domain.ChitSubscriber{}
	for rows.Next() {
		var cs domain.ChitSubscriber
		if err := rows.Scan(
			&cs.ChitSubscriberID, &cs.ChitID, &cs.ChitName,
			&cs.SubscriberID, &cs.SubscriberName,
			&cs.JoinDate, &cs.SlotNumber, &cs.TicketCount,
			&cs.IsPrized, &cs.PrizedMonth, &cs.PrizedDate,
			&cs.SuretyRequired, &cs.Status, &cs.EnrollmentFee,
			&cs.CreatedAt, &cs.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subscribers = append(subscribers, &cs)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return subscribers, nil
}

// ─── RemoveEnrollment ─────────────────────────────────────────────────────────

func (r *ChitGroupRepository) RemoveEnrollment(ctx context.Context, csID, chitID int) error {
	query := `
		UPDATE chit_subscribers
		SET status = 'DROPPED', dropped_date = CURRENT_DATE, updated_at = NOW()
		WHERE chit_subscriber_id = $1 AND chit_id = $2 AND status = 'ACTIVE'
	`
	result, err := r.db.Exec(ctx, query, csID, chitID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
