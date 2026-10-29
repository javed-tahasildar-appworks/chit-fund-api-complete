package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
	"chit-fund-api/pkg/cache"
)

// ─── Company Service ──────────────────────────────────────────────────────────

type CompanyService struct {
	repo  *repository.CompanyRepository
	cache *cache.RedisClient
}

func NewCompanyService(repo *repository.CompanyRepository, cache *cache.RedisClient) *CompanyService {
	return &CompanyService{repo: repo, cache: cache}
}

func (s *CompanyService) GetProfile(ctx context.Context, companyID int) (*domain.Company, error) {
	return s.repo.GetProfile(ctx, companyID)
}

func (s *CompanyService) Update(ctx context.Context, companyID int, req *domain.UpdateCompanyRequest) error {
	return s.repo.Update(ctx, companyID, req)
}

func (s *CompanyService) Create(ctx context.Context, req *domain.CreateCompanyRequest) (*domain.Company, error) {
	company := &domain.Company{
		CompanyName:              req.CompanyName,
		RegistrationNumber:       req.RegistrationNumber,
		PANNumber:                &req.PANNumber,
		GSTIN:                    &req.GSTIN,
		StateID:                  &req.StateID,
		RegisteredAddress:        req.RegisteredAddress,
		ContactEmail:             &req.ContactEmail,
		ContactPhone:             &req.ContactPhone,
		ForemanCommissionPercent: req.ForemanCommissionPercent,
		Status:                   "ACTIVE",
	}
	if err := s.repo.Create(ctx, company); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to create company", err)
	}
	return company, nil
}

func (s *CompanyService) GetByID(ctx context.Context, id int) (*domain.Company, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CompanyService) List(ctx context.Context, filters map[string]interface{}) ([]*domain.Company, int64, error) {
	return s.repo.List(ctx, filters)
}

// ─── Branch Service ───────────────────────────────────────────────────────────

type BranchService struct {
	branchRepo *repository.BranchRepository
	stateRepo  *repository.StateRepository
	roleRepo   *repository.RoleRepository
	cache      *cache.RedisClient
}

func NewBranchService(branchRepo *repository.BranchRepository, stateRepo *repository.StateRepository, roleRepo *repository.RoleRepository, cache *cache.RedisClient) *BranchService {
	return &BranchService{branchRepo: branchRepo, stateRepo: stateRepo, roleRepo: roleRepo, cache: cache}
}

func (s *BranchService) CreateBranch(ctx context.Context, companyID int, req *domain.CreateBranchRequest) (*domain.Branch, error) {
	// Only use fields that exist in the actual branches table
	branch := &domain.Branch{
		CompanyID:  companyID,
		BranchCode: req.BranchCode,
		BranchName: req.BranchName,
		IsActive:   true,
	}
	if req.Address != nil {
		branch.Address = *req.Address
	}
	if req.City != nil {
		branch.City = *req.City
	}
	if req.Pincode != nil {
		branch.Pincode = *req.Pincode
	}
	if req.Phone != nil {
		branch.ContactPhone = *req.Phone
	}
	if err := s.branchRepo.Create(ctx, branch); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to create branch", err)
	}
	return branch, nil
}

func (s *BranchService) ListBranches(ctx context.Context, companyID int) ([]*domain.Branch, error) {
	return s.branchRepo.List(ctx, companyID)
}

func (s *BranchService) GetBranch(ctx context.Context, id, companyID int) (*domain.Branch, error) {
	return s.branchRepo.GetByID(ctx, id, companyID)
}

func (s *BranchService) UpdateBranch(ctx context.Context, id, companyID int, req *domain.UpdateBranchRequest) error {
	return s.branchRepo.Update(ctx, id, companyID, req)
}

func (s *BranchService) ListStates(ctx context.Context) ([]*domain.State, error) {
	return s.stateRepo.List(ctx)
}

