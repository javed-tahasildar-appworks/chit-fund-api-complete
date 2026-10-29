package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"
)

type DashboardRepository struct {
	db *database.PostgresPool
}

func NewDashboardRepository(db *database.PostgresPool) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// ─── KPIs ─────────────────────────────────────────────────────────────────────

func (r *DashboardRepository) GetKPIs(ctx context.Context, companyID int) (*domain.DashboardKPIs, error) {
	kpis := &domain.DashboardKPIs{}

	// Total collection this month in lakhs (1 lakh = 100,000)
	r.db.QueryRow(ctx, `
		SELECT COALESCE(ROUND(SUM(pt.amount)/100000.0, 2), 0)
		FROM payment_transactions pt
		INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
		WHERE s.company_id = $1
		  AND DATE_TRUNC('month', pt.transaction_date) = DATE_TRUNC('month', NOW())
		  AND pt.status = 'COMPLETED'
	`, companyID).Scan(&kpis.TotalCollectionLakhs)

	// Collection change % vs last month
	var lastMonthCollection float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(pt.amount)/100000.0, 0)
		FROM payment_transactions pt
		INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
		WHERE s.company_id = $1
		  AND DATE_TRUNC('month', pt.transaction_date) = DATE_TRUNC('month', NOW() - INTERVAL '1 month')
		  AND pt.status = 'COMPLETED'
	`, companyID).Scan(&lastMonthCollection)

	if lastMonthCollection > 0 {
		kpis.CollectionChangePercent = ((kpis.TotalCollectionLakhs - lastMonthCollection) / lastMonthCollection) * 100
		kpis.CollectionChangePercent = float64(int(kpis.CollectionChangePercent*100)) / 100
	}

	// Active chit groups
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM chit_groups WHERE company_id = $1 AND status = 'ACTIVE'`,
		companyID).Scan(&kpis.ActiveChitGroups)

	// New chit groups created this month
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM chit_groups
		WHERE company_id = $1
		  AND DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW())
	`, companyID).Scan(&kpis.NewChitGroupsThisMonth)

	// Pending defaults
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM defaults WHERE company_id = $1 AND status = 'ACTIVE'`,
		companyID).Scan(&kpis.PendingDefaults)

	// Escalated to legal
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM legal_notices WHERE company_id = $1 AND notice_type = 'COURT_SUMMON' AND status != 'CLOSED'`,
		companyID).Scan(&kpis.EscalatedToLegal)

	// Auctions this month (all statuses)
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM auctions a
		INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
		WHERE cg.company_id = $1
		  AND DATE_TRUNC('month', a.auction_date) = DATE_TRUNC('month', NOW())
	`, companyID).Scan(&kpis.AuctionsThisMonth)

	// Upcoming auctions (scheduled, future date)
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM auctions a
		INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
		WHERE cg.company_id = $1 AND a.status = 'SCHEDULED' AND a.auction_date >= NOW()
	`, companyID).Scan(&kpis.UpcomingAuctions)

	// Completed auctions this month
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM auctions a
		INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
		WHERE cg.company_id = $1
		  AND a.status = 'COMPLETED'
		  AND DATE_TRUNC('month', a.auction_date) = DATE_TRUNC('month', NOW())
	`, companyID).Scan(&kpis.CompletedAuctions)

	return kpis, nil
}

// ─── Monthly Collections ──────────────────────────────────────────────────────

