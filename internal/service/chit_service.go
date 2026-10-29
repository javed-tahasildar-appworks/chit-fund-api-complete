package service

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
	"chit-fund-api/pkg/cache"

	"github.com/google/uuid"
)

type ChitGroupService struct {
	repo  *repository.ChitGroupRepository
	cache *cache.RedisClient
}

func NewChitGroupService(repo *repository.ChitGroupRepository, cache *cache.RedisClient) *ChitGroupService {
	return &ChitGroupService{
		repo:  repo,
		cache: cache,
	}
}

func (s *ChitGroupService) Create(ctx context.Context, req *domain.CreateChitGroupRequest, companyID, branchID, userID int) (*domain.ChitGroup, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, domain.NewAppError("INVALID_DATE", "Invalid start date format", err)
	}

	// Calculate installment amount
	installmentAmount := req.ChitValue / float64(req.TotalSubscribers)

	// Generate unique chit code
	chitCode := fmt.Sprintf("CHT-%s-%s", time.Now().Format("2006"), uuid.New().String()[:8])

	// ✅ FIXED: Convert values to pointers for nullable fields
	branchIDPtr := &branchID
	auctionDayPtr := &req.AuctionDayOfMonth
	auctionTimePtr := &req.AuctionTime
	userIDPtr := &userID

	chit := &domain.ChitGroup{
		CompanyID:                companyID,
		BranchID:                 branchIDPtr, // ✅ Pointer
		ChitCode:                 chitCode,
		ChitName:                 req.ChitName,
		ChitValue:                req.ChitValue,
		TotalSubscribers:         req.TotalSubscribers,
		InstallmentAmount:        installmentAmount,
		TotalMonths:              req.TotalMonths,
		StartDate:                startDate,
		AuctionDayOfMonth:        auctionDayPtr,  // ✅ Pointer
		AuctionTime:              auctionTimePtr, // ✅ Pointer
		AuctionType:              req.AuctionType,
		InstallmentType:          "VARIABLE",
		ForemanCommissionPercent: req.ForemanCommissionPercent,
		MaxDiscountPercent:       req.MaxDiscountPercent,
		SecurityDeposit:          0,
		Status:                   "PLANNED",
		CurrentMonth:             0,
		CreatedBy:                userIDPtr, // ✅ Pointer
	}

	if err := s.repo.Create(ctx, chit); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to create chit group", err)
	}

	// Invalidate cache
	s.cache.Delete(ctx, fmt.Sprintf("chits:company:%d", companyID))

	return chit, nil
}

func (s *ChitGroupService) GetByID(ctx context.Context, id, companyID int) (*domain.ChitGroup, error) {
	cacheKey := fmt.Sprintf("chit:%d", id)

	var chit domain.ChitGroup
	if err := s.cache.Get(ctx, cacheKey, &chit); err == nil {
		if chit.CompanyID == companyID {
			return &chit, nil
		}
	}

	chitPtr, err := s.repo.GetByID(ctx, id, companyID)
	if err != nil {
		return nil, err
	}

	s.cache.Set(ctx, cacheKey, chitPtr, 15*time.Minute)

	return chitPtr, nil
}

func (s *ChitGroupService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.ChitGroup, error) {
	return s.repo.List(ctx, companyID, filters)
}

func (s *ChitGroupService) AddSubscriber(ctx context.Context, chitID, subscriberID, companyID int, ticketCount int, suretyRequired bool) (*domain.ChitSubscriber, error) {
	// Verify chit exists and belongs to company
	chit, err := s.repo.GetByID(ctx, chitID, companyID)
	if err != nil {
		return nil, err
	}

	if chit.Status != "PLANNED" && chit.Status != "ACTIVE" {
		return nil, domain.NewAppError("INVALID_STATUS", "Cannot add subscribers to closed chit", nil)
	}

	cs := &domain.ChitSubscriber{
		ChitID:         chitID,
		SubscriberID:   subscriberID,
		JoinDate:       time.Now(),
		TicketCount:    ticketCount,
		SuretyRequired: suretyRequired,
		Status:         "ACTIVE",
		EnrollmentFee:  0,
	}

	if err := s.repo.AddSubscriber(ctx, cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (s *ChitGroupService) Update(ctx context.Context, chit *domain.ChitGroup) error {
	return s.repo.Update(ctx, chit)
}

func (s *ChitGroupService) GetSubscribers(ctx context.Context, chitID, companyID int) ([]*domain.ChitSubscriber, error) {
	return s.repo.GetSubscribers(ctx, chitID, companyID)
}

// RemoveEnrollment drops an active chit-subscriber enrollment
func (s *ChitGroupService) RemoveEnrollment(ctx context.Context, chitID, csID, companyID int) error {
	if _, err := s.repo.GetByID(ctx, chitID, companyID); err != nil {
		return err
	}
	return s.repo.RemoveEnrollment(ctx, csID, chitID)
}
