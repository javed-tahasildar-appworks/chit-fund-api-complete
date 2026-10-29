package repository

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *database.PostgresPool
}

func NewUserRepository(db *database.PostgresPool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT u.user_id, u.company_id, u.branch_id, u.role_id,
			u.username, u.email, u.mobile, u.password_hash,
			u.full_name, u.employee_code, u.status,
			u.last_login, u.failed_login_attempts, u.is_locked,
			u.created_at, u.updated_at, r.role_name, r.permissions
		FROM users u INNER JOIN roles r ON u.role_id = r.role_id
		WHERE u.username = $1 AND u.status = 'ACTIVE'
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.UserID, &user.CompanyID, &user.BranchID, &user.RoleID,
		&user.Username, &user.Email, &user.Mobile, &user.PasswordHash,
		&user.FullName, &user.EmployeeCode, &user.Status,
		&user.LastLogin, &user.FailedLoginAttempts, &user.IsLocked,
		&user.CreatedAt, &user.UpdatedAt, &user.RoleName, &user.Permissions,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id, companyID int) (*domain.User, error) {
	query := `
		SELECT u.user_id, u.company_id, u.branch_id, u.role_id,
			u.username, u.email, u.mobile, u.full_name,
			u.employee_code, u.status, u.last_login,
			u.failed_login_attempts, u.is_locked, u.created_at, u.updated_at,
			r.role_name
		FROM users u INNER JOIN roles r ON u.role_id = r.role_id
		WHERE u.user_id = $1 AND u.company_id = $2
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id, companyID).Scan(
		&user.UserID, &user.CompanyID, &user.BranchID, &user.RoleID,
		&user.Username, &user.Email, &user.Mobile, &user.FullName,
		&user.EmployeeCode, &user.Status, &user.LastLogin,
		&user.FailedLoginAttempts, &user.IsLocked, &user.CreatedAt, &user.UpdatedAt,
		&user.RoleName,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &user, err
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			company_id, branch_id, role_id, username, email, mobile,
			password_hash, full_name, employee_code, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'ACTIVE')
		RETURNING user_id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		user.CompanyID, user.BranchID, user.RoleID,
		user.Username, user.Email, user.Mobile,
		user.PasswordHash, user.FullName, user.EmployeeCode,
	).Scan(&user.UserID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) Update(ctx context.Context, userID, companyID int, req *domain.UpdateUserRequest) error {
	query := `
		UPDATE users SET
			full_name     = COALESCE($1, full_name),
			email         = COALESCE($2, email),
			mobile        = COALESCE($3, mobile),
			role_id       = COALESCE($4, role_id),
			branch_id     = COALESCE($5, branch_id),
			employee_code = COALESCE($6, employee_code),
			updated_at    = NOW()
		WHERE user_id = $7 AND company_id = $8
	`
	result, err := r.db.Exec(ctx, query,
		req.FullName, req.Email, req.Mobile,
		req.RoleID, req.BranchID, req.EmployeeCode,
		userID, companyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) SetStatus(ctx context.Context, userID, companyID int, status string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE users SET status = $1, updated_at = NOW() WHERE user_id = $2 AND company_id = $3`,
		status, userID, companyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID int, newHash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE user_id = $2`,
		newHash, userID,
	)
	return err
}

func (r *UserRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.User, int64, error) {
	base := `SELECT u.user_id, u.company_id, u.branch_id, u.role_id,
		u.username, u.email, u.mobile, u.full_name, u.employee_code, u.status,
		u.last_login, u.failed_login_attempts, u.is_locked, u.created_at, u.updated_at,
		r.role_name FROM users u INNER JOIN roles r ON u.role_id = r.role_id`

	args := []interface{}{companyID}
	where := " WHERE u.company_id = $1"
	argPos := 2

	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND u.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if branchID, ok := filters["branch_id"].(int); ok && branchID > 0 {
		where += fmt.Sprintf(" AND u.branch_id = $%d", argPos)
		args = append(args, branchID)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users u`+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY u.full_name ASC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.UserID, &u.CompanyID, &u.BranchID, &u.RoleID,
			&u.Username, &u.Email, &u.Mobile, &u.FullName,
			&u.EmployeeCode, &u.Status, &u.LastLogin,
			&u.FailedLoginAttempts, &u.IsLocked, &u.CreatedAt, &u.UpdatedAt,
			&u.RoleName,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	return users, total, nil
}
