package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type PaymentRepository struct {
	db *database.PostgresPool
}

func NewPaymentRepository(db *database.PostgresPool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) DB() *database.PostgresPool { return r.db }

// ── Installment idempotency guard ─────────────────────────────────────────────

// GetExistingPaymentForInstallment returns a non-nil Payment if the installment
// already has an active payment order. Callers must return it instead of creating a new one.
func (r *PaymentRepository) GetExistingPaymentForInstallment(
	ctx context.Context, subscriberInstallmentID, companyID int,
) (*domain.Payment, error) {
	var pm domain.Payment
	var err error

	txErr := r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		err = tx.QueryRow(ctx, `
			SELECT p.payment_id, p.payment_uid, p.company_id, p.branch_id,
				p.subscriber_id, p.chit_id, p.subscriber_installment_id,
				p.config_id, p.provider, p.purpose, p.amount, p.currency,
				p.amount_captured, p.amount_refunded, p.gateway_order_id,
				p.gateway_payment_id, p.status, p.idempotency_key,
				p.expires_at, p.failure_code, p.failure_reason,
				p.notes, p.initiated_by, p.created_at, p.updated_at
			FROM payments p
			WHERE p.subscriber_installment_id = $1
			  AND p.company_id = $2
			  AND p.status NOT IN ('FAILED','CANCELLED','EXPIRED','REFUNDED')
			ORDER BY p.created_at DESC
			LIMIT 1
		`, subscriberInstallmentID, companyID).Scan(
			&pm.PaymentID, &pm.PaymentUID, &pm.CompanyID, &pm.BranchID,
			&pm.SubscriberID, &pm.ChitID, &pm.SubscriberInstallmentID,
			&pm.ConfigID, &pm.Provider, &pm.Purpose, &pm.Amount, &pm.Currency,
			&pm.AmountCaptured, &pm.AmountRefunded, &pm.GatewayOrderID,
			&pm.GatewayPaymentID, &pm.Status, &pm.IdempotencyKey,
			&pm.ExpiresAt, &pm.FailureCode, &pm.FailureReason,
			&pm.Notes, &pm.InitiatedBy, &pm.CreatedAt, &pm.UpdatedAt,
		)
		return err
	})
	if txErr != nil || err == pgx.ErrNoRows {
		return nil, nil // no existing payment
	}
	return &pm, nil
}

// ── Create payment order ──────────────────────────────────────────────────────

// CreatePayment inserts a new payment record (status=CREATED) and
// pins the payment_id on the subscriber_installment row.
// Uses WithCompanyTx so RLS sees the correct company_id.
func (r *PaymentRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	notes, _ := json.Marshal(map[string]string{})
	idempotencyKey := generatePaymentKey()

	return r.db.WithCompanyTx(ctx, p.CompanyID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO payments (
				company_id, branch_id, subscriber_id, chit_id,
				subscriber_installment_id, config_id, provider, purpose,
				amount, currency, status, idempotency_key,
				expires_at, notes, initiated_by
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'INR','CREATED',$10,
			          NOW() + INTERVAL '15 minutes', $11, $12)
			RETURNING payment_id, payment_uid, created_at, updated_at
		`,
			p.CompanyID, p.BranchID, p.SubscriberID, p.ChitID,
			p.SubscriberInstallmentID, p.ConfigID, p.Provider, p.Purpose,
			p.Amount, idempotencyKey, notes, p.InitiatedBy,
		).Scan(&p.PaymentID, &p.PaymentUID, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return err
		}
		p.IdempotencyKey = idempotencyKey

		// Pin payment_id on the installment (idempotency guard at DB level)
		if p.SubscriberInstallmentID != nil {
			_, err = tx.Exec(ctx, `
				UPDATE subscriber_installments
				SET payment_id = $1, updated_at = NOW()
				WHERE subscriber_installment_id = $2
				  AND payment_id IS NULL
			`, p.PaymentID, *p.SubscriberInstallmentID)
		}
		return err
	})
}

// UpdatePaymentGatewayOrder sets gateway_order_id and moves to INITIATED.
func (r *PaymentRepository) UpdatePaymentGatewayOrder(ctx context.Context, paymentID int,
	gatewayOrderID string, companyID int) error {
	return r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE payments
			SET gateway_order_id = $1, status = 'INITIATED', updated_at = NOW()
			WHERE payment_id = $2
		`, gatewayOrderID, paymentID)
		return err
	})
}

