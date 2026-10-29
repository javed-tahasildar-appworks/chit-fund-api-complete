package service

import (
	"context"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// List returns paginated notifications for the authenticated user,
// applying the visibility predicate (user_id / role / branch_id / broadcast).
func (s *NotificationService) List(
	ctx context.Context,
	companyID, userID int,
	roleName string,
	branchID *int,
	f domain.NotificationListFilters,
) ([]*domain.StaffNotification, int64, error) {
	return s.repo.List(ctx, companyID, userID, roleName, branchID, f)
}

// UnreadCount returns the badge count for the bell icon.
func (s *NotificationService) UnreadCount(
	ctx context.Context,
	companyID, userID int,
	roleName string,
	branchID *int,
) (int, error) {
	return s.repo.UnreadCount(ctx, companyID, userID, roleName, branchID)
}

// MarkRead marks one notification read and returns the updated object.
func (s *NotificationService) MarkRead(
	ctx context.Context,
	notificationID int64,
	companyID, userID int,
	roleName string,
	branchID *int,
) (*domain.StaffNotification, error) {
	if err := s.repo.MarkRead(ctx, notificationID, companyID, userID, roleName, branchID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, notificationID, companyID, userID, roleName, branchID)
}

// MarkAllRead marks all (or type-scoped) unread notifications read.
func (s *NotificationService) MarkAllRead(
	ctx context.Context,
	companyID, userID int,
	roleName string,
	branchID *int,
	notifType string,
) (int, error) {
	return s.repo.MarkAllRead(ctx, companyID, userID, roleName, branchID, notifType)
}

// Dismiss hides a notification from the user's list.
func (s *NotificationService) Dismiss(
	ctx context.Context,
	notificationID int64,
	companyID, userID int,
	roleName string,
	branchID *int,
) error {
	return s.repo.Dismiss(ctx, notificationID, companyID, userID, roleName, branchID)
}

// Create is called by domain event publishers (not HTTP).
// e.g. after a payout is marked PENDING: notifSvc.Create(ctx, &domain.CreateStaffNotificationRequest{...})
func (s *NotificationService) Create(
	ctx context.Context,
	req *domain.CreateStaffNotificationRequest,
) error {
	return s.repo.Create(ctx, req)
}
