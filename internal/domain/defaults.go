package domain

import "time"

// Default matches 'defaults' table exactly
type Default struct {
	DefaultID               int        `json:"default_id"                 db:"default_id"`
	SubscriberInstallmentID int        `json:"subscriber_installment_id"  db:"subscriber_installment_id"`
	ChitSubscriberID        int        `json:"chit_subscriber_id"         db:"chit_subscriber_id"`
	SubscriberID            int        `json:"subscriber_id"              db:"subscriber_id"`
	ChitID                  int        `json:"chit_id"                    db:"chit_id"`
	DefaultDate             time.Time  `json:"default_date"               db:"default_date"`
	InstallmentDueDate      time.Time  `json:"installment_due_date"       db:"installment_due_date"`
	DaysOverdue             int        `json:"days_overdue"               db:"days_overdue"`
	OutstandingPrincipal    float64    `json:"outstanding_principal"      db:"outstanding_principal"`
	PenaltyAmount           float64    `json:"penalty_amount"             db:"penalty_amount"`
	TotalOutstanding        float64    `json:"total_outstanding"          db:"total_outstanding"`
	ReminderSentCount       int        `json:"reminder_sent_count"        db:"reminder_sent_count"`
	LastReminderDate        *time.Time `json:"last_reminder_date"         db:"last_reminder_date"`
	LegalNoticeSent         bool       `json:"legal_notice_sent"          db:"legal_notice_sent"`
	LegalNoticeDate         *time.Time `json:"legal_notice_date"          db:"legal_notice_date"`
	Status                  string     `json:"status"                     db:"status"`
	ClearedDate             *time.Time `json:"cleared_date"               db:"cleared_date"`
	Remarks                 *string    `json:"remarks"                    db:"remarks"`
	CreatedAt               time.Time  `json:"created_at"                 db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"                 db:"updated_at"`

	// Joined fields
	SubscriberName *string `json:"subscriber_name,omitempty" db:"subscriber_name"`
	ChitCode       *string `json:"chit_code,omitempty"       db:"chit_code"`
}

// LegalNotice matches 'legal_notices' table
// LegalNotice matches 'legal_notices' table exactly
type LegalNotice struct {
	NoticeID          int        `json:"notice_id"          db:"notice_id"`
	DefaultID         *int       `json:"default_id"         db:"default_id"`
	SubscriberID      int        `json:"subscriber_id"      db:"subscriber_id"`
	ChitID            int        `json:"chit_id"            db:"chit_id"`
	NoticeDate        time.Time  `json:"notice_date"        db:"notice_date"`
	NoticeType        *string    `json:"notice_type"        db:"notice_type"`
	NoticeNumber      *string    `json:"notice_number"      db:"notice_number"`
	OutstandingAmount float64    `json:"outstanding_amount" db:"outstanding_amount"`
	ResponseDeadline  *time.Time `json:"response_deadline"  db:"response_deadline"`
	DeliveryMode      *string    `json:"delivery_mode"      db:"delivery_mode"`
	TrackingNumber    *string    `json:"tracking_number"    db:"tracking_number"`
	DeliveredDate     *time.Time `json:"delivered_date"     db:"delivered_date"`
	ResponseReceived  bool       `json:"response_received"  db:"response_received"`
	ResponseDate      *time.Time `json:"response_date"      db:"response_date"`
	ResponseDetails   *string    `json:"response_details"   db:"response_details"`
	DocumentPath      *string    `json:"document_path"      db:"document_path"`
	IssuedBy          *int       `json:"issued_by"          db:"issued_by"`
	Status            string     `json:"status"             db:"status"`
	Remarks           *string    `json:"remarks"            db:"remarks"`
	CreatedAt         time.Time  `json:"created_at"         db:"created_at"`

	// Joined / virtual — not in legal_notices table
	NoticeSentVia *string `json:"notice_sent_via,omitempty"  db:"-"`
	NoticeContent *string `json:"notice_content,omitempty"   db:"-"`
	CompanyID     int     `json:"company_id,omitempty"       db:"-"`
}

// AuditLog matches 'audit_logs' table
type AuditLog struct {
	LogID        int64     `json:"log_id"      db:"log_id"`
	UserID       *int      `json:"user_id"     db:"user_id"`
	SubscriberID *int      `json:"subscriber_id" db:"subscriber_id"`
	Action       string    `json:"action"      db:"action"`
	EntityName   string    `json:"entity_name" db:"entity_name"`
	EntityID     *int      `json:"entity_id"   db:"entity_id"`
	OldValues    []byte    `json:"old_values"  db:"old_values"`
	NewValues    []byte    `json:"new_values"  db:"new_values"`
	IPAddress    *string   `json:"ip_address"  db:"ip_address"`
	UserAgent    *string   `json:"user_agent"  db:"user_agent"`
	Timestamp    time.Time `json:"timestamp"   db:"timestamp"`
}

type SystemLog struct {
	LogID      int64     `json:"log_id"      db:"log_id"`
	LogLevel   *string   `json:"log_level"   db:"log_level"`
	Module     *string   `json:"module"      db:"module"`
	Message    string    `json:"message"     db:"message"`
	StackTrace *string   `json:"stack_trace" db:"stack_trace"`
	UserID     *int      `json:"user_id"     db:"user_id"`
	IPAddress  *string   `json:"ip_address"  db:"ip_address"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}