func (s *BranchService) GetState(ctx context.Context, id int) (*domain.State, error) {
	return s.stateRepo.GetByID(ctx, id)
}

func (s *BranchService) ListRoles(ctx context.Context, companyID int) ([]*domain.Role, error) {
	return s.roleRepo.List(ctx, companyID)
}

func (s *BranchService) GetRole(ctx context.Context, id, companyID int) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id, companyID)
}

func (s *BranchService) UpdateRolePermissions(ctx context.Context, id, companyID int, req *domain.UpdateRolePermissionsRequest) error {
	return s.roleRepo.UpdatePermissions(ctx, id, companyID, req.Permissions)
}

// ─── User Service ─────────────────────────────────────────────────────────────

type UserService struct {
	repo  *repository.UserRepository
	cache *cache.RedisClient
}

func NewUserService(repo *repository.UserRepository, cache *cache.RedisClient) *UserService {
	return &UserService{repo: repo, cache: cache}
}

func (s *UserService) Create(ctx context.Context, companyID int, req *domain.CreateUserRequest) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.NewAppError("HASH_ERROR", "Failed to process password", err)
	}
	user := &domain.User{
		CompanyID:    companyID,
		BranchID:     req.BranchID,
		RoleID:       req.RoleID,
		Username:     req.Username,
		Email:        req.Email,
		Mobile:       req.Mobile,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		EmployeeCode: req.EmployeeCode,
		Status:       "ACTIVE",
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to create user", err)
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id, companyID int) (*domain.User, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *UserService) Update(ctx context.Context, id, companyID int, req *domain.UpdateUserRequest) error {
	return s.repo.Update(ctx, id, companyID, req)
}

func (s *UserService) Deactivate(ctx context.Context, id, companyID int) error {
	return s.repo.SetStatus(ctx, id, companyID, "INACTIVE")
}

func (s *UserService) Activate(ctx context.Context, id, companyID int) error {
	return s.repo.SetStatus(ctx, id, companyID, "ACTIVE")
}

func (s *UserService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.User, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}

// ─── Subscriber Service ───────────────────────────────────────────────────────

type SubscriberService struct {
	repo    *repository.SubscriberRepository
	kycRepo *repository.KYCRepository
	cache   *cache.RedisClient
}

func NewSubscriberService(repo *repository.SubscriberRepository, kycRepo *repository.KYCRepository, cache *cache.RedisClient) *SubscriberService {
	return &SubscriberService{repo: repo, kycRepo: kycRepo, cache: cache}
}

