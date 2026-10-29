package repository

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type MobileRepository struct {
	db *database.PostgresPool
}

func NewMobileRepository(db *database.PostgresPool) *MobileRepository {
	return &MobileRepository{db: db}
}

func (r *MobileRepository) DB() *database.PostgresPool { return r.db }

// ── App User (Subscriber) ─────────────────────────────────────────────────────

func (r *MobileRepository) GetAppUserByMobile(ctx context.Context, mobile string) (*domain.AppUser, error) {
	var u domain.AppUser
	err := r.db.QueryRow(ctx, `
		SELECT app_user_id, subscriber_id, mobile, email, password_hash,
		       device_token, device_type, app_version, is_active, is_verified,
		       otp_code, otp_expiry, last_login, created_at, updated_at
		FROM app_users WHERE mobile = $1
	`, mobile).Scan(
		&u.AppUserID, &u.SubscriberID, &u.Mobile, &u.Email, &u.PasswordHash,
		&u.DeviceToken, &u.DeviceType, &u.AppVersion, &u.IsActive, &u.IsVerified,
		&u.OTPCode, &u.OTPExpiry, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *MobileRepository) GetAppUserByID(ctx context.Context, appUserID int) (*domain.AppUser, error) {
	var u domain.AppUser
	err := r.db.QueryRow(ctx, `
		SELECT app_user_id, subscriber_id, mobile, email, password_hash,
		       device_token, device_type, app_version, is_active, is_verified,
		       otp_code, otp_expiry, last_login, created_at, updated_at
		FROM app_users WHERE app_user_id = $1
	`, appUserID).Scan(
		&u.AppUserID, &u.SubscriberID, &u.Mobile, &u.Email, &u.PasswordHash,
		&u.DeviceToken, &u.DeviceType, &u.AppVersion, &u.IsActive, &u.IsVerified,
		&u.OTPCode, &u.OTPExpiry, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

// GetSubscriberByMobile checks if a subscriber with this mobile is registered in the company
func (r *MobileRepository) GetSubscriberByMobile(ctx context.Context, mobile string) (*domain.AppSubscriberProfile, error) {
	var p domain.AppSubscriberProfile
	err := r.db.QueryRow(ctx, `
		SELECT subscriber_id, subscriber_code, full_name, mobile, email, kyc_status
		FROM subscribers
		WHERE mobile = $1
	`, mobile).Scan(
		&p.SubscriberID, &p.SubscriberCode, &p.FullName, &p.Mobile, &p.Email, &p.KYCStatus,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *MobileRepository) CreateAppUser(ctx context.Context, u *domain.AppUser) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO app_users
			(subscriber_id, mobile, email, password_hash, device_token,
			 device_type, app_version, is_active, is_verified)
		VALUES ($1,$2,$3,$4,$5,$6,$7,true,false)
		RETURNING app_user_id, created_at, updated_at
	`, u.SubscriberID, u.Mobile, u.Email, u.PasswordHash,
		u.DeviceToken, u.DeviceType, u.AppVersion,
	).Scan(&u.AppUserID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *MobileRepository) SetOTP(ctx context.Context, appUserID int, otpCode string, expiry time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE app_users SET otp_code = $1, otp_expiry = $2, updated_at = NOW()
		WHERE app_user_id = $3
	`, otpCode, expiry, appUserID)
	return err
}

func (r *MobileRepository) VerifyOTP(ctx context.Context, mobile, otp string) (*domain.AppUser, error) {
	var u domain.AppUser
	err := r.db.QueryRow(ctx, `
		SELECT app_user_id, subscriber_id, mobile, email, password_hash,
		       is_active, is_verified, otp_code, otp_expiry
		FROM app_users
		WHERE mobile = $1 AND otp_code = $2 AND otp_expiry > NOW()
	`, mobile, otp).Scan(
		&u.AppUserID, &u.SubscriberID, &u.Mobile, &u.Email, &u.PasswordHash,
		&u.IsActive, &u.IsVerified, &u.OTPCode, &u.OTPExpiry,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.NewAppError("OTP_INVALID", "OTP is invalid or expired", nil)
	}
	return &u, err
}

func (r *MobileRepository) MarkVerified(ctx context.Context, appUserID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE app_users SET is_verified = true, otp_code = NULL,
		       otp_expiry = NULL, updated_at = NOW()
		WHERE app_user_id = $1
	`, appUserID)
	return err
}

func (r *MobileRepository) UpdateLastLogin(ctx context.Context, appUserID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE app_users SET last_login = NOW(), updated_at = NOW()
		WHERE app_user_id = $1
	`, appUserID)
	return err
}

func (r *MobileRepository) UpdateDevice(ctx context.Context, appUserID int, deviceToken, deviceType, appVersion string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE app_users
		SET device_token = $1, device_type = $2, app_version = $3, updated_at = NOW()
		WHERE app_user_id = $4
	`, deviceToken, deviceType, appVersion, appUserID)
	return err
}

// ── Subscriber App — read data ────────────────────────────────────────────────

func (r *MobileRepository) GetSubscriberProfile(ctx context.Context, appUserID, subscriberID int) (*domain.AppSubscriberProfile, error) {
	var p domain.AppSubscriberProfile
	err := r.db.QueryRow(ctx, `
		SELECT au.app_user_id, s.subscriber_id, s.subscriber_code,
		       s.full_name, s.mobile, s.email, s.kyc_status,
		       au.is_verified, au.last_login
		FROM app_users au
		INNER JOIN subscribers s ON au.subscriber_id = s.subscriber_id
		WHERE au.app_user_id = $1 AND s.subscriber_id = $2
	`, appUserID, subscriberID).Scan(
		&p.AppUserID, &p.SubscriberID, &p.SubscriberCode,
		&p.FullName, &p.Mobile, &p.Email, &p.KYCStatus,
		&p.IsVerified, &p.LastLogin,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *MobileRepository) GetSubscriberDashboard(ctx context.Context, subscriberID int) (*domain.AppDashboard, error) {
	var d domain.AppDashboard
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT cs.chit_id)                                         AS total_chits,
			COUNT(DISTINCT CASE WHEN cg.status = 'ACTIVE' THEN cs.chit_id END) AS active_chits,
			COALESCE(SUM(si.amount_paid), 0)                                   AS total_paid,
			COALESCE(SUM(si.balance_amount), 0)                                AS total_pending,
			COALESCE(SUM(CASE WHEN si.status = 'OVERDUE' THEN si.balance_amount ELSE 0 END), 0) AS overdue_amount
		FROM subscribers s
		LEFT JOIN chit_subscribers cs ON s.subscriber_id = cs.subscriber_id AND cs.status = 'ACTIVE'
		LEFT JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		LEFT JOIN subscriber_installments si ON cs.chit_subscriber_id = si.chit_subscriber_id
		WHERE s.subscriber_id = $1
	`, subscriberID).Scan(
		&d.TotalChits, &d.ActiveChits, &d.TotalPaid, &d.TotalPending, &d.OverdueAmount,
	)
	if err != nil {
		return nil, err
	}

	// Next upcoming due
	var nextDue domain.AppInstallmentDue
	errDue := r.db.QueryRow(ctx, `
		SELECT TO_CHAR(i.due_date, 'YYYY-MM-DD'), si.total_payable
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		WHERE si.subscriber_id = $1
		  AND si.status IN ('PENDING','PARTIAL')
		  AND i.due_date >= CURRENT_DATE
		ORDER BY i.due_date ASC
		LIMIT 1
	`, subscriberID).Scan(&nextDue.DueDate, &nextDue.TotalPayable)
	if errDue == nil {
		d.UpcomingDueDate = &nextDue.DueDate
		d.UpcomingDueAmount = nextDue.TotalPayable
	}

	// Unread notifications
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM app_notifications WHERE subscriber_id = $1 AND is_read = false`,
		subscriberID,
	).Scan(&d.UnreadNotifications)

	return &d, nil
}

func (r *MobileRepository) GetSubscriberChits(ctx context.Context, subscriberID int) ([]*domain.AppChitSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cg.chit_id, cg.chit_code, cg.chit_name, cg.chit_value,
		       cg.installment_amount, cg.current_month, cg.total_months, cg.status,
		       cs.is_prized, cs.prized_month, cs.slot_number,
		       TO_CHAR(MIN(i.due_date), 'YYYY-MM-DD') AS next_due_date,
		       COALESCE(MIN(si.total_payable), 0) AS next_due_amount
		FROM chit_subscribers cs
		INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
		LEFT JOIN subscriber_installments si ON cs.chit_subscriber_id = si.chit_subscriber_id
		    AND si.status IN ('PENDING','PARTIAL')
		LEFT JOIN installments i ON si.installment_id = i.installment_id
		    AND i.due_date >= CURRENT_DATE
		WHERE cs.subscriber_id = $1 AND cs.status = 'ACTIVE'
		GROUP BY cg.chit_id, cg.chit_code, cg.chit_name, cg.chit_value,
		         cg.installment_amount, cg.current_month, cg.total_months, cg.status,
		         cs.is_prized, cs.prized_month, cs.slot_number
		ORDER BY cg.status DESC, cg.chit_name
	`, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chits []*domain.AppChitSummary
	for rows.Next() {
		var c domain.AppChitSummary
		if err := rows.Scan(
			&c.ChitID, &c.ChitCode, &c.ChitName, &c.ChitValue,
			&c.InstallmentAmount, &c.CurrentMonth, &c.TotalMonths, &c.Status,
			&c.IsPrized, &c.PrizedMonth, &c.SlotNumber,
			&c.NextDueDate, &c.NextDueAmount,
		); err != nil {
			return nil, err
		}
		chits = append(chits, &c)
	}
	return chits, rows.Err()
}

