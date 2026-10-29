package repository

import (
	"context"
	"fmt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/database"

	"github.com/jackc/pgx/v5"
)

type AuctionRepository struct {
	db *database.PostgresPool
}

func NewAuctionRepository(db *database.PostgresPool) *AuctionRepository {
	return &AuctionRepository{db: db}
}

// GetByChitAndMonth returns an existing auction for a given chit + month, or nil if none exists
func (r *AuctionRepository) GetByChitAndMonth(ctx context.Context, chitID, monthNumber int) (*domain.Auction, error) {
	query := `
		SELECT auction_id, chit_id, month_number, auction_date, auction_time,
			venue, max_discount_allowed, winning_bid_id, auction_method,
			status, conducted_by, remarks, created_at, updated_at
		FROM auctions
		WHERE chit_id = $1 AND month_number = $2
		LIMIT 1
	`
	var a domain.Auction
	var auctionTime, venue, auctionMethod, remarks *string
	var winningBidID, conductedBy *int
	err := r.db.QueryRow(ctx, query, chitID, monthNumber).Scan(
		&a.AuctionID, &a.ChitID, &a.MonthNumber, &a.AuctionDate, &auctionTime,
		&venue, &a.MaxDiscountAllowed, &winningBidID, &auctionMethod,
		&a.Status, &conductedBy, &remarks, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil // no existing auction — safe to create
	}
	if err != nil {
		return nil, err
	}
	a.AuctionTime = auctionTime
	a.Venue = venue
	a.AuctionMethod = auctionMethod
	a.Remarks = remarks
	a.WinningBidID = winningBidID
	a.ConductedBy = conductedBy
	return &a, nil
}