func (s *SubscriberService) Create(ctx context.Context, companyID, createdBy int, req *domain.CreateSubscriberRequest) (*domain.SubscriberListItem, error) {
	// Auto-generate subscriber code: SUB + companyID + timestamp
	code := fmt.Sprintf("SUB%d%d", companyID, time.Now().UnixMilli())
	if len(code) > 50 {
		code = code[:50]
	}

	// Validate gender against DB CHECK constraint
	gender := strings.ToUpper(req.Gender)
	if gender != "" && gender != "MALE" && gender != "FEMALE" && gender != "OTHER" {
		return nil, domain.NewAppError("INVALID_GENDER", "gender must be MALE, FEMALE, or OTHER", nil)
	}

	// Validate kyc_status against DB CHECK constraint
	kycStatus := strings.ToUpper(req.KYCStatus)
	if kycStatus == "" {
		kycStatus = "PENDING"
	}
	if kycStatus != "PENDING" && kycStatus != "VERIFIED" && kycStatus != "REJECTED" {
		return nil, domain.NewAppError("INVALID_KYC_STATUS", "kyc_status must be PENDING, VERIFIED, or REJECTED", nil)
	}

	// Validate credit_score against DB CHECK constraint (300–900)
	if req.CreditScore != nil && (*req.CreditScore < 300 || *req.CreditScore > 900) {
		return nil, domain.NewAppError("INVALID_CREDIT_SCORE", "credit_score must be between 300 and 900", nil)
	}

	createdByPtr := &createdBy
	sub := &domain.Subscriber{
		CompanyID:      companyID,
		BranchID:       req.BranchID,
		SubscriberCode: code,
		FullName:       req.FullName,
		Address:        req.Address,
		Mobile:         req.Mobile,
		KYCStatus:      kycStatus,
		IsBlacklisted:  false,
		CreatedBy:      createdByPtr,
	}

	// Optional string fields — only set if non-empty
	if req.FatherName != "" {
		sub.FatherName = &req.FatherName
	}
	if gender != "" {
		sub.Gender = &gender
	}
	if req.AadhaarNumber != "" {
		sub.AadhaarNumber = &req.AadhaarNumber
	}
	if req.PANNumber != "" {
		sub.PANNumber = &req.PANNumber
	}
	if req.City != "" {
		sub.City = &req.City
	}
	if req.Pincode != "" {
		sub.Pincode = &req.Pincode
	}
	if req.AlternateMobile != "" {
		sub.AlternateMobile = &req.AlternateMobile
	}
	if req.Email != "" {
		sub.Email = &req.Email
	}
	if req.Occupation != "" {
		sub.Occupation = &req.Occupation
	}
	if req.PhotoPath != "" {
		sub.PhotoPath = &req.PhotoPath
	}
	if req.SignaturePath != "" {
		sub.SignaturePath = &req.SignaturePath
	}
	if req.AnnualIncome > 0 {
		sub.AnnualIncome = &req.AnnualIncome
	}
	if req.StateID != nil {
		sub.StateID = req.StateID
	}
	if req.CreditScore != nil {
		sub.CreditScore = req.CreditScore
	}

	// Parse DOB
	if req.DOB != "" {
		dob, err := time.Parse("2006-01-02", req.DOB)
		if err != nil {
			return nil, domain.NewAppError("INVALID_DOB", "dob must be in YYYY-MM-DD format", err)
		}
		sub.DOB = &dob
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to create subscriber", err)
	}

	// Return the full shaped response (same as GetByID) — null-free
	return s.repo.GetByID(ctx, sub.SubscriberID, companyID)
}

func (s *SubscriberService) GetByID(ctx context.Context, id, companyID int) (*domain.SubscriberListItem, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *SubscriberService) Update(ctx context.Context, id, companyID int, req *domain.UpdateSubscriberRequest) error {
	return s.repo.Update(ctx, id, companyID, req)
}

func (s *SubscriberService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.SubscriberListItem, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}

func (s *SubscriberService) Blacklist(ctx context.Context, id, companyID int, req *domain.BlacklistRequest) error {
	return s.repo.SetBlacklist(ctx, id, companyID, true, &req.Reason)
}

func (s *SubscriberService) Unblacklist(ctx context.Context, id, companyID int) error {
	return s.repo.SetBlacklist(ctx, id, companyID, false, nil)
}

func (s *SubscriberService) ListDocuments(ctx context.Context, subscriberID, companyID int) ([]*domain.SubscriberDocument, error) {
	return s.kycRepo.ListDocuments(ctx, subscriberID, companyID)
}

