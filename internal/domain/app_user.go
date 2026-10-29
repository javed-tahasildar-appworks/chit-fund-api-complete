package domain

import (
	"encoding/json"
	"time"
)

// AppUser matches 'app_users' table for mobile app
type AppUser struct {
	AppUserID    int        `json:"app_user_id"`
	SubscriberID int        `json:"subscriber_id"`
	Mobile       string     `json:"mobile"`
	Email        *string    `json:"email"`
	PasswordHash string     `json:"-"`
	DeviceToken  *string    `json:"device_token"`
	DeviceType   *string    `json:"device_type"`
	AppVersion   *string    `json:"app_version"`
	IsActive     bool       `json:"is_active"`
	IsVerified   bool       `json:"is_verified"`
	OTPCode      *string    `json:"-"`
	OTPExpiry    *time.Time `json:"-"`
	LastLogin    *time.Time `json:"last_login"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AppNotification matches 'app_notifications' table
type AppNotification struct {
	NotificationID   int             `json:"notification_id"`
	SubscriberID     int             `json:"subscriber_id"`
	Title            string          `json:"title"`
	Message          string          `json:"message"`
	NotificationType *string         `json:"notification_type"`
	IsRead           bool            `json:"is_read"`
	ReadAt           *time.Time      `json:"read_at"`
	Data             json.RawMessage `json:"data"` // JSONB field
	SentAt           time.Time       `json:"sent_at"`
}
