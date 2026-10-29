package service

import (
	"context"
	"fmt"
	"time"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
	"chit-fund-api/pkg/cache"
)

type AuctionService struct {
	repo     *repository.AuctionRepository
	chitRepo *repository.ChitGroupRepository
	cache    *cache.RedisClient
}

func NewAuctionService(repo *repository.AuctionRepository, chitRepo *repository.ChitGroupRepository, cache *cache.RedisClient) *AuctionService {
	return &AuctionService{
		repo:     repo,
		chitRepo: chitRepo,
		cache:    cache,
	}
}

func (s *AuctionService) Create(ctx context.Context, chitID, companyID, userID int, auctionDate time.Time, auctionTime, venue string) (*domain.Auction, error) {
	// Verify chit exists and belongs to company
	chit, err := s.chitRepo.GetByID(ctx, chitID, companyID)
	if err != nil {
		return nil, err
	}

	if chit.Status != "ACTIVE" && chit.Status != "PLANNED" {
		return nil, domain.NewAppError("INVALID_STATUS", "Chit must be ACTIVE or PLANNED to create auction", nil)
	}

	monthNumber := chit.CurrentMonth + 1

	// Check if an auction already exists for this chit + month
	existing, err := s.repo.GetByChitAndMonth(ctx, chitID, monthNumber)
	if err != nil {
		return nil, domain.NewAppError("LOOKUP_FAILED", "Failed to check existing auctions", err)
	}
	if existing != nil {
		// Return a clear error with the existing auction_id so the caller can use PUT to update it
		return nil, domain.NewAppError(
			"DUPLICATE_AUCTION",
			fmt.Sprintf(
				"An auction already exists for month %d of this chit group (auction_id: %d, status: %s). Use PUT /auctions/%d to update it instead.",
				monthNumber, existing.AuctionID, existing.Status, existing.AuctionID,
			),
			nil,
		)
	}

	auctionMethod := "OPEN_BID"
	auctionMethodPtr := &auctionMethod
	userIDPtr := &userID

	var auctionTimePtr *string
	if auctionTime != "" {
		auctionTimePtr = &auctionTime
	}

	var venuePtr *string
	if venue != "" {
		venuePtr = &venue
	}

	auction := &domain.Auction{
		ChitID:             chitID,
		MonthNumber:        monthNumber,
		AuctionDate:        auctionDate,
		AuctionTime:        auctionTimePtr,
		Venue:              venuePtr,
		MaxDiscountAllowed: chit.MaxDiscountPercent,
		AuctionMethod:      auctionMethodPtr,
		Status:             "SCHEDULED",
		ConductedBy:        userIDPtr,
	}

	if err := s.repo.Create(ctx, auction); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to create auction", err)
	}

	return auction, nil
}

func (s *AuctionService) GetByID(ctx context.Context, id, companyID int) (*domain.Auction, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *AuctionService) PlaceBid(ctx context.Context, auctionID, companyID int, req *domain.PlaceBidRequest) error {
	// Verify auction exists and is open
	auction, err := s.repo.GetByID(ctx, auctionID, companyID)
	if err != nil {
		return err
	}

	if auction.Status != "SCHEDULED" && auction.Status != "IN_PROGRESS" {
		return domain.ErrAuctionClosed
	}

	// Verify bid is within limits
	if req.BidDiscountPercent > auction.MaxDiscountAllowed {
		return domain.ErrInvalidBid
	}

	// Look up the real subscriber_id from chit_subscriber_id
	subscriberID, err := s.repo.GetSubscriberIDByChitSubscriberID(ctx, req.ChitSubscriberID, auction.ChitID)
	if err != nil {
		return domain.NewAppError("INVALID_SUBSCRIBER", "chit_subscriber_id not found or not enrolled in this chit", err)
	}

	bid := &domain.AuctionBid{
		AuctionID:          auctionID,
		ChitSubscriberID:   req.ChitSubscriberID,
		SubscriberID:       subscriberID,
		BidDiscountAmount:  req.BidDiscountAmount,
		BidDiscountPercent: req.BidDiscountPercent,
	}

	return s.repo.PlaceBid(ctx, bid)
}

func (s *AuctionService) Finalize(ctx context.Context, auctionID, companyID, userID int, req *domain.FinalizeAuctionRequest) error {
	// Verify auction exists
	auction, err := s.repo.GetByID(ctx, auctionID, companyID)
	if err != nil {
		return err
	}

	if auction.Status == "COMPLETED" {
		return domain.NewAppError("ALREADY_COMPLETED", "Auction already finalized", nil)
	}

	// Finalize auction with transaction
	if err := s.repo.Finalize(ctx, auctionID, req.WinningBidID, userID); err != nil {
		return domain.NewAppError("FINALIZE_FAILED", "Failed to finalize auction", err)
	}

	// Invalidate cache
	s.cache.Delete(ctx, fmt.Sprintf("auction:%d", auctionID))

	return nil
}

func (s *AuctionService) GetBids(ctx context.Context, auctionID, companyID int) ([]*domain.AuctionBid, error) {
	// Verify auction belongs to company
	if _, err := s.repo.GetByID(ctx, auctionID, companyID); err != nil {
		return nil, err
	}

	return s.repo.GetBids(ctx, auctionID)
}

// Update updates auction scheduling details
func (s *AuctionService) Update(ctx context.Context, id, companyID int, req *domain.UpdateAuctionRequest) error {
	return s.repo.Update(ctx, id, companyID, req)
}

// Conduct marks an auction as IN_PROGRESS / conducted
func (s *AuctionService) Conduct(ctx context.Context, id, companyID, userID int) error {
	auction, err := s.repo.GetByID(ctx, id, companyID)
	if err != nil {
		return err
	}
	if auction.Status != "SCHEDULED" {
		return domain.NewAppError("INVALID_STATUS", "Auction must be in SCHEDULED state to conduct", nil)
	}
	return s.repo.SetStatus(ctx, id, "IN_PROGRESS", userID)
}

// Cancel cancels a SCHEDULED or IN_PROGRESS auction
func (s *AuctionService) Cancel(ctx context.Context, id, companyID, userID int, remarks string) error {
	return s.repo.Cancel(ctx, id, companyID, userID, remarks)
}

// List returns paginated auctions
func (s *AuctionService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Auction, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}