func (s *SubscriberService) UploadDocument(ctx context.Context, subscriberID, companyID int, req *domain.UploadDocumentRequest) (*domain.SubscriberDocument, error) {
	// Verify subscriber belongs to this company
	if _, err := s.repo.GetByID(ctx, subscriberID, companyID); err != nil {
		return nil, err
	}
	doc := &domain.SubscriberDocument{
		SubscriberID:       subscriberID,
		DocumentType:       req.DocumentType,
		FilePath:           req.DocumentPath,
		DocumentNumber:     req.DocumentNumber,
		ExpiryDate:         req.ExpiryDate,
		VerificationStatus: "PENDING",
		VerifiedByText:     "",
		UploadedDate:       "",
	}
	if err := s.kycRepo.UploadDocument(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *SubscriberService) VerifyDocument(ctx context.Context, docID, subscriberID, verifiedBy, companyID int, req *domain.VerifyDocumentRequest) error {
	// Ensure subscriber belongs to company first
	if _, err := s.repo.GetByID(ctx, subscriberID, companyID); err != nil {
		return err
	}
	return s.kycRepo.VerifyDocument(ctx, docID, subscriberID, verifiedBy, req)
}

func (s *SubscriberService) ListBankAccounts(ctx context.Context, subscriberID, companyID int) ([]*domain.SubscriberBank, error) {
	return s.kycRepo.ListBankAccounts(ctx, subscriberID, companyID)
}

func (s *SubscriberService) AddBankAccount(ctx context.Context, subscriberID, companyID int, req *domain.AddBankAccountRequest) (*domain.SubscriberBank, error) {
	if _, err := s.repo.GetByID(ctx, subscriberID, companyID); err != nil {
		return nil, err
	}
	bank := &domain.SubscriberBank{
		SubscriberID:      subscriberID,
		BankName:          req.BankName,
		AccountNumber:     req.AccountNumber,
		IFSCCode:          req.IFSCCode,
		AccountHolderName: req.AccountHolderName,
		AccountType:       req.AccountType,
		IsPrimary:         req.IsPrimary,
	}
	if err := s.kycRepo.AddBankAccount(ctx, bank); err != nil {
		return nil, domain.NewAppError("CREATE_FAILED", "Failed to add bank account", err)
	}
	return bank, nil
}

func (s *SubscriberService) DeleteBankAccount(ctx context.Context, bankID, subscriberID, companyID int) error {
	return s.kycRepo.DeleteBankAccount(ctx, bankID, subscriberID, companyID)
}

// ─── Installment Service ──────────────────────────────────────────────────────

type InstallmentService struct {
	repo  *repository.InstallmentRepository
	cache *cache.RedisClient
}

func NewInstallmentService(repo *repository.InstallmentRepository, cache *cache.RedisClient) *InstallmentService {
	return &InstallmentService{repo: repo, cache: cache}
}

func (s *InstallmentService) GenerateBatch(ctx context.Context, req *domain.GenerateInstallmentRequest, companyID int) (*domain.Installment, error) {
	return s.repo.GenerateBatch(ctx, req, companyID)
}

func (s *InstallmentService) GetByID(ctx context.Context, id, companyID int) (*domain.Installment, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *InstallmentService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Installment, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}

func (s *InstallmentService) ListSubscriberInstallments(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.SubscriberInstallment, int64, error) {
	return s.repo.ListSubscriberInstallments(ctx, companyID, filters)
}

func (s *InstallmentService) GetSubscriberInstallmentByID(ctx context.Context, id, companyID int) (*domain.SubscriberInstallment, error) {
	return s.repo.GetSubscriberInstallmentByID(ctx, id, companyID)
}

func (s *InstallmentService) RecordPayment(ctx context.Context, siID, collectedBy int, req *domain.RecordPaymentRequest) (*domain.PaymentTransaction, error) {
	return s.repo.RecordPayment(ctx, siID, collectedBy, req)
}

func (s *InstallmentService) ListPaymentTransactions(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.PaymentTransaction, int64, error) {
	return s.repo.ListPaymentTransactions(ctx, companyID, filters)
}

func (s *InstallmentService) GetPaymentTransactionByID(ctx context.Context, id, companyID int) (*domain.PaymentTransaction, error) {
	return s.repo.GetPaymentTransactionByID(ctx, id, companyID)
}

// ─── Payout Service ───────────────────────────────────────────────────────────

type PayoutService struct {
	repo     *repository.PayoutRepository
	chitRepo *repository.ChitGroupRepository
	cache    *cache.RedisClient
}

func NewPayoutService(repo *repository.PayoutRepository, chitRepo *repository.ChitGroupRepository, cache *cache.RedisClient) *PayoutService {
	return &PayoutService{repo: repo, chitRepo: chitRepo, cache: cache}
}

func (s *PayoutService) Initiate(ctx context.Context, companyID int, req *domain.InitiatePayoutRequest) (*domain.Payout, error) {
	// All financial calculations (gross_chit_value, discount_amount,
	// foreman_commission, net_payout) are derived inside the repository
	// directly from the auction's winning bid and chit group config.
	// The caller only supplies auction_id, bank_id, and remarks.
	remarks := req.Remarks
	payout := &domain.Payout{
		AuctionID: req.AuctionID,
		BankID:    req.BankID,
		Remarks:   &remarks,
		Status:    "PENDING",
	}
	if err := s.repo.Initiate(ctx, payout); err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.NewAppError("AUCTION_NOT_FOUND",
				"Auction not found, not completed, or has no winning bid", err)
		}
		return nil, domain.NewAppError("INITIATE_FAILED", "Failed to initiate payout", err)
	}
	return payout, nil
}