func (r *MobileRepository) GetSubscriberDues(ctx context.Context, subscriberID int) ([]*domain.AppInstallmentDue, error) {
	rows, err := r.db.Query(ctx, `
		SELECT si.subscriber_installment_id, cg.chit_id, cg.chit_name, cg.chit_code,
		       i.month_number, TO_CHAR(i.due_date, 'YYYY-MM-DD'),
		       si.amount_due, si.amount_paid, si.balance_amount,
		       si.penalty_amount, si.total_payable, si.status,
		       (i.due_date < CURRENT_DATE AND si.status != 'PAID') AS is_overdue
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
		INNER JOIN chit_subscribers cs ON si.chit_subscriber_id = cs.chit_subscriber_id
		WHERE si.subscriber_id = $1
		  AND si.status IN ('PENDING','PARTIAL','OVERDUE')
		ORDER BY i.due_date ASC
	`, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dues []*domain.AppInstallmentDue
	for rows.Next() {
		var d domain.AppInstallmentDue
		if err := rows.Scan(
			&d.SubscriberInstallmentID, &d.ChitID, &d.ChitName, &d.ChitCode,
			&d.MonthNumber, &d.DueDate,
			&d.AmountDue, &d.AmountPaid, &d.BalanceAmount,
			&d.PenaltyAmount, &d.TotalPayable, &d.Status, &d.IsOverdue,
		); err != nil {
			return nil, err
		}
		dues = append(dues, &d)
	}
	return dues, rows.Err()
}

func (r *MobileRepository) GetSubscriberPaymentHistory(ctx context.Context, subscriberID, limit, offset int) ([]*domain.AppPaymentHistoryItem, int64, error) {
	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM payment_transactions WHERE subscriber_id = $1`,
		subscriberID,
	).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT pt.transaction_id, cg.chit_name, i.month_number,
		       pt.amount, pt.payment_mode,
		       pt.transaction_reference, pt.receipt_number,
		       TO_CHAR(pt.transaction_date, 'YYYY-MM-DD'),
		       pt.status, pt.is_online
		FROM payment_transactions pt
		LEFT JOIN subscriber_installments si ON pt.subscriber_installment_id = si.subscriber_installment_id
		LEFT JOIN installments i ON si.installment_id = i.installment_id
		LEFT JOIN chit_groups cg ON pt.chit_id = cg.chit_id
		WHERE pt.subscriber_id = $1
		ORDER BY pt.transaction_date DESC
		LIMIT $2 OFFSET $3
	`, subscriberID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*domain.AppPaymentHistoryItem
	for rows.Next() {
		var item domain.AppPaymentHistoryItem
		if err := rows.Scan(
			&item.TransactionID, &item.ChitName, &item.MonthNumber,
			&item.Amount, &item.PaymentMode,
			&item.Reference, &item.ReceiptNumber,
			&item.PaymentDate, &item.Status, &item.IsOnline,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, &item)
	}
	return items, total, rows.Err()
}

func (r *MobileRepository) GetNotifications(ctx context.Context, subscriberID, limit, offset int) ([]*domain.AppNotification, int64, error) {
	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM app_notifications WHERE subscriber_id = $1`,
		subscriberID,
	).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT notification_id, subscriber_id, title, message,
		       notification_type, is_read, read_at, data, sent_at
		FROM app_notifications
		WHERE subscriber_id = $1
		ORDER BY sent_at DESC LIMIT $2 OFFSET $3
	`, subscriberID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifs []*domain.AppNotification
	for rows.Next() {
		var n domain.AppNotification
		if err := rows.Scan(
			&n.NotificationID, &n.SubscriberID, &n.Title, &n.Message,
			&n.NotificationType, &n.IsRead, &n.ReadAt, &n.Data, &n.SentAt,
		); err != nil {
			return nil, 0, err
		}
		notifs = append(notifs, &n)
	}
	return notifs, total, rows.Err()
}

func (r *MobileRepository) MarkNotificationRead(ctx context.Context, notificationID, subscriberID int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE app_notifications SET is_read = true, read_at = NOW()
		WHERE notification_id = $1 AND subscriber_id = $2
	`, notificationID, subscriberID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MobileRepository) MarkAllNotificationsRead(ctx context.Context, subscriberID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE app_notifications SET is_read = true, read_at = NOW()
		WHERE subscriber_id = $1 AND is_read = false
	`, subscriberID)
	return err
}

