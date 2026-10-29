package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type NotificationRepository struct {
	db *database.PostgresPool
}

func NewNotificationRepository(db *database.PostgresPool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// ── Visibility predicate ──────────────────────────────────────────────────────
// A notification is visible to (companyID, userID, roleName, branchID) if:
//   company matches AND (user_id matches OR role matches OR branch_id matches
//   OR all three are NULL — broadcast)

const visibilityWhere = `
	n.company_id = $1
	AND (
		n.user_id   = $2
		OR n.role   = $3
		OR n.branch_id = $4
		OR (n.user_id IS NULL AND n.role IS NULL AND n.branch_id IS NULL)
	)
	AND (nr.dismissed_at IS NULL OR nr.dismissed_at IS NOT NULL AND false)
`

// buildVisArgs returns [companyID, userID, roleName, branchID, ...]
func buildVisArgs(companyID, userID int, roleName string, branchID *int) []interface{} {
	bid := 0
	if branchID != nil {
		bid = *branchID
	}
	return []interface{}{companyID, userID, roleName, bid}
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *NotificationRepository) List(
	ctx context.Context,
	companyID, userID int,
	roleName string,
	branchID *int,
	f domain.NotificationListFilters,
) ([]*domain.StaffNotification, int64, error) {

	args := buildVisArgs(companyID, userID, roleName, branchID)
	argPos := 5 // next placeholder after visibility args

	// Base query with LEFT JOIN to get per-user read state
	base := `
		SELECT n.notification_id, n.notification_uid, n.company_id,
			n.type, n.severity, n.title, n.body,
			CASE WHEN nr.read_at IS NOT NULL THEN true ELSE false END AS is_read,
			n.action_route, n.related_entity_type, n.related_entity_id,
			n.branch_id, n.created_at, nr.read_at
		FROM staff_notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.notification_id AND nr.user_id = $2`

	where := fmt.Sprintf(`
		WHERE n.company_id = $1
		AND (
			n.user_id   = $2
			OR n.role   = $3
			OR n.branch_id = $4
			OR (n.user_id IS NULL AND n.role IS NULL AND n.branch_id IS NULL)
		)
		AND (nr.dismissed_at IS NULL)`)

	switch f.Status {
	case "unread":
		where += " AND nr.read_at IS NULL"
	case "read":
		where += " AND nr.read_at IS NOT NULL"
	}
	if f.Type != "" {
		where += fmt.Sprintf(" AND n.type = $%d", argPos)
		args = append(args, f.Type)
		argPos++
	}

	// Count
	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM staff_notifications n
		 LEFT JOIN notification_reads nr ON nr.notification_id = n.notification_id AND nr.user_id = $2`+
			where, args...).Scan(&total)

	// Paginate
	limit, offset := 20, 0
	if f.Limit > 0 && f.Limit <= 100 {
		limit = f.Limit
	}
	if f.Offset > 0 {
		offset = f.Offset
	}
	query := base + where +
		fmt.Sprintf(" ORDER BY n.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifs []*domain.StaffNotification
	for rows.Next() {
		var n domain.StaffNotification
		if err := rows.Scan(
			&n.NotificationID, &n.NotificationUID, &n.CompanyID,
			&n.Type, &n.Severity, &n.Title, &n.Body,
			&n.IsRead,
			&n.ActionRoute, &n.RelatedEntityType, &n.RelatedEntityID,
			&n.BranchID, &n.CreatedAt, &n.ReadAt,
		); err != nil {
			return nil, 0, err
		}
		notifs = append(notifs, &n)
	}
	if notifs == nil {
		notifs = []*domain.StaffNotification{}
	}
	return notifs, total, rows.Err()
}

// ── Unread count (badge) ──────────────────────────────────────────────────────

func (r *NotificationRepository) UnreadCount(
	ctx context.Context,
	companyID, userID int,
	roleName string,
	branchID *int,
) (int, error) {
	args := buildVisArgs(companyID, userID, roleName, branchID)
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM staff_notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.notification_id AND nr.user_id = $2
		WHERE n.company_id = $1
		  AND (
			n.user_id    = $2
			OR n.role    = $3
			OR n.branch_id = $4
			OR (n.user_id IS NULL AND n.role IS NULL AND n.branch_id IS NULL)
		  )
		  AND nr.read_at IS NULL
		  AND (nr.dismissed_at IS NULL)
	`, args...).Scan(&count)
	return count, err
}

// ── Get single (used after mark-read to return updated object) ────────────────