func (s *PayoutService) GetByID(ctx context.Context, id, companyID int) (*domain.Payout, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *PayoutService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Payout, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}

func (s *PayoutService) Approve(ctx context.Context, id, companyID, approvedBy int, req *domain.ApprovePayoutRequest) error {
	return s.repo.Approve(ctx, id, companyID, approvedBy, req.Remarks)
}

func (s *PayoutService) Disburse(ctx context.Context, id, companyID, paidBy int, req *domain.DisbursePayoutRequest) error {
	return s.repo.Disburse(ctx, id, companyID, paidBy, req)
}

func (s *PayoutService) CalculateDividend(ctx context.Context, auctionID, companyID int) (*domain.Dividend, error) {
	return s.repo.CalculateDividend(ctx, auctionID, companyID)
}

func (s *PayoutService) PostDividend(ctx context.Context, dividendID, companyID int) error {
	return s.repo.PostDividend(ctx, dividendID, companyID)
}

func (s *PayoutService) ListDividends(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Dividend, int64, error) {
	return s.repo.ListDividends(ctx, companyID, filters)
}

// ─── Default Service ──────────────────────────────────────────────────────────

type DefaultService struct {
	repo  *repository.DefaultRepository
	cache *cache.RedisClient
}

func NewDefaultService(repo *repository.DefaultRepository, cache *cache.RedisClient) *DefaultService {
	return &DefaultService{repo: repo, cache: cache}
}

func (s *DefaultService) Detect(ctx context.Context, companyID int) (int, error) {
	return s.repo.Detect(ctx, companyID)
}

func (s *DefaultService) Flag(ctx context.Context, companyID, branchID, subscriberID, chitID, chitSubscriberID int, reason string) (*domain.Default, error) {
	// Look up the subscriber_installment_id and due_date for the latest unpaid installment
	var siID int
	var dueDate time.Time
	var outstandingPrincipal, penaltyAmt, totalOutstanding float64

	err := s.repo.DB().QueryRow(context.Background(), `
		SELECT si.subscriber_installment_id, i.due_date,
			si.amount_due, si.penalty_amount, si.total_payable
		FROM subscriber_installments si
		INNER JOIN installments i ON si.installment_id = i.installment_id
		WHERE si.subscriber_id = $1
		  AND si.chit_subscriber_id = $2
		  AND si.status IN ('PENDING', 'PARTIAL')
		ORDER BY i.due_date ASC
		LIMIT 1
	`, subscriberID, chitSubscriberID).Scan(&siID, &dueDate, &outstandingPrincipal, &penaltyAmt, &totalOutstanding)
	if err != nil {
		// Fallback: use zero values and let caller supply data if lookup fails
		siID = 0
		dueDate = time.Now()
	}

	remarks := reason
	def := &domain.Default{
		SubscriberInstallmentID: siID,
		ChitSubscriberID:        chitSubscriberID,
		SubscriberID:            subscriberID,
		ChitID:                  chitID,
		InstallmentDueDate:      dueDate,
		OutstandingPrincipal:    outstandingPrincipal,
		PenaltyAmount:           penaltyAmt,
		TotalOutstanding:        totalOutstanding,
		Remarks:                 &remarks,
	}
	if err := s.repo.FlagManual(ctx, def); err != nil {
		return nil, domain.NewAppError("FLAG_FAILED", "Failed to flag default", err)
	}
	return def, nil
}