func (r *DashboardRepository) GetMonthlyCollections(ctx context.Context, companyID int) ([]domain.MonthlyCollectionItem, error) {
	query := `
		WITH months AS (
			SELECT generate_series(
				DATE_TRUNC('month', NOW() - INTERVAL '11 months'),
				DATE_TRUNC('month', NOW() + INTERVAL '2 months'),
				'1 month'::interval
			) AS month_start
		),
		collected AS (
			SELECT
				DATE_TRUNC('month', pt.transaction_date) AS month_start,
				ROUND(SUM(pt.amount) / 100000.0, 2)      AS collected_l
			FROM payment_transactions pt
			INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
			WHERE s.company_id = $1 AND pt.status = 'COMPLETED'
			GROUP BY 1
		),
		targets AS (
			SELECT
				DATE_TRUNC('month', i.due_date) AS month_start,
				ROUND(SUM(i.net_payable_amount * cg.total_subscribers) / 100000.0, 2) AS target_l
			FROM installments i
			INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
			WHERE cg.company_id = $1
			GROUP BY 1
		)
		SELECT
			TO_CHAR(m.month_start, 'Mon')       AS month,
			COALESCE(c.collected_l, 0)           AS collected_l,
			COALESCE(t.target_l, 0)              AS target_l,
			m.month_start > DATE_TRUNC('month', NOW()) AS is_future
		FROM months m
		LEFT JOIN collected c ON c.month_start = m.month_start
		LEFT JOIN targets  t ON t.month_start  = m.month_start
		ORDER BY m.month_start
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.MonthlyCollectionItem{}
	for rows.Next() {
		var item domain.MonthlyCollectionItem
		if err := rows.Scan(&item.Month, &item.CollectedL, &item.TargetL, &item.IsFuture); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// ─── Subscriber Stats ─────────────────────────────────────────────────────────

func (r *DashboardRepository) GetSubscriberStats(ctx context.Context, companyID int) (*domain.SubscriberStats, error) {
	stats := &domain.SubscriberStats{}

	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscribers WHERE company_id = $1`,
		companyID).Scan(&stats.Total)

	r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT cs.subscriber_id)
		FROM chit_subscribers cs
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		WHERE cg.company_id = $1 AND cs.status = 'ACTIVE'
	`, companyID).Scan(&stats.Active)

	r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT cs.subscriber_id)
		FROM chit_subscribers cs
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		WHERE cg.company_id = $1 AND cs.is_prized = true
	`, companyID).Scan(&stats.Prized)

	r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT subscriber_id) FROM defaults WHERE company_id = $1 AND status = 'ACTIVE'`,
		companyID).Scan(&stats.Defaulters)

	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscribers WHERE company_id = $1 AND kyc_status = 'PENDING'`,
		companyID).Scan(&stats.KYCPending)

	return stats, nil
}

// ─── Upcoming Auctions ────────────────────────────────────────────────────────

func (r *DashboardRepository) GetUpcomingAuctions(ctx context.Context, companyID int) ([]domain.UpcomingAuction, error) {
	query := `
		SELECT
			'AUC' || LPAD(a.auction_id::text, 3, '0')  AS id,
			cg.chit_name                                 AS group_name,
			ROUND(cg.chit_value / 100000.0, 2)          AS chit_value_lakhs,
			a.auction_date,
			cg.total_subscribers                         AS member_count,
			COALESCE(a.auction_method, cg.auction_type)  AS method,
			COALESCE(b.branch_name, 'Head Office')       AS branch_name
		FROM auctions a
		INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
		LEFT  JOIN branches b    ON cg.branch_id = b.branch_id
		WHERE cg.company_id = $1
		  AND a.status = 'SCHEDULED'
		  AND a.auction_date >= NOW()
		ORDER BY a.auction_date ASC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	auctions := []domain.UpcomingAuction{}
	for rows.Next() {
		var ua domain.UpcomingAuction
		if err := rows.Scan(
			&ua.ID, &ua.GroupName, &ua.ChitValueLakhs,
			&ua.AuctionDate, &ua.MemberCount, &ua.Method, &ua.BranchName,
		); err != nil {
			return nil, err
		}
		auctions = append(auctions, ua)
	}
	return auctions, nil
}

// ─── Branch Collections ───────────────────────────────────────────────────────

func (r *DashboardRepository) GetBranchCollections(ctx context.Context, companyID int) ([]domain.BranchCollection, error) {
	query := `
		SELECT
			'BR' || LPAD(b.branch_id::text, 3, '0')    AS branch_id,
			b.branch_name,
			COALESCE(ROUND(SUM(pt.amount) / 100000.0, 2), 0) AS collected_l,
			COALESCE(ROUND(SUM(DISTINCT i.net_payable_amount * cg.total_subscribers) / 100000.0, 2), 0) AS target_l
		FROM branches b
		LEFT JOIN chit_groups cg  ON cg.branch_id  = b.branch_id
		LEFT JOIN installments i  ON i.chit_id      = cg.chit_id
			AND DATE_TRUNC('month', i.due_date) = DATE_TRUNC('month', NOW())
		LEFT JOIN chit_subscribers cs ON cs.chit_id  = cg.chit_id AND cs.status = 'ACTIVE'
		LEFT JOIN subscriber_installments si ON si.chit_subscriber_id = cs.chit_subscriber_id
			AND si.installment_id = i.installment_id
		LEFT JOIN payment_transactions pt ON pt.subscriber_installment_id = si.subscriber_installment_id
			AND pt.status = 'COMPLETED'
		WHERE b.company_id = $1
		GROUP BY b.branch_id, b.branch_name
		ORDER BY collected_l DESC
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	branches := []domain.BranchCollection{}
	for rows.Next() {
		var bc domain.BranchCollection
		if err := rows.Scan(&bc.BranchID, &bc.BranchName, &bc.CollectedL, &bc.TargetL); err != nil {
			return nil, err
		}
		branches = append(branches, bc)
	}
	return branches, nil
}

