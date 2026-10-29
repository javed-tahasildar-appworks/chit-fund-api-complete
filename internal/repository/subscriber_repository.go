package repository

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type SubscriberRepository struct {
	db *database.PostgresPool
}

func NewSubscriberRepository(db *database.PostgresPool) *SubscriberRepository {
	return &SubscriberRepository{db: db}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func (r *SubscriberRepository) Create(ctx context.Context, s *domain.Subscriber) error {
	// All columns from the subscribers table — matches DB schema exactly
	query := `
		INSERT INTO subscribers (
			company_id, branch_id, subscriber_code, full_name, father_name, dob, gender,
			aadhaar_number, pan_number, address, city, state_id, pincode,
			mobile, alternate_mobile, email, occupation, annual_income,
			kyc_status, photo_path, signature_path, credit_score,
			is_blacklisted, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21, $22,
			false, $23
		)
		RETURNING subscriber_id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		s.CompanyID, s.BranchID, s.SubscriberCode, s.FullName, s.FatherName, s.DOB, s.Gender,
		s.AadhaarNumber, s.PANNumber, s.Address, s.City, s.StateID, s.Pincode,
		s.Mobile, s.AlternateMobile, s.Email, s.Occupation, s.AnnualIncome,
		s.KYCStatus, s.PhotoPath, s.SignaturePath, s.CreditScore,
		s.CreatedBy,
	).Scan(&s.SubscriberID, &s.CreatedAt, &s.UpdatedAt)
}

// ─── GetByID — returns SubscriberListItem shaped for Flutter SubscriberModel ──

func (r *SubscriberRepository) GetByID(ctx context.Context, id, companyID int) (*domain.SubscriberListItem, error) {
	query := `
		SELECT
			s.subscriber_id::text                                AS id,
			COALESCE(s.subscriber_code, '')                      AS subscriber_code,
			s.company_id::text                                   AS company_id,
			COALESCE((
				SELECT cg.branch_id::text
				FROM chit_subscribers cs
				INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
				WHERE cs.subscriber_id = s.subscriber_id AND cs.status = 'ACTIVE'
				ORDER BY cs.created_at DESC LIMIT 1
			), '')                                               AS branch_id,
			COALESCE(s.full_name, '')                            AS full_name,
			COALESCE(TO_CHAR(s.dob, 'YYYY-MM-DD'), '')           AS date_of_birth,
			COALESCE(LOWER(s.gender), '')                        AS gender,
			COALESCE(s.mobile, '')                               AS mobile,
			COALESCE(s.alternate_mobile, '')                     AS alternate_mobile,
			COALESCE(s.email, '')                                AS email,
			COALESCE(s.address, '')                              AS address,
			COALESCE(s.city, '')                                 AS city,
			COALESCE(st.state_name, '')                          AS state,
			COALESCE(s.pincode, '')                              AS pincode,
			COALESCE(s.occupation, '')                           AS occupation,
			COALESCE(s.annual_income, 0)                         AS annual_income,
			COALESCE(s.aadhaar_number, '')                       AS aadhaar_number,
			COALESCE(s.pan_number, '')                           AS pan_number,
			COALESCE(s.credit_score, 0)                          AS credit_score,
			CASE
				WHEN s.is_blacklisted          THEN 'blacklisted'
				WHEN s.kyc_status = 'VERIFIED' THEN 'active'
				ELSE 'inactive'
			END                                                  AS status,
			s.is_blacklisted,
			COALESCE(s.blacklist_reason, '')                     AS blacklist_reason,
			CASE WHEN s.kyc_status = 'VERIFIED' THEN true ELSE false END AS is_kyc_complete,
			COALESCE((
				SELECT COUNT(*) FROM subscriber_documents sd
				WHERE sd.subscriber_id = s.subscriber_id
			), 0)::int                                           AS kyc_document_count,
			COALESCE((
				SELECT COUNT(*) FROM subscriber_documents sd
				WHERE sd.subscriber_id = s.subscriber_id
				  AND sd.verification_status = 'VERIFIED'
			), 0)::int                                           AS verified_document_count,
			COALESCE((
				SELECT COUNT(*) FROM chit_subscribers cs
				WHERE cs.subscriber_id = s.subscriber_id AND cs.status = 'ACTIVE'
			), 0)::int                                           AS active_chit_count,
			TO_CHAR(s.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			TO_CHAR(s.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
		FROM subscribers s
		LEFT JOIN states st ON s.state_id = st.state_id
		WHERE s.subscriber_id = $1 AND s.company_id = $2
	`
	var s domain.SubscriberListItem
	err := r.db.QueryRow(ctx, query, id, companyID).Scan(
		&s.ID, &s.SubscriberCode, &s.CompanyID, &s.BranchID,
		&s.FullName, &s.DateOfBirth, &s.Gender,
		&s.Mobile, &s.AlternateMobile, &s.Email,
		&s.Address, &s.City, &s.State, &s.Pincode,
		&s.Occupation, &s.AnnualIncome,
		&s.AadhaarNumber, &s.PANNumber, &s.CreditScore,
		&s.Status, &s.IsBlacklisted, &s.BlacklistReason,
		&s.IsKycComplete, &s.KycDocumentCount, &s.VerifiedDocumentCount,
		&s.ActiveChitCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &s, err
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (r *SubscriberRepository) Update(ctx context.Context, id, companyID int, req *domain.UpdateSubscriberRequest) error {
	query := `
		UPDATE subscribers SET
			branch_id        = COALESCE($1,  branch_id),
			full_name        = COALESCE($2,  full_name),
			mobile           = COALESCE($3,  mobile),
			alternate_mobile = COALESCE($4,  alternate_mobile),
			email            = COALESCE($5,  email),
			address          = COALESCE($6,  address),
			city             = COALESCE($7,  city),
			state_id         = COALESCE($8,  state_id),
			pincode          = COALESCE($9,  pincode),
			occupation       = COALESCE($10, occupation),
			annual_income    = COALESCE($11, annual_income),
			updated_at       = NOW()
		WHERE subscriber_id = $12 AND company_id = $13
	`
	result, err := r.db.Exec(ctx, query,
		req.BranchID, req.FullName, req.Mobile, req.AlternateMobile,
		req.Email, req.Address, req.City, req.StateID, req.Pincode,
		req.Occupation, req.AnnualIncome, id, companyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ─── List (paginated) ────────────────────────────────────────────────────────

func (r *SubscriberRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.SubscriberListItem, int64, error) {
	// branch_id: now a direct column on subscribers table.
	// Falls back to chit enrollment branch if direct column is null.
	// status is derived from is_blacklisted + kyc_status.
	base := `
		SELECT
			s.subscriber_id::text                                AS id,
			COALESCE(s.subscriber_code, '')                      AS subscriber_code,
			s.company_id::text                                   AS company_id,
			-- branch_id: use direct column, fall back to chit enrollment branch
			COALESCE(
				s.branch_id::text,
				(
					SELECT cg.branch_id::text
					FROM chit_subscribers cs
					INNER JOIN chit_groups cg ON cs.chit_id = cg.chit_id
					WHERE cs.subscriber_id = s.subscriber_id AND cs.status = 'ACTIVE'
					ORDER BY cs.created_at DESC LIMIT 1
				),
				''
			)                                                    AS branch_id,
			COALESCE(s.full_name, '')                            AS full_name,
			COALESCE(TO_CHAR(s.dob, 'YYYY-MM-DD'), '')           AS date_of_birth,
			COALESCE(LOWER(s.gender), '')                        AS gender,
			COALESCE(s.mobile, '')                               AS mobile,
			COALESCE(s.alternate_mobile, '')                     AS alternate_mobile,
			COALESCE(s.email, '')                                AS email,
			COALESCE(s.address, '')                              AS address,
			COALESCE(s.city, '')                                 AS city,
			COALESCE(st.state_name, '')                          AS state,
			COALESCE(s.pincode, '')                              AS pincode,
			COALESCE(s.occupation, '')                           AS occupation,
			COALESCE(s.annual_income, 0)                         AS annual_income,
			COALESCE(s.aadhaar_number, '')                       AS aadhaar_number,
			COALESCE(s.pan_number, '')                           AS pan_number,
			COALESCE(s.credit_score, 0)                          AS credit_score,
			-- status: derived from blacklist + kyc (no status column in DB)
			CASE
				WHEN s.is_blacklisted            THEN 'blacklisted'
				WHEN s.kyc_status = 'VERIFIED'   THEN 'active'
				ELSE 'inactive'
			END                                                  AS status,
			s.is_blacklisted,
			COALESCE(s.blacklist_reason, '')                     AS blacklist_reason,
			CASE WHEN s.kyc_status = 'VERIFIED' THEN true ELSE false END AS is_kyc_complete,
			COALESCE((
				SELECT COUNT(*) FROM subscriber_documents sd
				WHERE sd.subscriber_id = s.subscriber_id
			), 0)::int                                           AS kyc_document_count,
			COALESCE((
				SELECT COUNT(*) FROM subscriber_documents sd
				WHERE sd.subscriber_id = s.subscriber_id
				  AND sd.verification_status = 'VERIFIED'
			), 0)::int                                           AS verified_document_count,
			COALESCE((
				SELECT COUNT(*) FROM chit_subscribers cs
				WHERE cs.subscriber_id = s.subscriber_id AND cs.status = 'ACTIVE'
			), 0)::int                                           AS active_chit_count,
			TO_CHAR(s.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			TO_CHAR(s.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
		FROM subscribers s
		LEFT JOIN states st ON s.state_id = st.state_id`

	args := []interface{}{companyID}
	where := " WHERE s.company_id = $1"
	argPos := 2

	if status, ok := filters["kyc_status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND s.kyc_status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if blacklisted, ok := filters["is_blacklisted"].(bool); ok {
		where += fmt.Sprintf(" AND s.is_blacklisted = $%d", argPos)
		args = append(args, blacklisted)
		argPos++
	}
	if search, ok := filters["search"].(string); ok && search != "" {
		where += fmt.Sprintf(
			" AND (s.full_name ILIKE $%d OR s.mobile ILIKE $%d OR s.subscriber_code ILIKE $%d)",
			argPos, argPos, argPos,
		)
		args = append(args, "%"+search+"%")
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM subscribers s`+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}

	query := base + where +
		fmt.Sprintf(" ORDER BY s.full_name ASC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	subs := []*domain.SubscriberListItem{}
	for rows.Next() {
		var s domain.SubscriberListItem
		if err := rows.Scan(
			&s.ID, &s.SubscriberCode, &s.CompanyID, &s.BranchID,
			&s.FullName, &s.DateOfBirth, &s.Gender,
			&s.Mobile, &s.AlternateMobile, &s.Email,
			&s.Address, &s.City, &s.State, &s.Pincode,
			&s.Occupation, &s.AnnualIncome,
			&s.AadhaarNumber, &s.PANNumber, &s.CreditScore,
			&s.Status, &s.IsBlacklisted, &s.BlacklistReason,
			&s.IsKycComplete, &s.KycDocumentCount, &s.VerifiedDocumentCount,
			&s.ActiveChitCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		subs = append(subs, &s)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return subs, total, nil
}

// ─── Blacklist ────────────────────────────────────────────────────────────────

func (r *SubscriberRepository) SetBlacklist(ctx context.Context, id, companyID int, blacklisted bool, reason *string) error {
	query := `
		UPDATE subscribers SET is_blacklisted = $1, blacklist_reason = $2, updated_at = NOW()
		WHERE subscriber_id = $3 AND company_id = $4
	`
	result, err := r.db.Exec(ctx, query, blacklisted, reason, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ─── KYC Documents ────────────────────────────────────────────────────────────

type KYCRepository struct {
	db *database.PostgresPool
}

func NewKYCRepository(db *database.PostgresPool) *KYCRepository {
	return &KYCRepository{db: db}
}

func (r *KYCRepository) ListDocuments(ctx context.Context, subscriberID, companyID int) ([]*domain.SubscriberDocument, error) {
	// Columns match actual DB: document_id, subscriber_id, document_type, document_path,
	// document_number, uploaded_date, expiry_date, verification_status, verified_by, created_at
	query := `
		SELECT
			sd.document_id,
			sd.subscriber_id,
			COALESCE(sd.document_type, '')        AS document_type,
			COALESCE(sd.document_path, '')        AS document_path,
			COALESCE(sd.document_number, '')      AS document_number,
			COALESCE(TO_CHAR(sd.uploaded_date, 'YYYY-MM-DD'), '') AS uploaded_date,
			COALESCE(TO_CHAR(sd.expiry_date,   'YYYY-MM-DD'), '') AS expiry_date,
			COALESCE(sd.verification_status, 'PENDING') AS verification_status,
			COALESCE(sd.verified_by::text, '')    AS verified_by,
			TO_CHAR(sd.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM subscriber_documents sd
		INNER JOIN subscribers s ON sd.subscriber_id = s.subscriber_id
		WHERE sd.subscriber_id = $1 AND s.company_id = $2
		ORDER BY sd.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, subscriberID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := []*domain.SubscriberDocument{}
	for rows.Next() {
		var d domain.SubscriberDocument
		if err := rows.Scan(
			&d.DocumentID, &d.SubscriberID, &d.DocumentType,
			&d.FilePath, &d.DocumentNumber,
			&d.UploadedDate, &d.ExpiryDate,
			&d.VerificationStatus, &d.VerifiedByText,
			&d.CreatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, rows.Err()
}

func (r *KYCRepository) UploadDocument(ctx context.Context, doc *domain.SubscriberDocument) error {
	// DB columns: document_id, subscriber_id, document_type, document_path,
	//             document_number, uploaded_date, expiry_date, verification_status, verified_by, created_at
	var expiryDate *string
	if doc.ExpiryDate != "" {
		expiryDate = &doc.ExpiryDate
	}
	query := `
		INSERT INTO subscriber_documents (
			subscriber_id, document_type, document_path,
			document_number, uploaded_date, expiry_date, verification_status
		) VALUES ($1, $2, $3, $4, CURRENT_DATE, $5::date, 'PENDING')
		RETURNING
			document_id,
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`
	return r.db.QueryRow(ctx, query,
		doc.SubscriberID, doc.DocumentType, doc.FilePath,
		func() *string { if doc.DocumentNumber == "" { return nil }; s := doc.DocumentNumber; return &s }(),
		expiryDate,
	).Scan(&doc.DocumentID, &doc.CreatedAt)
}

func (r *KYCRepository) VerifyDocument(ctx context.Context, docID, subscriberID, verifiedBy int, req *domain.VerifyDocumentRequest) error {
	// Actual DB columns: verification_status, verified_by (no verified_at/rejection_reason/updated_at)
	query := `
		UPDATE subscriber_documents SET
			verification_status = $1,
			verified_by         = $2
		WHERE document_id = $3 AND subscriber_id = $4
	`
	result, err := r.db.Exec(ctx, query,
		req.VerificationStatus, verifiedBy, docID, subscriberID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *KYCRepository) ListBankAccounts(ctx context.Context, subscriberID, companyID int) ([]*domain.SubscriberBank, error) {
	query := `
		SELECT sb.bank_id, sb.subscriber_id, sb.bank_name, sb.account_number,
			sb.ifsc_code, sb.account_holder_name, sb.account_type,
			sb.is_primary, sb.is_verified, sb.verified_date, sb.created_at
		FROM subscriber_banks sb
		INNER JOIN subscribers s ON sb.subscriber_id = s.subscriber_id
		WHERE sb.subscriber_id = $1 AND s.company_id = $2
		ORDER BY sb.is_primary DESC, sb.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, subscriberID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	banks := []*domain.SubscriberBank{}
	for rows.Next() {
		var b domain.SubscriberBank
		if err := rows.Scan(
			&b.BankID, &b.SubscriberID, &b.BankName, &b.AccountNumber,
			&b.IFSCCode, &b.AccountHolderName, &b.AccountType,
			&b.IsPrimary, &b.IsVerified, &b.VerifiedDate, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		banks = append(banks, &b)
	}
	return banks, rows.Err()
}

func (r *KYCRepository) AddBankAccount(ctx context.Context, bank *domain.SubscriberBank) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if bank.IsPrimary {
		tx.Exec(ctx, `UPDATE subscriber_banks SET is_primary = false WHERE subscriber_id = $1`, bank.SubscriberID)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO subscriber_banks (
			subscriber_id, bank_name, account_number, ifsc_code,
			account_holder_name, account_type, is_primary, is_verified
		) VALUES ($1,$2,$3,$4,$5,$6,$7,false)
		RETURNING bank_id, created_at
	`,
		bank.SubscriberID, bank.BankName, bank.AccountNumber, bank.IFSCCode,
		bank.AccountHolderName, bank.AccountType, bank.IsPrimary,
	).Scan(&bank.BankID, &bank.CreatedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *KYCRepository) DeleteBankAccount(ctx context.Context, bankID, subscriberID, companyID int) error {
	query := `
		DELETE FROM subscriber_banks sb
		USING subscribers s
		WHERE sb.bank_id = $1 AND sb.subscriber_id = $2
		  AND s.subscriber_id = sb.subscriber_id AND s.company_id = $3
	`
	result, err := r.db.Exec(ctx, query, bankID, subscriberID, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
