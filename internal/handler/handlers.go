package handler

// Handlers holds all HTTP handler instances for the entire application.
type Handlers struct {
	// Staff / Admin
	Auth        *AuthHandler
	Company     *CompanyHandler
	User        *UserHandler
	Subscriber  *SubscriberHandler
	ChitGroup   *ChitGroupHandler
	Auction     *AuctionHandler
	Installment *InstallmentHandler
	Payout      *PayoutHandler
	Default     *DefaultHandler
	LegalNotice *LegalNoticeHandler
	Dashboard   *DashboardHandler
	Report      *ReportHandler
	// Mobile Apps
	Mobile *MobileHandler
	// Digital Payments (Feature 1: Collections)
	Payment *PaymentHandler
	// Staff Notifications
	Notification *NotificationHandler
	// Financial Suite (P&L, Balance Sheet, Cash Flow, GST, Expenses)
	Financial *FinancialHandler
}