func (s *DefaultService) GetByID(ctx context.Context, id, companyID int) (*domain.Default, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *DefaultService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.Default, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}

func (s *DefaultService) Resolve(ctx context.Context, id, companyID, resolvedBy int, req *domain.ResolveDefaultRequest) error {
	return s.repo.Resolve(ctx, id, companyID, resolvedBy, req.ResolutionNotes)
}

func (s *DefaultService) SendReminder(ctx context.Context, id, companyID int, req *domain.SendReminderRequest) error {
	// In production: send SMS/email via notification service
	return s.repo.SendReminder(ctx, id, companyID)
}

// ─── Legal Notice Service ─────────────────────────────────────────────────────

type LegalNoticeService struct {
	repo  *repository.LegalNoticeRepository
	cache *cache.RedisClient
}

func NewLegalNoticeService(repo *repository.LegalNoticeRepository, cache *cache.RedisClient) *LegalNoticeService {
	return &LegalNoticeService{repo: repo, cache: cache}
}

func (s *LegalNoticeService) Issue(ctx context.Context, companyID, issuedBy int, req *domain.IssueLegalNoticeRequest) (*domain.LegalNotice, error) {
	noticeSentVia := req.NoticeSentVia
	noticeContent := req.NoticeContent
	noticeType := req.NoticeType
	defaultID := req.DefaultID

	issuedByPtr := issuedBy
	notice := &domain.LegalNotice{
		DefaultID:       &defaultID,
		NoticeType:      &noticeType,
		DeliveryMode:    &noticeSentVia,
		NoticeSentVia:   &noticeSentVia,
		ResponseDetails: &noticeContent,
		NoticeContent:   &noticeContent,
		IssuedBy:        &issuedByPtr,
		Status:          "SENT",
	}
	if req.ResponseDeadline != nil {
		t, err := time.Parse("2006-01-02", *req.ResponseDeadline)
		if err == nil {
			notice.ResponseDeadline = &t
		}
	}
	if err := s.repo.Issue(ctx, notice); err != nil {
		return nil, domain.NewAppError("ISSUE_FAILED", "Failed to issue legal notice", err)
	}
	return notice, nil
}

func (s *LegalNoticeService) GetByID(ctx context.Context, id, companyID int) (*domain.LegalNotice, error) {
	return s.repo.GetByID(ctx, id, companyID)
}

func (s *LegalNoticeService) List(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.LegalNotice, int64, error) {
	return s.repo.List(ctx, companyID, filters)
}

func (s *LegalNoticeService) Escalate(ctx context.Context, id, companyID int, req *domain.EscalateLegalNoticeRequest) error {
	return s.repo.Escalate(ctx, id, companyID, req)
}

// ─── Dashboard Service ────────────────────────────────────────────────────────

type DashboardService struct {
	repo *repository.DashboardRepository
}

