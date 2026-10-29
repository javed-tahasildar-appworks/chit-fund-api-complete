package domain

import (
	"encoding/json"
	"time"
)

// Branch matches actual 'branches' table columns exactly.
// DB columns: branch_id, company_id, branch_name, branch_code, address,
//
//	city, pincode, contact_phone, manager_user_id, is_active,
//	created_at, updated_at
type Branch struct {
	BranchID      int    `json:"branch_id"`
	CompanyID     int    `json:"company_id"`
	BranchName    string `json:"branch_name"`
	BranchCode    string `json:"branch_code"`
	Address       string `json:"address"`
	City          string `json:"city"`
	Pincode       string `json:"pincode"`
	ContactPhone  string `json:"contact_phone"`
	ManagerUserID int    `json:"manager_user_id"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// State matches actual 'states' table columns exactly.
// DB columns: state_id, state_name, state_code, max_discount_percent,
//
//	min_foreman_security_percent, penalty_interest_percent,
//	gst_rate, is_active, created_at, updated_at
type State struct {
	StateID                   int     `json:"state_id"`
	StateName                 string  `json:"state_name"`
	StateCode                 string  `json:"state_code"`
	MaxDiscountPercent        float64 `json:"max_discount_percent"`
	MinForemanSecurityPercent float64 `json:"min_foreman_security_percent"`
	PenaltyInterestPercent    float64 `json:"penalty_interest_percent"`
	GSTRate                   float64 `json:"gst_rate"`
	IsActive                  bool    `json:"is_active"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
}

// Role matches 'roles' table
type Role struct {
	RoleID          int             `json:"role_id"          db:"role_id"`
	RoleName        string          `json:"role_name"        db:"role_name"`
	RoleDescription *string         `json:"description"      db:"role_description"`
	Permissions     json.RawMessage `json:"permissions"      db:"permissions"`
	CreatedAt       time.Time       `json:"created_at"       db:"created_at"`
}