// ── Agent App ─────────────────────────────────────────────────────────────────

func (r *MobileRepository) GetAgentDashboard(ctx context.Context, agentUserID, companyID int) (*domain.AgentCollectionSummary, error) {
	var d domain.AgentCollectionSummary
	d.TodayDate = time.Now().Format("2006-01-02")

	// Total due today across all subscribers
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(si.total_payable), 0) AS total_due,
			COALESCE(SUM(CASE WHEN si.status = 'PAID' THEN si.amount_paid ELSE 0 END), 0) AS total_collected
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		INNER JOIN chit_groups cg ON i.chit_id = cg.chit_id
		WHERE cg.company_id = $1
		  AND i.due_date = CURRENT_DATE
	`, companyID).Scan(&d.TotalDueToday, &d.TotalCollectedToday)
	d.PendingToCollect = d.TotalDueToday - d.TotalCollectedToday

	// Subscribers visited vs pending today (via transactions by this agent today)
	r.db.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT subscriber_id) AS visited,
			0
		FROM payment_transactions
		WHERE collected_by = $1
		  AND DATE(transaction_date) = CURRENT_DATE
	`, agentUserID).Scan(&d.SubscribersVisited, &d.SubscribersPending)

	// Recent collections by this agent today
	rows, err := r.db.Query(ctx, `
		SELECT pt.transaction_id,
		       COALESCE(s.full_name, '') AS subscriber_name,
		       COALESCE(cg.chit_name, '') AS chit_name,
		       pt.amount, pt.payment_mode,
		       TO_CHAR(pt.transaction_date, 'YYYY-MM-DD HH24:MI')
		FROM payment_transactions pt
		LEFT JOIN subscribers s ON pt.subscriber_id = s.subscriber_id
		LEFT JOIN chit_groups cg ON pt.chit_id = cg.chit_id
		WHERE pt.collected_by = $1
		  AND DATE(pt.transaction_date) = CURRENT_DATE
		ORDER BY pt.transaction_date DESC
		LIMIT 10
	`, agentUserID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rc domain.AgentRecentCollection
			if errScan := rows.Scan(
				&rc.TransactionID, &rc.SubscriberName, &rc.ChitName,
				&rc.Amount, &rc.PaymentMode, &rc.RecordedAt,
			); errScan == nil {
				d.RecentCollections = append(d.RecentCollections, &rc)
			}
		}
	}
	if d.RecentCollections == nil {
		d.RecentCollections = []*domain.AgentRecentCollection{}
	}
	return &d, nil
}