func (r *NotificationRepository) GetByID(
	ctx context.Context,
	notificationID int64,
	companyID, userID int,
	roleName string,
	branchID *int,
) (*domain.StaffNotification, error) {
	args := buildVisArgs(companyID, userID, roleName, branchID)
	args = append(args, notificationID) // $5

	var n domain.StaffNotification
	err := r.db.QueryRow(ctx, `
		SELECT n.notification_id, n.notification_uid, n.company_id,
			n.type, n.severity, n.title, n.body,
			CASE WHEN nr.read_at IS NOT NULL THEN true ELSE false END AS is_read,
			n.action_route, n.related_entity_type, n.related_entity_id,
			n.branch_id, n.created_at, nr.read_at
		FROM staff_notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.notification_id AND nr.user_id = $2
		WHERE n.notification_id = $5
		  AND n.company_id = $1
		  AND (
			n.user_id   = $2
			OR n.role   = $3
			OR n.branch_id = $4
			OR (n.user_id IS NULL AND n.role IS NULL AND n.branch_id IS NULL)
		  )
	`, args...).Scan(
		&n.NotificationID, &n.NotificationUID, &n.CompanyID,
		&n.Type, &n.Severity, &n.Title, &n.Body,
		&n.IsRead,
		&n.ActionRoute, &n.RelatedEntityType, &n.RelatedEntityID,
		&n.BranchID, &n.CreatedAt, &n.ReadAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &n, err
}

// ── Mark one read ─────────────────────────────────────────────────────────────

func (r *NotificationRepository) MarkRead(
	ctx context.Context,
	notificationID int64,
	companyID, userID int,
	roleName string,
	branchID *int,
) error {
	// First verify visibility
	_, err := r.GetByID(ctx, notificationID, companyID, userID, roleName, branchID)
	if err != nil {
		return err
	}

	// Upsert into notification_reads (idempotent)
	_, err = r.db.Exec(ctx, `
		INSERT INTO notification_reads (notification_id, user_id, read_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (notification_id, user_id)
		DO UPDATE SET read_at = COALESCE(notification_reads.read_at, EXCLUDED.read_at)
	`, notificationID, userID)
	return err
}

// ── Mark all read ─────────────────────────────────────────────────────────────

func (r *NotificationRepository) MarkAllRead(
	ctx context.Context,
	companyID, userID int,
	roleName string,
	branchID *int,
	notifType string, // empty = all types
) (int, error) {
	args := buildVisArgs(companyID, userID, roleName, branchID)
	argPos := 5
	typeFilter := ""
	if notifType != "" {
		typeFilter = fmt.Sprintf(" AND n.type = $%d", argPos)
		args = append(args, notifType)
	}

	// Get all visible unread notification IDs for this user
	rows, err := r.db.Query(ctx, `
		SELECT n.notification_id
		FROM staff_notifications n
		LEFT JOIN notification_reads nr
			ON nr.notification_id = n.notification_id AND nr.user_id = $2
		WHERE n.company_id = $1
		  AND (
			n.user_id   = $2 OR n.role = $3 OR n.branch_id = $4
			OR (n.user_id IS NULL AND n.role IS NULL AND n.branch_id IS NULL)
		  )
		  AND nr.read_at IS NULL
		`+typeFilter, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	// Bulk upsert
	now := time.Now()
	count := 0
	for _, nid := range ids {
		_, err := r.db.Exec(ctx, `
			INSERT INTO notification_reads (notification_id, user_id, read_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (notification_id, user_id)
			DO UPDATE SET read_at = COALESCE(notification_reads.read_at, EXCLUDED.read_at)
		`, nid, userID, now)
		if err == nil {
			count++
		}
	}
	return count, nil
}

// ── Dismiss ───────────────────────────────────────────────────────────────────

func (r *NotificationRepository) Dismiss(
	ctx context.Context,
	notificationID int64,
	companyID, userID int,
	roleName string,
	branchID *int,
) error {
	// Verify visibility first
	_, err := r.GetByID(ctx, notificationID, companyID, userID, roleName, branchID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO notification_reads (notification_id, user_id, read_at, dismissed_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (notification_id, user_id)
		DO UPDATE SET dismissed_at = NOW()
	`, notificationID, userID)
	return err
}

// ── Create (internal — domain events, not HTTP) ───────────────────────────────

func (r *NotificationRepository) Create(
	ctx context.Context,
	req *domain.CreateStaffNotificationRequest,
) error {
	severity := req.Severity
	if severity == "" {
		if s, ok := domain.NotifDefaultSeverity[req.Type]; ok {
			severity = s
		} else {
			severity = "INFO"
		}
	}
	actionRoute := req.ActionRoute
	if actionRoute == "" {
		if a, ok := domain.NotifDefaultActionRoute[req.Type]; ok {
			actionRoute = a
		}
	}

	var entityType interface{} = nil
	if req.RelatedEntityType != "" {
		entityType = req.RelatedEntityType
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO staff_notifications (
			company_id, user_id, role, branch_id,
			type, severity, title, body,
			action_route, related_entity_type, related_entity_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`,
		req.CompanyID, req.UserID, req.Role, req.BranchID,
		req.Type, severity, req.Title, req.Body,
		actionRoute, entityType, req.RelatedEntityID,
	)
	return err
}