func NewDashboardService(repo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) GetBundle(ctx context.Context, companyID int) (*domain.DashboardBundle, error) {
	kpis, err := s.repo.GetKPIs(ctx, companyID)
	if err != nil {
		return nil, err
	}
	collections, err := s.repo.GetMonthlyCollections(ctx, companyID)
	if err != nil {
		return nil, err
	}
	subscriberStats, err := s.repo.GetSubscriberStats(ctx, companyID)
	if err != nil {
		return nil, err
	}
	upcomingAuctions, err := s.repo.GetUpcomingAuctions(ctx, companyID)
	if err != nil {
		return nil, err
	}
	branchCollections, err := s.repo.GetBranchCollections(ctx, companyID)
	if err != nil {
		return nil, err
	}
	recentActivity, err := s.repo.GetRecentActivity(ctx, companyID)
	if err != nil {
		return nil, err
	}
	// Ensure slices are never nil — always return [] not null in JSON
	if collections == nil {
		collections = []domain.MonthlyCollectionItem{}
	}
	if upcomingAuctions == nil {
		upcomingAuctions = []domain.UpcomingAuction{}
	}
	if branchCollections == nil {
		branchCollections = []domain.BranchCollection{}
	}
	if recentActivity == nil {
		recentActivity = []domain.RecentActivity{}
	}

	return &domain.DashboardBundle{
		KPIs:               kpis,
		MonthlyCollections: collections,
		SubscriberStats:    subscriberStats,
		UpcomingAuctions:   upcomingAuctions,
		BranchCollections:  branchCollections,
		RecentActivity:     recentActivity,
		LastRefreshed:      time.Now(),
	}, nil
}

func (s *DashboardService) GetKPIs(ctx context.Context, companyID int) (*domain.DashboardKPIs, error) {
	return s.repo.GetKPIs(ctx, companyID)
}

func (s *DashboardService) GetMonthlyCollections(ctx context.Context, companyID int) ([]domain.MonthlyCollectionItem, error) {
	return s.repo.GetMonthlyCollections(ctx, companyID)
}

func (s *DashboardService) GetSubscriberStats(ctx context.Context, companyID int) (*domain.SubscriberStats, error) {
	return s.repo.GetSubscriberStats(ctx, companyID)
}

func (s *DashboardService) GetUpcomingAuctions(ctx context.Context, companyID int) ([]domain.UpcomingAuction, error) {
	return s.repo.GetUpcomingAuctions(ctx, companyID)
}

func (s *DashboardService) GetBranchCollections(ctx context.Context, companyID int) ([]domain.BranchCollection, error) {
	return s.repo.GetBranchCollections(ctx, companyID)
}

func (s *DashboardService) GetRecentActivity(ctx context.Context, companyID int) ([]domain.RecentActivity, error) {
	return s.repo.GetRecentActivity(ctx, companyID)
}

// ─── Report Service ───────────────────────────────────────────────────────────

type ReportService struct {
	repo      *repository.DashboardRepository
	auditRepo *repository.AuditLogRepository
}

func NewReportService(repo *repository.DashboardRepository, auditRepo *repository.AuditLogRepository) *ReportService {
	return &ReportService{repo: repo, auditRepo: auditRepo}
}

func (s *ReportService) GetDefaultersReport(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.DefaulterReport, error) {
	return s.repo.GetDefaultersReport(ctx, companyID, filters)
}

func (s *ReportService) GetChitPerformance(ctx context.Context, companyID int) ([]*domain.ChitPerformanceReport, error) {
	return s.repo.GetChitPerformance(ctx, companyID)
}

func (s *ReportService) GetSubscriberLedger(ctx context.Context, subscriberID, companyID int) ([]*domain.SubscriberLedgerEntry, error) {
	return s.repo.GetSubscriberLedger(ctx, subscriberID, companyID)
}

func (s *ReportService) GetMonthlyCollections(ctx context.Context, companyID int) ([]domain.MonthlyCollectionItem, error) {
	return s.repo.GetMonthlyCollectionSummary(ctx, companyID)
}

func (s *ReportService) GetAuditLogs(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.AuditLog, int64, error) {
	return s.auditRepo.List(ctx, companyID, filters)
}

func (s *ReportService) GetSystemLogs(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.SystemLog, int64, error) {
	return s.auditRepo.ListSystemLogs(ctx, companyID, filters)
}