func (r *AuctionRepository) Create(ctx context.Context, auction *domain.Auction) error {
	// auction_time is PostgreSQL TIME type — must be cast explicitly from string.
	// Using $4::time handles "13:30:00" string → TIME correctly.
	// auction_date is PostgreSQL DATE type — cast from Go time.Time.
	query := `
		INSERT INTO auctions (
			chit_id, month_number, auction_date, auction_time, venue,
			max_discount_allowed, auction_method, status, conducted_by
		) VALUES ($1, $2, $3::date, $4::time, $5, $6, $7, $8, $9)
		RETURNING auction_id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		auction.ChitID, auction.MonthNumber, auction.AuctionDate, auction.AuctionTime,
		auction.Venue, auction.MaxDiscountAllowed, auction.AuctionMethod, auction.Status,
		auction.ConductedBy,
	).Scan(&auction.AuctionID, &auction.CreatedAt, &auction.UpdatedAt)
}

func (r *AuctionRepository) GetByID(ctx context.Context, id, companyID int) (*domain.Auction, error) {
	query := `
		SELECT
			a.auction_id, a.chit_id,
			COALESCE(cg.chit_code, '')         AS chit_code,
			COALESCE(cg.chit_name, '')         AS chit_name,
			cg.chit_value,
			a.month_number, a.auction_date, a.auction_time,
			a.venue, a.max_discount_allowed, a.winning_bid_id, a.auction_method,
			a.status, a.conducted_by,
			COALESCE(u.full_name, '')           AS conducted_by_name,
			COALESCE(s.full_name, '')           AS winner_name,
			a.remarks, a.created_at, a.updated_at
		FROM auctions a
		INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id
		LEFT  JOIN users u         ON a.conducted_by   = u.user_id
		LEFT  JOIN auction_bids ab ON ab.bid_id         = a.winning_bid_id
		LEFT  JOIN subscribers s   ON ab.subscriber_id  = s.subscriber_id
		WHERE a.auction_id = $1 AND cg.company_id = $2
	`
	var a domain.Auction
	err := r.db.QueryRow(ctx, query, id, companyID).Scan(
		&a.AuctionID, &a.ChitID, &a.ChitCode, &a.ChitName, &a.ChitValue,
		&a.MonthNumber, &a.AuctionDate, &a.AuctionTime,
		&a.Venue, &a.MaxDiscountAllowed, &a.WinningBidID, &a.AuctionMethod,
		&a.Status, &a.ConductedBy, &a.ConductedByName, &a.WinnerName,
		&a.Remarks, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AuctionRepository) GetSubscriberIDByChitSubscriberID(ctx context.Context, chitSubscriberID, chitID int) (int, error) {
	var subscriberID int
	err := r.db.QueryRow(ctx,
		`SELECT subscriber_id FROM chit_subscribers
		 WHERE chit_subscriber_id = $1 AND chit_id = $2 AND status = 'ACTIVE'`,
		chitSubscriberID, chitID,
	).Scan(&subscriberID)
	if err == pgx.ErrNoRows {
		return 0, domain.ErrNotFound
	}
	return subscriberID, err
}

func (r *AuctionRepository) PlaceBid(ctx context.Context, bid *domain.AuctionBid) error {
	query := `
		INSERT INTO auction_bids (
			auction_id, chit_subscriber_id, subscriber_id, bid_discount_amount,
			bid_discount_percent, bid_time, is_winner, is_valid
		) VALUES ($1, $2, $3, $4, $5, NOW(), false, true)
		RETURNING bid_id, bid_time, created_at
	`

	return r.db.QueryRow(ctx, query,
		bid.AuctionID, bid.ChitSubscriberID, bid.SubscriberID,
		bid.BidDiscountAmount, bid.BidDiscountPercent,
	).Scan(&bid.BidID, &bid.BidTime, &bid.CreatedAt) // ✅ FIXED: BidID
}

func (r *AuctionRepository) GetBids(ctx context.Context, auctionID int) ([]*domain.AuctionBid, error) {
	query := `
		SELECT ab.bid_id, ab.auction_id, ab.chit_subscriber_id, ab.subscriber_id,
		       COALESCE(s.full_name, '') AS subscriber_name,
		       ab.bid_discount_amount, ab.bid_discount_percent, ab.bid_time,
		       ab.is_winner, ab.is_valid, ab.rejection_reason, ab.created_at
		FROM auction_bids ab
		LEFT JOIN subscribers s ON ab.subscriber_id = s.subscriber_id
		WHERE ab.auction_id = $1 AND ab.is_valid = true
		ORDER BY ab.bid_discount_amount DESC, ab.bid_time ASC
	`

	rows, err := r.db.Query(ctx, query, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bids, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.AuctionBid])
	if err != nil {
		return nil, err
	}

	return bids, nil
}

func (r *AuctionRepository) Finalize(ctx context.Context, auctionID, winningBidID, userID int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update auction with winning bid
	_, err = tx.Exec(ctx, `
		UPDATE auctions
		SET winning_bid_id = $1, status = 'COMPLETED', conducted_by = $2, updated_at = NOW()
		WHERE auction_id = $3
	`, winningBidID, userID, auctionID)
	if err != nil {
		return err
	}

	// Mark winning bid
	_, err = tx.Exec(ctx, `
		UPDATE auction_bids
		SET is_winner = true
		WHERE bid_id = $1
	`, winningBidID)
	if err != nil {
		return err
	}

	// Mark subscriber as prized
	_, err = tx.Exec(ctx, `
		UPDATE chit_subscribers
		SET is_prized = true, prized_month = (
			SELECT month_number FROM auctions WHERE auction_id = $1
		), prized_date = NOW()
		WHERE chit_subscriber_id = (
			SELECT chit_subscriber_id FROM auction_bids WHERE bid_id = $2
		)
	`, auctionID, winningBidID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Update updates scheduling details for an auction — auction_date, auction_time, venue
func (r *AuctionRepository) Update(ctx context.Context, id, companyID int, req *domain.UpdateAuctionRequest) error {
	// Build auction_date cast only if provided
	var auctionDateExpr = "auction_date"
	if req.AuctionDate != nil {
		auctionDateExpr = "$4::date"
	}
	query := fmt.Sprintf(`
		UPDATE auctions SET
			venue        = COALESCE($1, venue),
			auction_time = COALESCE($2::time, auction_time),
			auction_date = COALESCE(%s, auction_date),
			updated_at   = NOW()
		WHERE auction_id = $3
		  AND status IN ('SCHEDULED', 'IN_PROGRESS')
		  AND EXISTS (SELECT 1 FROM chit_groups cg WHERE cg.chit_id = auctions.chit_id AND cg.company_id = $5)
	`, auctionDateExpr)

	var auctionDateVal interface{}
	if req.AuctionDate != nil {
		auctionDateVal = *req.AuctionDate
	}

	result, err := r.db.Exec(ctx, query,
		req.Venue, req.AuctionTime, id, auctionDateVal, companyID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetStatus sets the auction status (SCHEDULED → IN_PROGRESS → COMPLETED)
func (r *AuctionRepository) SetStatus(ctx context.Context, id int, status string, userID int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE auctions SET status = $1, conducted_by = $2, updated_at = NOW() WHERE auction_id = $3`,
		status, userID, id,
	)
	return err
}