// ─── Recent Activity ──────────────────────────────────────────────────────────

func (r *DashboardRepository) GetRecentActivity(ctx context.Context, companyID int) ([]domain.RecentActivity, error) {
	query := `
		(
			SELECT
				'ACT-PAY-' || pt.transaction_id::text   AS id,
				'payment'                                AS type,
				'Installment Collected'                  AS title,
				'Subscriber: ' || s.full_name || ' - ₹' || TO_CHAR(pt.amount, 'FM99,99,999') AS subtitle,
				pt.transaction_date                      AS timestamp
			FROM payment_transactions pt
			INNER JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
			WHERE s.company_id = $1 AND pt.status = 'COMPLETED'
			ORDER BY pt.transaction_date DESC
			LIMIT 5
		)
		UNION ALL
		(
			SELECT
				'ACT-AUC-' || a.auction_id::text        AS id,
				'auction'                                AS type,
				CASE a.status
					WHEN 'COMPLETED'   THEN 'Auction Completed'
					WHEN 'SCHEDULED'   THEN 'Auction Scheduled'
					WHEN 'IN_PROGRESS' THEN 'Auction In Progress'
					ELSE 'Auction Updated'
				END                                      AS title,
				cg.chit_name || COALESCE(
					' - Winner: ' || (
						SELECT s2.full_name FROM auction_bids ab
						INNER JOIN subscribers s2 ON ab.subscriber_id = s2.subscriber_id
						WHERE ab.auction_id = a.auction_id AND ab.is_winner = true
						LIMIT 1
					), ''
				)                                        AS subtitle,
				a.updated_at                             AS timestamp
			FROM auctions a
			INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
			WHERE cg.company_id = $1
			ORDER BY a.updated_at DESC
			LIMIT 5
		)
		ORDER BY timestamp DESC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []domain.RecentActivity{}
	for rows.Next() {
		var ra domain.RecentActivity
		if err := rows.Scan(&ra.ID, &ra.Type, &ra.Title, &ra.Subtitle, &ra.Timestamp); err != nil {
			return nil, err
		}
		activities = append(activities, ra)
	}
	return activities, nil
}

// ─── Reports (used by report handler) ────────────────────────────────────────

func (r *DashboardRepository) GetDefaultersReport(ctx context.Context, companyID int, _ map[string]interface{}) ([]*domain.DefaulterReport, error) {
	query := `
		SELECT DISTINCT s.subscriber_id, s.subscriber_code, s.full_name, s.mobile,
			cg.chit_code, cg.chit_name,
			d.days_overdue,
			d.total_outstanding,
			TO_CHAR(d.last_reminder_date, 'YYYY-MM-DD'),
			d.status
		FROM defaults d
		INNER JOIN subscribers s  ON d.subscriber_id = s.subscriber_id
		INNER JOIN chit_groups cg ON d.chit_id = cg.chit_id
		WHERE cg.company_id = $1 AND d.status = 'ACTIVE'
		ORDER BY d.total_outstanding DESC
		LIMIT 100
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []*domain.DefaulterReport{}
	for rows.Next() {
		var rep domain.DefaulterReport
		if err := rows.Scan(
			&rep.SubscriberID, &rep.SubscriberCode, &rep.FullName, &rep.Mobile,
			&rep.ChitCode, &rep.ChitName, &rep.OverdueMonths,
			&rep.TotalOverdue, &rep.LastPaymentDate, &rep.DefaultStatus,
		); err != nil {
			return nil, err
		}
		reports = append(reports, &rep)
	}
	return reports, nil
}

