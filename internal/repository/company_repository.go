package repository

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type CompanyRepository struct {
	db *database.PostgresPool
}

func NewCompanyRepository(db *database.PostgresPool) *CompanyRepository {
	return &CompanyRepository{db: db}
}

func (r *CompanyRepository) Create(ctx context.Context, company *domain.Company) error {
	query := `
		INSERT INTO companies (
			company_name, registration_number, pan_number, gstin, state_id,
			registered_address, office_address, city, pincode,
			contact_email, contact_phone, foreman_commission_percent,
			minimum_capital_lakhs, status, license_number
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING company_id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		company.CompanyName, company.RegistrationNumber, company.PANNumber,
		company.GSTIN, company.StateID, company.RegisteredAddress,
		company.OfficeAddress, company.City, company.Pincode,
		company.ContactEmail, company.ContactPhone,
		company.ForemanCommissionPercent, company.MinimumCapitalLakhs,
		company.Status, company.LicenseNumber,
	).Scan(&company.CompanyID, &company.CreatedAt, &company.UpdatedAt)
}

func (r *CompanyRepository) GetByID(ctx context.Context, id int) (*domain.Company, error) {
	query := `
		SELECT company_id, company_name, registration_number, pan_number, gstin,
			state_id, registered_address, office_address, city, pincode,
			contact_email, contact_phone, foreman_commission_percent,
			minimum_capital_lakhs, status, license_number, license_valid_till,
			created_at, updated_at
		FROM companies WHERE company_id = $1
	`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	c, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Company])
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &c, err
}

func (r *CompanyRepository) GetProfile(ctx context.Context, companyID int) (*domain.Company, error) {
	return r.GetByID(ctx, companyID)
}

func (r *CompanyRepository) Update(ctx context.Context, companyID int, req *domain.UpdateCompanyRequest) error {
	query := `
		UPDATE companies SET
			company_name   = COALESCE($1, company_name),
			contact_email  = COALESCE($2, contact_email),
			contact_phone  = COALESCE($3, contact_phone),
			office_address = COALESCE($4, office_address),
			city           = COALESCE($5, city),
			pincode        = COALESCE($6, pincode),
			foreman_commission_percent = COALESCE($7, foreman_commission_percent),
			status         = COALESCE($8, status),
			updated_at     = NOW()
		WHERE company_id = $9
	`
	result, err := r.db.Exec(ctx, query,
		req.CompanyName, req.ContactEmail, req.ContactPhone,
		req.OfficeAddress, req.City, req.Pincode,
		req.ForemanCommissionPercent, req.Status, companyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CompanyRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.Company, int64, error) {
	base := `SELECT company_id, company_name, registration_number, pan_number, gstin,
		state_id, registered_address, contact_email, contact_phone,
		foreman_commission_percent, status, license_number, license_valid_till,
		created_at, updated_at FROM companies`
	count := `SELECT COUNT(*) FROM companies`

	args := []interface{}{}
	where := ""
	argPos := 1

	if status, ok := filters["status"].(string); ok && status != "" {
		where = fmt.Sprintf(" WHERE status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, count+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}

	query := base + where + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	companies, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.Company])
	return companies, total, err
}