// MarkPaymentSuccess captures the payment and updates all downstream tables atomically.
// Tables updated: payments, payment_attempts, subscriber_installments, financial_ledger_entries, payment_transactions.
func (r *PaymentRepository) MarkPaymentSuccess(ctx context.Context,
	paymentID int, gatewayPaymentID string, amountCaptured float64,
	attemptPayload []byte, companyID int) error {

	return r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		// 1. Mark payment captured
		var subscriberID, chitID int
		var siID *int
		var branchID *int
		err := tx.QueryRow(ctx, `
			UPDATE payments SET
				status = 'SUCCESS',
				gateway_payment_id = $1,
				amount_captured = $2,
				updated_at = NOW()
			WHERE payment_id = $3
			RETURNING subscriber_id, chit_id, subscriber_installment_id, branch_id
		`, gatewayPaymentID, amountCaptured, paymentID,
		).Scan(&subscriberID, &chitID, &siID, &branchID)
		if err != nil {
			return fmt.Errorf("mark payment success: %w", err)
		}

		// 2. Record payment attempt
		meta := attemptPayload
		if meta == nil {
			meta = []byte("{}")
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO payment_attempts
				(company_id, payment_id, attempt_number, provider,
				 amount, status, response_payload, completed_at)
			SELECT $1, $2,
				COALESCE((SELECT MAX(attempt_number) FROM payment_attempts WHERE payment_id = $2), 0) + 1,
				provider, amount_captured, 'SUCCESS', $3, NOW()
			FROM payments WHERE payment_id = $2
		`, companyID, paymentID, meta)
		if err != nil {
			return fmt.Errorf("record attempt: %w", err)
		}

		// 3. Update subscriber_installment
		if siID != nil {
			_, err = tx.Exec(ctx, `
				UPDATE subscriber_installments SET
					amount_paid    = $1,
					balance_amount = GREATEST(0, amount_due - $1),
					status         = CASE WHEN $1 >= amount_due THEN 'PAID' ELSE 'PARTIAL' END,
					payment_mode   = 'ONLINE',
					payment_date   = CURRENT_DATE,
					updated_at     = NOW()
				WHERE subscriber_installment_id = $2
			`, amountCaptured, *siID)
			if err != nil {
				return fmt.Errorf("update installment: %w", err)
			}
		}

		// 4. Double-entry ledger (DEBIT receivable, CREDIT collections)
		journalID := newUUID()
		entries := []struct{ account, direction string }{
			{"installment_receivable", "DEBIT"},
			{"collections", "CREDIT"},
		}
		for _, e := range entries {
			_, err = tx.Exec(ctx, `
				INSERT INTO financial_ledger_entries
					(company_id, branch_id, journal_id, account, direction, amount,
					 source_type, source_id, subscriber_id, chit_id, description)
				VALUES ($1,$2,$3,$4,$5,$6,'payment',$7,$8,$9,'Online installment payment')
			`, companyID, branchID, journalID, e.account, e.direction,
				amountCaptured, paymentID, subscriberID, chitID)
			if err != nil {
				return fmt.Errorf("ledger entry %s: %w", e.direction, err)
			}
		}

		// 5. payment_transactions record
		_, err = tx.Exec(ctx, `
			INSERT INTO payment_transactions
				(subscriber_installment_id, subscriber_id, chit_id,
				 transaction_type, amount, payment_mode, transaction_reference,
				 transaction_date, branch_id, status, company_id,
				 payment_id, gateway_provider, is_online)
			VALUES ($1,$2,$3,'INSTALLMENT',$4,'ONLINE',$5,NOW(),
				$6,'SUCCESS',$7,$8,
				(SELECT provider FROM payments WHERE payment_id = $8),
				true)
		`, siID, subscriberID, chitID, amountCaptured, gatewayPaymentID,
			branchID, companyID, paymentID)
		return err
	})
}

// MarkPaymentFailed records failure details and a failed attempt.
func (r *PaymentRepository) MarkPaymentFailed(ctx context.Context,
	paymentID int, failureCode, failureReason string, companyID int) error {
	return r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE payments SET
				status = 'FAILED', failure_code = $1,
				failure_reason = $2, updated_at = NOW()
			WHERE payment_id = $3
		`, failureCode, failureReason, paymentID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO payment_attempts
				(company_id, payment_id, attempt_number,
				 amount, status, error_code, error_description, completed_at)
			SELECT $1, $2,
				COALESCE((SELECT MAX(attempt_number) FROM payment_attempts WHERE payment_id = $2), 0) + 1,
				amount, 'FAILED', $3, $4, NOW()
			FROM payments WHERE payment_id = $2
		`, companyID, paymentID, failureCode, failureReason)
		return err
	})
}

// GetPaymentByID fetches a single payment scoped to company.
func (r *PaymentRepository) GetPaymentByID(ctx context.Context, paymentID, companyID int) (*domain.Payment, error) {
	var pm domain.Payment
	var scanErr error

	txErr := r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		scanErr = tx.QueryRow(ctx, `
			SELECT p.payment_id, p.payment_uid, p.company_id, p.branch_id,
				p.subscriber_id, p.chit_id, p.subscriber_installment_id,
				p.config_id, p.provider, p.purpose, p.amount, p.currency,
				p.amount_captured, p.amount_refunded, p.gateway_order_id,
				p.gateway_payment_id, p.status, p.idempotency_key,
				p.expires_at, p.failure_code, p.failure_reason,
				p.notes, p.initiated_by, p.created_at, p.updated_at,
				COALESCE(s.full_name,'') AS subscriber_name,
				COALESCE(cg.chit_name,'') AS chit_name
			FROM payments p
			INNER JOIN subscribers s ON p.subscriber_id = s.subscriber_id
			LEFT JOIN chit_groups cg ON p.chit_id = cg.chit_id
			WHERE p.payment_id = $1 AND p.company_id = $2
		`, paymentID, companyID).Scan(
			&pm.PaymentID, &pm.PaymentUID, &pm.CompanyID, &pm.BranchID,
			&pm.SubscriberID, &pm.ChitID, &pm.SubscriberInstallmentID,
			&pm.ConfigID, &pm.Provider, &pm.Purpose, &pm.Amount, &pm.Currency,
			&pm.AmountCaptured, &pm.AmountRefunded, &pm.GatewayOrderID,
			&pm.GatewayPaymentID, &pm.Status, &pm.IdempotencyKey,
			&pm.ExpiresAt, &pm.FailureCode, &pm.FailureReason,
			&pm.Notes, &pm.InitiatedBy, &pm.CreatedAt, &pm.UpdatedAt,
			&pm.SubscriberName, &pm.ChitName,
		)
		return scanErr
	})
	if txErr == pgx.ErrNoRows || scanErr == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if txErr != nil {
		return nil, txErr
	}
	return &pm, nil
}

// ListPayments returns paginated payments for a company.
func (r *PaymentRepository) ListPayments(ctx context.Context, companyID int,
	filters map[string]interface{}) ([]*domain.Payment, int64, error) {

	var payments []*domain.Payment
	var total int64

	txErr := r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		base := `
			SELECT p.payment_id, p.payment_uid, p.company_id, p.branch_id,
				p.subscriber_id, p.chit_id, p.subscriber_installment_id,
				p.config_id, p.provider, p.purpose, p.amount, p.currency,
				p.amount_captured, p.amount_refunded, p.gateway_order_id,
				p.gateway_payment_id, p.status, p.idempotency_key,
				p.expires_at, p.failure_code, p.failure_reason,
				p.notes, p.initiated_by, p.created_at, p.updated_at,
				COALESCE(s.full_name,'') AS subscriber_name,
				COALESCE(cg.chit_name,'') AS chit_name
			FROM payments p
			INNER JOIN subscribers s ON p.subscriber_id = s.subscriber_id
			LEFT JOIN chit_groups cg ON p.chit_id = cg.chit_id`

		args := []interface{}{companyID}
		where := " WHERE p.company_id = $1"
		argPos := 2

		if status, ok := filters["status"].(string); ok && status != "" {
			where += fmt.Sprintf(" AND p.status = $%d", argPos)
			args = append(args, status)
			argPos++
		}
		if subID, ok := filters["subscriber_id"].(int); ok && subID > 0 {
			where += fmt.Sprintf(" AND p.subscriber_id = $%d", argPos)
			args = append(args, subID)
			argPos++
		}

		tx.QueryRow(ctx, `SELECT COUNT(*) FROM payments p`+where, args...).Scan(&total)

		limit, offset := 20, 0
		if l, ok := filters["limit"].(int); ok && l > 0 {
			limit = l
		}
		if o, ok := filters["offset"].(int); ok {
			offset = o
		}
		query := base + where + fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var pm domain.Payment
			if err := rows.Scan(
				&pm.PaymentID, &pm.PaymentUID, &pm.CompanyID, &pm.BranchID,
				&pm.SubscriberID, &pm.ChitID, &pm.SubscriberInstallmentID,
				&pm.ConfigID, &pm.Provider, &pm.Purpose, &pm.Amount, &pm.Currency,
				&pm.AmountCaptured, &pm.AmountRefunded, &pm.GatewayOrderID,
				&pm.GatewayPaymentID, &pm.Status, &pm.IdempotencyKey,
				&pm.ExpiresAt, &pm.FailureCode, &pm.FailureReason,
				&pm.Notes, &pm.InitiatedBy, &pm.CreatedAt, &pm.UpdatedAt,
				&pm.SubscriberName, &pm.ChitName,
			); err != nil {
				return err
			}
			payments = append(payments, &pm)
		}
		return rows.Err()
	})
	return payments, total, txErr
}

// ── Gateway config ────────────────────────────────────────────────────────────

// GetActiveGatewayConfig fetches the highest-priority active config for a provider.
func (r *PaymentRepository) GetActiveGatewayConfig(ctx context.Context,
	companyID int, provider string, forPayouts bool) (*domain.PaymentGatewayConfig, error) {

	var cfg domain.PaymentGatewayConfig
	var scanErr error
	var supportsCol string
	if forPayouts {
		supportsCol = "supports_payouts"
	} else {
		supportsCol = "supports_payins"
	}

	txErr := r.db.WithCompanyTx(ctx, companyID, func(tx pgx.Tx) error {
		scanErr = tx.QueryRow(ctx, fmt.Sprintf(`
			SELECT config_id, company_id, provider, display_name, mode,
				merchant_id, public_key, api_key_secret_ref, api_secret_secret_ref,
				webhook_secret_ref, api_secret_encrypted, webhook_secret_encrypted,
				supports_payins, supports_payouts, priority, is_active,
				metadata, created_by, created_at, updated_at, secret_storage_mode
			FROM payment_gateway_configs
			WHERE company_id = $1 AND provider = $2
			  AND %s = true AND is_active = true
			ORDER BY priority ASC LIMIT 1
		`, supportsCol), companyID, provider).Scan(
			&cfg.ConfigID, &cfg.CompanyID, &cfg.Provider, &cfg.DisplayName, &cfg.Mode,
			&cfg.MerchantID, &cfg.PublicKey, &cfg.APIKeySecretRef, &cfg.APISecretSecretRef,
			&cfg.WebhookSecretRef, &cfg.APISecretEncrypted, &cfg.WebhookSecretEncrypted,
			&cfg.SupportsPayins, &cfg.SupportsPayouts, &cfg.Priority, &cfg.IsActive,
			&cfg.Metadata, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.UpdatedAt, &cfg.SecretStorageMode,
		)
		return scanErr
	})
	if txErr == pgx.ErrNoRows || scanErr == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if txErr != nil {
		return nil, txErr
	}
	return &cfg, nil
}

// GetInstallmentForPayment resolves subscriber, chit, and amount for an installment.
// Also checks that payment_id IS NULL (idempotency guard).
func (r *PaymentRepository) GetInstallmentForPayment(ctx context.Context,
	subscriberInstallmentID int) (subscriberID int, chitID int, amount float64, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT si.subscriber_id, i.chit_id, si.net_payable
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		WHERE si.subscriber_installment_id = $1
		  AND si.status IN ('PENDING','PARTIAL','OVERDUE')
		  AND si.payment_id IS NULL
	`, subscriberInstallmentID).Scan(&subscriberID, &chitID, &amount)
	// net_payable is a GENERATED column: total_payable - dividend_adjustment
	return
}

// ── helpers ───────────────────────────────────────────────────────────────────

func generatePaymentKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b) + fmt.Sprintf("_%d", time.Now().UnixNano())
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