func (r *DashboardRepository) GetChitPerformance(ctx context.Context, companyID int) ([]*domain.ChitPerformanceReport, error) {
	query := `
		SELECT cg.chit_id, cg.chit_code, cg.chit_name, cg.chit_value,
			cg.total_subscribers, cg.current_month, cg.total_months,
			COALESCE(SUM(si.amount_paid), 0),
			COALESCE(SUM(si.balance_amount), 0),
			COUNT(DISTINCT d.default_id) FILTER (WHERE d.status = 'ACTIVE'),
			cg.status
		FROM chit_groups cg
		LEFT JOIN chit_subscribers cs ON cs.chit_id = cg.chit_id
		LEFT JOIN subscriber_installments si ON si.chit_subscriber_id = cs.chit_subscriber_id
		LEFT JOIN defaults d ON d.chit_id = cg.chit_id
		WHERE cg.company_id = $1
		GROUP BY cg.chit_id, cg.chit_code, cg.chit_name, cg.chit_value,
			cg.total_subscribers, cg.current_month, cg.total_months, cg.status
		ORDER BY cg.status, cg.chit_name
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []*domain.ChitPerformanceReport{}
	for rows.Next() {
		var rep domain.ChitPerformanceReport
		if err := rows.Scan(
			&rep.ChitID, &rep.ChitCode, &rep.ChitName, &rep.ChitValue,
			&rep.TotalSubscribers, &rep.CurrentMonth, &rep.TotalMonths,
			&rep.TotalCollected, &rep.TotalPending, &rep.DefaultersCount, &rep.Status,
		); err != nil {
			return nil, err
		}
		reports = append(reports, &rep)
	}
	return reports, nil
}

func (r *DashboardRepository) GetSubscriberLedger(ctx context.Context, subscriberID, companyID int) ([]*domain.SubscriberLedgerEntry, error) {
	query := `
		SELECT
			TO_CHAR(COALESCE(si.payment_date, i.due_date), 'YYYY-MM-DD'),
			cg.chit_code,
			CASE si.status
				WHEN 'PAID'    THEN 'Installment paid - Month '    || i.month_number
				WHEN 'PARTIAL' THEN 'Partial payment - Month '     || i.month_number
				ELSE                'Installment due - Month '     || i.month_number
			END,
			i.month_number,
			si.amount_due, si.amount_paid, si.balance_amount,
			si.payment_mode, si.receipt_number, si.status
		FROM subscriber_installments si
		INNER JOIN installments i  ON si.installment_id = i.installment_id
		INNER JOIN chit_groups cg  ON i.chit_id = cg.chit_id
		WHERE si.subscriber_id = $1 AND cg.company_id = $2
		ORDER BY i.due_date DESC
		LIMIT 200
	`
	rows, err := r.db.Query(ctx, query, subscriberID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []*domain.SubscriberLedgerEntry{}
	for rows.Next() {
		var e domain.SubscriberLedgerEntry
		if err := rows.Scan(
			&e.Date, &e.ChitCode, &e.Description, &e.MonthNumber,
			&e.AmountDue, &e.AmountPaid, &e.Balance,
			&e.PaymentMode, &e.ReceiptNumber, &e.Status,
		); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, nil
}

func (r *DashboardRepository) GetMonthlyCollectionSummary(ctx context.Context, companyID int) ([]domain.MonthlyCollectionItem, error) {
	return r.GetMonthlyCollections(ctx, companyID)
}

// ─── Audit Logs ───────────────────────────────────────────────────────────────

type AuditLogRepository struct {
	db *database.PostgresPool
}

func NewAuditLogRepository(db *database.PostgresPool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO audit_logs (user_id, subscriber_id, action, entity_name, entity_id, old_values, new_values, ip_address, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING log_id, timestamp
	`, log.UserID, log.SubscriberID, log.Action, log.EntityName, log.EntityID,
		log.OldValues, log.NewValues, log.IPAddress, log.UserAgent,
	).Scan(&log.LogID, &log.Timestamp)
}

func (r *AuditLogRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.AuditLog, int64, error) {
	base := `SELECT log_id, user_id, subscriber_id, action, entity_name, entity_id,
		old_values, new_values, CAST(ip_address AS text), user_agent, timestamp
		FROM audit_logs`

	args := []interface{}{}
	where := " WHERE 1=1"
	argPos := 1

	if t, ok := filters["table_name"].(string); ok && t != "" {
		where += fmt.Sprintf(" AND entity_name = $%d", argPos)
		args = append(args, t)
		argPos++
	}
	if u, ok := filters["user_id"].(int); ok && u > 0 {
		where += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, u)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`+where, args...).Scan(&total)

	limit := 50
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(
			&l.LogID, &l.UserID, &l.SubscriberID, &l.Action, &l.EntityName, &l.EntityID,
			&l.OldValues, &l.NewValues, &l.IPAddress, &l.UserAgent, &l.Timestamp,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return logs, total, nil
}

func (r *AuditLogRepository) ListSystemLogs(ctx context.Context, _ int, filters map[string]interface{}) ([]*domain.SystemLog, int64, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if level, ok := filters["level"].(string); ok && level != "" {
		where += fmt.Sprintf(" AND log_level = $%d", argPos)
		args = append(args, level)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM system_logs`+where, args...).Scan(&total)

	limit := 50
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := `SELECT log_id, log_level, module, message, stack_trace, user_id,
		CAST(ip_address AS text), created_at FROM system_logs` +
		where + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sysLogs []*domain.SystemLog
	for rows.Next() {
		var l domain.SystemLog
		if err := rows.Scan(
			&l.LogID, &l.LogLevel, &l.Module, &l.Message,
			&l.StackTrace, &l.UserID, &l.IPAddress, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		sysLogs = append(sysLogs, &l)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return sysLogs, total, nil
}

// keep unused import happy
var _ = time.Now
