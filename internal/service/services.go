package service

type Services struct {
	Auth        *AuthService
	Company     *CompanyService
	Branch      *BranchService
	User        *UserService
	Subscriber  *SubscriberService
	ChitGroup   *ChitGroupService
	Auction     *AuctionService
	Installment *InstallmentService
	Payout      *PayoutService
	Default     *DefaultService
	LegalNotice *LegalNoticeService
	Dashboard   *DashboardService
	Report      *ReportService
	Mobile      *MobileService
	// Digital Payments
	Payment *PaymentService
	// Staff Notifications
	Notification *NotificationService
	// Financial Suite
	Financial *FinancialService
}