func (r *MobileRepository) GetAgentSubscriberList(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.AgentSubscriberListItem, int64, error) {
	base := `
		SELECT DISTINCT s.subscriber_id, s.subscriber_code, s.full_name, s.mobile,
		       COALESCE(s.address, '') AS address,
		       COALESCE(SUM(si.balance_amount), 0) AS total_pending,
		       (COALESCE(SUM(CASE WHEN si.status = 'OVERDUE' THEN si.balance_amount ELSE 0 END), 0) > 0) AS is_overdue
		FROM subscribers s
		LEFT JOIN chit_subscribers cs ON s.subscriber_id = cs.subscriber_id AND cs.status = 'ACTIVE'
		LEFT JOIN subscriber_installments si ON cs.chit_subscriber_id = si.chit_subscriber_id
		    AND si.status IN ('PENDING','PARTIAL','OVERDUE')`

	args := []interface{}{companyID}
	where := " WHERE s.company_id = $1"
	argPos := 2

	if search, ok := filters["search"].(string); ok && search != "" {
		where += fmt.Sprintf(" AND (s.full_name ILIKE $%d OR s.mobile ILIKE $%d OR s.subscriber_code ILIKE $%d)",
			argPos, argPos, argPos)
		args = append(args, "%"+search+"%")
		argPos++
	}
	if overdueOnly, ok := filters["overdue"].(bool); ok && overdueOnly {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM subscriber_installments si2 "+
			"INNER JOIN chit_subscribers cs2 ON si2.chit_subscriber_id = cs2.chit_subscriber_id "+
			"WHERE cs2.subscriber_id = s.subscriber_id AND si2.status = 'OVERDUE')")
	}

	groupBy := " GROUP BY s.subscriber_id, s.subscriber_code, s.full_name, s.mobile, s.address"

	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM (`+base+where+groupBy+`) sq`,
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
	orderLimit := fmt.Sprintf(" ORDER BY is_overdue DESC, s.full_name LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, base+where+groupBy+orderLimit, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*domain.AgentSubscriberListItem
	for rows.Next() {
		var item domain.AgentSubscriberListItem
		if err := rows.Scan(
			&item.SubscriberID, &item.SubscriberCode, &item.FullName,
			&item.Mobile, &item.Address, &item.TotalPending, &item.IsOverdue,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, &item)
	}
	if items == nil {
		items = []*domain.AgentSubscriberListItem{}
	}
	return items, total, rows.Err()
}

// RecordCashPayment records an offline cash/cheque collection by the agent.
// Uses a transaction: updates subscriber_installment + inserts payment_transaction.
func (r *MobileRepository) RecordCashPayment(ctx context.Context,
	req *domain.AgentRecordCashPaymentRequest,
	agentUserID, companyID, branchID int,
) (*domain.AgentRecentCollection, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Lock and verify installment
	var subscriberID, chitID int
	var amountDue, amountPaid float64
	var siStatus string
	err = tx.QueryRow(ctx, `
		SELECT si.subscriber_id,
		       i.chit_id,
		       si.amount_due,
		       si.amount_paid,
		       si.status
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		WHERE si.subscriber_installment_id = $1
		  AND si.status IN ('PENDING','PARTIAL','OVERDUE')
		FOR UPDATE
	`, req.SubscriberInstallmentID).Scan(&subscriberID, &chitID, &amountDue, &amountPaid, &siStatus)
	if err == pgx.ErrNoRows {
		return nil, domain.NewAppError("INSTALLMENT_NOT_FOUND",
			"Installment not found or already paid", nil)
	}
	if err != nil {
		return nil, err
	}

	// 2. Validate amount
	remaining := amountDue - amountPaid
	if req.Amount > remaining {
		return nil, domain.NewAppError("AMOUNT_EXCEEDS_DUE",
			fmt.Sprintf("Payment amount %.2f exceeds remaining balance %.2f", req.Amount, remaining), nil)
	}

	newAmountPaid := amountPaid + req.Amount
	newStatus := "PARTIAL"
	if newAmountPaid >= amountDue {
		newStatus = "PAID"
	}

	// 3. Update subscriber_installment
	_, err = tx.Exec(ctx, `
		UPDATE subscriber_installments
		SET amount_paid      = $1,
		    status           = $2,
		    payment_mode     = $3,
		    transaction_reference = $4,
		    payment_date     = CURRENT_DATE,
		    payment_collected_by = $5,
		    updated_at       = NOW()
		WHERE subscriber_installment_id = $6
	`, newAmountPaid, newStatus, req.PaymentMode,
		req.TransactionReference, agentUserID, req.SubscriberInstallmentID)
	if err != nil {
		return nil, err
	}

	// 4. Insert payment_transaction
	var txnID int
	err = tx.QueryRow(ctx, `
		INSERT INTO payment_transactions
			(subscriber_installment_id, subscriber_id, chit_id,
			 transaction_type, amount, payment_mode, transaction_reference,
			 transaction_date, collected_by, branch_id,
			 status, company_id, is_online, remarks)
		VALUES ($1,$2,$3,'INSTALLMENT',$4,$5,$6,NOW(),$7,$8,'SUCCESS',$9,false,$10)
		RETURNING transaction_id
	`, req.SubscriberInstallmentID, subscriberID, chitID,
		req.Amount, req.PaymentMode, req.TransactionReference,
		agentUserID, branchID, companyID, req.Remarks,
	).Scan(&txnID)
	if err != nil {
		return nil, err
	}

	// 5. Financial ledger entry
	_, err = tx.Exec(ctx, `
		INSERT INTO financial_ledger_entries
			(company_id, account, direction, amount, source_type,
			 source_id, subscriber_id, chit_id, description)
		VALUES ($1,'COLLECTIONS','CREDIT',$2,'payment_transaction',$3,$4,$5,$6)
	`, companyID, req.Amount, txnID, subscriberID, chitID,
		fmt.Sprintf("Cash collection by agent %d", agentUserID))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 6. Return summary for response
	var result domain.AgentRecentCollection
	result.TransactionID = txnID
	result.Amount = req.Amount
	result.PaymentMode = req.PaymentMode
	result.RecordedAt = time.Now().Format("2006-01-02 15:04")
	r.db.QueryRow(ctx,
		`SELECT full_name FROM subscribers WHERE subscriber_id = $1`, subscriberID,
	).Scan(&result.SubscriberName)
	r.db.QueryRow(ctx,
		`SELECT chit_name FROM chit_groups WHERE chit_id = $1`, chitID,
	).Scan(&result.ChitName)
	return &result, nil
}
