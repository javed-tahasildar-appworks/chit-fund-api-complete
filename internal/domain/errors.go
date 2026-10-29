package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInternalServer    = errors.New("internal server error")
	ErrTenantMismatch    = errors.New("tenant mismatch")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAuctionClosed     = errors.New("auction already closed")
	ErrInvalidBid        = errors.New("invalid bid amount")
	ErrAlreadyBlacklisted = errors.New("subscriber is already blacklisted")
	ErrNotBlacklisted     = errors.New("subscriber is not blacklisted")
	ErrChitGroupClosed    = errors.New("chit group is already closed")
	ErrDefaultNotFound    = errors.New("default record not found")
	ErrPayoutNotPending   = errors.New("payout is not in pending state")
)

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