// Cancel sets auction status to CANCELLED — only allowed if SCHEDULED or IN_PROGRESS
func (r *AuctionRepository) Cancel(ctx context.Context, id, companyID, userID int, remarks string) error {
	var remarksPtr *string
	if remarks != "" {
		remarksPtr = &remarks
	}
	result, err := r.db.Exec(ctx, `
		UPDATE auctions SET
			status     = 'CANCELLED',
			remarks    = COALESCE($1, remarks),
			updated_at = NOW()
		WHERE auction_id = $2
		  AND status IN ('SCHEDULED', 'IN_PROGRESS')
		  AND EXISTS (
			SELECT 1 FROM chit_groups cg
			WHERE cg.chit_id = auctions.chit_id AND cg.company_id = $3
		  )
	`, remarksPtr, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.NewAppError("CANCEL_FAILED",
			"Auction not found or cannot be cancelled (only SCHEDULED or IN_PROGRESS auctions can be cancelled)", nil)
	}
	return nil
}

// List returns paginated auctions for a company
func (r *AuctionRepository) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Auction, int64, error) {
	base := `
		SELECT
			a.auction_id, a.chit_id,
			COALESCE(cg.chit_code, '')  AS chit_code,
			COALESCE(cg.chit_name, '')  AS chit_name,
			cg.chit_value,
			a.month_number, a.auction_date, a.auction_time,
			a.venue, a.max_discount_allowed, a.winning_bid_id, a.auction_method,
			a.status, a.conducted_by,
			COALESCE(u.full_name, '')    AS conducted_by_name,
			COALESCE(s.full_name, '')    AS winner_name,
			a.remarks, a.created_at, a.updated_at
		FROM auctions a
		INNER JOIN chit_groups cg  ON a.chit_id        = cg.chit_id
		LEFT  JOIN users u         ON a.conducted_by   = u.user_id
		LEFT  JOIN auction_bids ab ON ab.bid_id         = a.winning_bid_id
		LEFT  JOIN subscribers s   ON ab.subscriber_id  = s.subscriber_id`

	args := []interface{}{companyID}
	where := " WHERE cg.company_id = $1"
	argPos := 2

	if chitID, ok := filters["chit_id"].(int); ok && chitID > 0 {
		where += fmt.Sprintf(" AND a.chit_id = $%d", argPos)
		args = append(args, chitID)
		argPos++
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		where += fmt.Sprintf(" AND a.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if month, ok := filters["month_number"].(int); ok && month > 0 {
		where += fmt.Sprintf(" AND a.month_number = $%d", argPos)
		args = append(args, month)
		argPos++
	}

	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM auctions a INNER JOIN chit_groups cg ON a.chit_id = cg.chit_id`+where, args...).Scan(&total)

	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok {
		offset = o
	}
	query := base + where + fmt.Sprintf(" ORDER BY a.auction_date DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	auctions := []*domain.Auction{}
	for rows.Next() {
		var a domain.Auction
		if err := rows.Scan(
			&a.AuctionID, &a.ChitID, &a.ChitCode, &a.ChitName, &a.ChitValue,
			&a.MonthNumber, &a.AuctionDate, &a.AuctionTime,
			&a.Venue, &a.MaxDiscountAllowed, &a.WinningBidID, &a.AuctionMethod,
			&a.Status, &a.ConductedBy, &a.ConductedByName, &a.WinnerName,
			&a.Remarks, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		auctions = append(auctions, &a)
	}
	return auctions, total, rows.Err()
}
