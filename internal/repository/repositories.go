package repository

type Repositories struct {
	Company     *CompanyRepository
	Branch      *BranchRepository
	State       *StateRepository
	User        *UserRepository
	Role        *RoleRepository
	Subscriber  *SubscriberRepository
	KYC         *KYCRepository
	ChitGroup   *ChitGroupRepository
	Auction     *AuctionRepository
	Installment *InstallmentRepository
	Payout      *PayoutRepository
	Default     *DefaultRepository
	LegalNotice *LegalNoticeRepository
	Dashboard   *DashboardRepository
	AuditLog    *AuditLogRepository
	Mobile      *MobileRepository
	// Digital Payments
	Payment *PaymentRepository
	// Staff Notifications
	Notification *NotificationRepository
	// Financial Suite
	Financial *FinancialRepository
}
