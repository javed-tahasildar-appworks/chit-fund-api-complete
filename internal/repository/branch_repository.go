package repository

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

// ─── Branch Repository ────────────────────────────────────────────────────────

type BranchRepository struct {
	db *database.PostgresPool
}

func NewBranchRepository(db *database.PostgresPool) *BranchRepository {
	return &BranchRepository{db: db}
}

// scanBranch scans a branch row — columns must match the SELECT list exactly
func scanBranch(rows interface{ Scan(...any) error }) (domain.Branch, error) {
	var b domain.Branch
	var managerID *int
	var contactPhone *string
	var city, pincode *string
	err := rows.Scan(
		&b.BranchID, &b.CompanyID, &b.BranchName, &b.BranchCode,
		&b.Address, &city, &pincode, &contactPhone,
		&managerID, &b.IsActive, &b.CreatedAt, &b.UpdatedAt,
	)
	if city != nil {
		b.City = *city
	}
	if pincode != nil {
		b.Pincode = *pincode
	}
	if contactPhone != nil {
		b.ContactPhone = *contactPhone
	}
	if managerID != nil {
		b.ManagerUserID = *managerID
	}
	return b, err
}

const branchSelect = `
	SELECT
		branch_id,
		company_id,
		branch_name,
		branch_code,
		COALESCE(address, '')                                                AS address,
		COALESCE(city, '')                                                   AS city,
		COALESCE(pincode, '')                                                AS pincode,
		COALESCE(contact_phone, '')                                          AS contact_phone,
		manager_user_id,
		is_active,
		TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
		TO_CHAR(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
	FROM branches`

func (r *BranchRepository) Create(ctx context.Context, branch *domain.Branch) error {
	// Actual DB columns: company_id, branch_code, branch_name, address, city,
	//                    pincode, contact_phone, is_active
	query := `
		INSERT INTO branches (
			company_id, branch_code, branch_name, address,
			city, pincode, contact_phone, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,true)
		RETURNING branch_id,
		  TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		  TO_CHAR(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`
	return r.db.QueryRow(ctx, query,
		branch.CompanyID, branch.BranchCode, branch.BranchName,
		branch.Address, branch.City, branch.Pincode, branch.ContactPhone,
	).Scan(&branch.BranchID, &branch.CreatedAt, &branch.UpdatedAt)
}

func (r *BranchRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Branch, error) {
	query := branchSelect + ` WHERE branch_id = $1 AND company_id = $2`
	row := r.db.QueryRow(ctx, query, id, companyID)
	var b domain.Branch
	var managerID *int
	var city, pincode, contactPhone *string
	err := row.Scan(
		&b.BranchID, &b.CompanyID, &b.BranchName, &b.BranchCode,
		&b.Address, &city, &pincode, &contactPhone,
		&managerID, &b.IsActive, &b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if city != nil {
		b.City = *city
	}
	if pincode != nil {
		b.Pincode = *pincode
	}
	if contactPhone != nil {
		b.ContactPhone = *contactPhone
	}
	if managerID != nil {
		b.ManagerUserID = *managerID
	}
	return &b, nil
}

func (r *BranchRepository) List(ctx context.Context, companyID int) ([]*domain.Branch, error) {
	query := branchSelect + ` WHERE company_id = $1 ORDER BY branch_name ASC`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	branches := []*domain.Branch{}
	for rows.Next() {
		var b domain.Branch
		var managerID *int
		var city, pincode, contactPhone *string
		if err := rows.Scan(
			&b.BranchID, &b.CompanyID, &b.BranchName, &b.BranchCode,
			&b.Address, &city, &pincode, &contactPhone,
			&managerID, &b.IsActive, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if city != nil {
			b.City = *city
		}
		if pincode != nil {
			b.Pincode = *pincode
		}
		if contactPhone != nil {
			b.ContactPhone = *contactPhone
		}
		if managerID != nil {
			b.ManagerUserID = *managerID
		}
		branches = append(branches, &b)
	}
	return branches, rows.Err()
}

func (r *BranchRepository) Update(ctx context.Context, branchID, companyID int, req *domain.UpdateBranchRequest) error {
	// Only update columns that exist in the actual DB
	query := `
		UPDATE branches SET
			branch_name   = COALESCE($1, branch_name),
			address       = COALESCE($2, address),
			city          = COALESCE($3, city),
			contact_phone = COALESCE($4, contact_phone),
			manager_user_id = COALESCE($5, manager_user_id),
			updated_at    = NOW()
		WHERE branch_id = $6 AND company_id = $7
	`
	result, err := r.db.Exec(ctx, query,
		req.BranchName, req.Address, req.City, req.Phone,
		req.ManagerUserID, branchID, companyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ─── State Repository ─────────────────────────────────────────────────────────

type StateRepository struct {
	db *database.PostgresPool
}

func NewStateRepository(db *database.PostgresPool) *StateRepository {
	return &StateRepository{db: db}
}

func (r *StateRepository) List(ctx context.Context) ([]*domain.State, error) {
	query := `
		SELECT
			state_id,
			state_name,
			state_code,
			COALESCE(max_discount_percent, 0)          AS max_discount_percent,
			COALESCE(min_foreman_security_percent, 0)   AS min_foreman_security_percent,
			COALESCE(penalty_interest_percent, 0)       AS penalty_interest_percent,
			COALESCE(gst_rate, 0)                       AS gst_rate,
			is_active,
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			TO_CHAR(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
		FROM states
		WHERE is_active = true
		ORDER BY state_name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := []*domain.State{}
	for rows.Next() {
		var s domain.State
		if err := rows.Scan(
			&s.StateID, &s.StateName, &s.StateCode,
			&s.MaxDiscountPercent, &s.MinForemanSecurityPercent,
			&s.PenaltyInterestPercent, &s.GSTRate,
			&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, &s)
	}
	return states, rows.Err()
}

func (r *StateRepository) GetByID(ctx context.Context, id int) (*domain.State, error) {
	query := `
		SELECT
			state_id,
			state_name,
			state_code,
			COALESCE(max_discount_percent, 0)          AS max_discount_percent,
			COALESCE(min_foreman_security_percent, 0)   AS min_foreman_security_percent,
			COALESCE(penalty_interest_percent, 0)       AS penalty_interest_percent,
			COALESCE(gst_rate, 0)                       AS gst_rate,
			is_active,
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			TO_CHAR(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS updated_at
		FROM states
		WHERE state_id = $1
	`
	var s domain.State
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.StateID, &s.StateName, &s.StateCode,
		&s.MaxDiscountPercent, &s.MinForemanSecurityPercent,
		&s.PenaltyInterestPercent, &s.GSTRate,
		&s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ─── Role Repository ──────────────────────────────────────────────────────────

type RoleRepository struct {
	db *database.PostgresPool
}

func NewRoleRepository(db *database.PostgresPool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) List(ctx context.Context, companyID int) ([]*domain.Role, error) {
	query := `
		SELECT role_id, role_name, role_description, permissions, created_at
		FROM roles ORDER BY role_name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.Role])
}

func (r *RoleRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Role, error) {
	query := `
		SELECT role_id, role_name, role_description, permissions, created_at
		FROM roles WHERE role_id = $1
	`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	role, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Role])
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &role, err
}

func (r *RoleRepository) UpdatePermissions(ctx context.Context, roleID, companyID int, permissions interface{}) error {
	query := `
		UPDATE roles SET permissions = $1, updated_at = NOW()
		WHERE role_id = $2 AND company_id = $3 AND is_system = false
	`
	result, err := r.db.Exec(ctx, query, permissions, roleID, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("role not found or is a system role")
	}
	return nil
}
