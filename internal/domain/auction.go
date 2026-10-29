package domain

import "time"

// Auction matches 'auctions' table + joined chit group info
type Auction struct {
	AuctionID          int       `json:"auction_id"`
	ChitID             int       `json:"chit_id"`
	ChitCode           string    `json:"chit_code"`  // joined from chit_groups
	ChitName           string    `json:"chit_name"`  // joined from chit_groups
	ChitValue          float64   `json:"chit_value"` // joined from chit_groups
	MonthNumber        int       `json:"month_number"`
	AuctionDate        time.Time `json:"auction_date"`
	AuctionTime        *string   `json:"auction_time"`
	Venue              *string   `json:"venue"`
	MaxDiscountAllowed float64   `json:"max_discount_allowed"`
	WinningBidID       *int      `json:"winning_bid_id"`
	AuctionMethod      *string   `json:"auction_method"`
	Status             string    `json:"status"`
	ConductedBy        *int      `json:"conducted_by"`
	ConductedByName    string    `json:"conducted_by_name"` // joined from users
	WinnerName         string    `json:"winner_name"`       // joined from subscribers via winning_bid
	Remarks            *string   `json:"remarks"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AuctionBid matches 'auction_bids' table exactly
type AuctionBid struct {
	BidID              int       `json:"bid_id"               db:"bid_id"`
	AuctionID          int       `json:"auction_id"           db:"auction_id"`
	ChitSubscriberID   int       `json:"chit_subscriber_id"   db:"chit_subscriber_id"`
	SubscriberID       int       `json:"subscriber_id"        db:"subscriber_id"`
	SubscriberName     string    `json:"subscriber_name"      db:"subscriber_name"`
	BidDiscountAmount  float64   `json:"bid_discount_amount"  db:"bid_discount_amount"`
	BidDiscountPercent float64   `json:"bid_discount_percent" db:"bid_discount_percent"`
	BidTime            time.Time `json:"bid_time"             db:"bid_time"`
	IsWinner           bool      `json:"is_winner"            db:"is_winner"`
	IsValid            bool      `json:"is_valid"             db:"is_valid"`
	RejectionReason    *string   `json:"rejection_reason"     db:"rejection_reason"`
	CreatedAt          time.Time `json:"created_at"           db:"created_at"`
}
