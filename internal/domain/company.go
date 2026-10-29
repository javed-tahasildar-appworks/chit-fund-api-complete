package domain

import "time"

// Company matches your 'companies' table exactly
type Company struct {
	CompanyID                int        `json:"company_id"`
	CompanyName              string     `json:"company_name"`
	RegistrationNumber       string     `json:"registration_number"`
	PANNumber                *string    `json:"pan_number"`
	GSTIN                    *string    `json:"gstin"`
	StateID                  *int       `json:"state_id"`
	RegisteredAddress        string     `json:"registered_address"`
	OfficeAddress            *string    `json:"office_address"`
	City                     *string    `json:"city"`
	Pincode                  *string    `json:"pincode"`
	ContactEmail             *string    `json:"contact_email"`
	ContactPhone             *string    `json:"contact_phone"`
	ForemanCommissionPercent float64    `json:"foreman_commission_percent"`
	MinimumCapitalLakhs      float64    `json:"minimum_capital_lakhs"`
	Status                   string     `json:"status"`
	LicenseNumber            *string    `json:"license_number"`
	LicenseValidTill         *time.Time `json:"license_valid_till"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}
