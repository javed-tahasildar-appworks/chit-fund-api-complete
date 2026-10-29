package domain

import (
	"encoding/json"
	"time"
)

// User matches your 'users' table with role permissions
type User struct {
	UserID              int        `json:"user_id"`
	CompanyID           int        `json:"company_id"`
	BranchID            *int       `json:"branch_id"`
	RoleID              int        `json:"role_id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	Mobile              string     `json:"mobile"`
	PasswordHash        string     `json:"-"` // Never expose in JSON
	FullName            string     `json:"full_name"`
	EmployeeCode        *string    `json:"employee_code"`
	Status              string     `json:"status"`
	LastLogin           *time.Time `json:"last_login"`
	FailedLoginAttempts int        `json:"failed_login_attempts"`
	IsLocked            bool       `json:"is_locked"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Joined data from roles table
	RoleName    string          `json:"role_name,omitempty"`
	Permissions json.RawMessage `json:"permissions,omitempty"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID    int    `json:"user_id"`
	CompanyID int    `json:"company_id"`
	BranchID  *int   `json:"branch_id,omitempty"`
	RoleID    int    `json:"role_id"`
	RoleName  string `json:"role_name"`
}
