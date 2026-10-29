package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"chit-fund-api/internal/config"
	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
	"chit-fund-api/pkg/cache"
	"chit-fund-api/pkg/jwt"
)

type MobileService struct {
	repo       *repository.MobileRepository
	jwtManager *jwt.JWTManager
	cache      *cache.RedisClient
}

func NewMobileService(
	repo *repository.MobileRepository,
	cfg config.JWTConfig,
	redisCache *cache.RedisClient,
) *MobileService {
	return &MobileService{
		repo:       repo,
		jwtManager: jwt.NewJWTManager(cfg.Secret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL),
		cache:      redisCache,
	}
}

// ── Subscriber App Auth ───────────────────────────────────────────────────────

// Register creates a new app_user record linked to an existing subscriber.
func (s *MobileService) Register(ctx context.Context, req *domain.AppRegisterRequest) error {
	// Verify subscriber exists
	profile, err := s.repo.GetSubscriberByMobile(ctx, req.Mobile)
	if err == domain.ErrNotFound {
		return domain.NewAppError("SUBSCRIBER_NOT_FOUND",
			"No subscriber found with this mobile number. Please contact your chit fund company.", nil)
	}
	if profile.SubscriberID != req.SubscriberID {
		return domain.NewAppError("SUBSCRIBER_MISMATCH",
			"Mobile number does not match the provided subscriber ID.", nil)
	}

	// Check not already registered
	existing, _ := s.repo.GetAppUserByMobile(ctx, req.Mobile)
	if existing != nil {
		return domain.NewAppError("ALREADY_REGISTERED",
			"This mobile number is already registered in the app.", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.NewAppError("HASH_ERROR", "Failed to process password", err)
	}

	u := &domain.AppUser{
		SubscriberID: req.SubscriberID,
		Mobile:       req.Mobile,
		PasswordHash: string(hash),
		DeviceType:   &req.DeviceType,
	}
	if req.DeviceToken != "" {
		u.DeviceToken = &req.DeviceToken
	}
	if req.AppVersion != "" {
		u.AppVersion = &req.AppVersion
	}
	return s.repo.CreateAppUser(ctx, u)
}

// Login authenticates the subscriber and returns a JWT pair.
func (s *MobileService) Login(ctx context.Context, req *domain.AppLoginRequest) (*domain.AppLoginResponse, error) {
	u, err := s.repo.GetAppUserByMobile(ctx, req.Mobile)
	if err == domain.ErrNotFound {
		return nil, domain.NewAppError("AUTH_FAILED", "Invalid mobile number or password", nil)
	}
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, domain.NewAppError("ACCOUNT_INACTIVE",
			"Your account is inactive. Please contact your chit fund company.", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.NewAppError("AUTH_FAILED", "Invalid mobile number or password", nil)
	}

	// Resolve companyID from subscriber
	var companyID int
	s.repo.DB().QueryRow(ctx,
		`SELECT company_id FROM subscribers WHERE subscriber_id = $1`, u.SubscriberID,
	).Scan(&companyID)

	claims := &domain.AppTokenClaims{
		AppUserID:    u.AppUserID,
		SubscriberID: u.SubscriberID,
		CompanyID:    companyID,
		AppType:      "subscriber",
	}

	// Reuse staff JWT manager but encode app-specific fields
	staffClaims := &domain.TokenClaims{
		UserID:    u.AppUserID,
		CompanyID: companyID,
		RoleID:    0,
		RoleName:  "APP_SUBSCRIBER",
	}
	_ = claims // staffClaims carries the same data via the existing JWT infra

	accessToken, err := s.jwtManager.GenerateAppToken(staffClaims, u.SubscriberID, false)
	if err != nil {
		return nil, domain.NewAppError("TOKEN_ERROR", "Failed to generate token", err)
	}
	refreshToken, err := s.jwtManager.GenerateAppToken(staffClaims, u.SubscriberID, true)
	if err != nil {
		return nil, domain.NewAppError("TOKEN_ERROR", "Failed to generate refresh token", err)
	}

	_ = s.repo.UpdateLastLogin(ctx, u.AppUserID)

	profile, _ := s.repo.GetSubscriberProfile(ctx, u.AppUserID, u.SubscriberID)
	if profile != nil {
		profile.IsVerified = u.IsVerified
	}

	return &domain.AppLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
		Subscriber:   profile,
	}, nil
}

// SendOTP generates a 6-digit OTP, stores it (hashed via bcrypt optional), and
// returns the code so the caller can send it via SMS provider.
func (s *MobileService) SendOTP(ctx context.Context, mobile string) (string, error) {
	u, err := s.repo.GetAppUserByMobile(ctx, mobile)
	if err == domain.ErrNotFound {
		// Allow OTP for registration too — just return success
		return "", domain.NewAppError("USER_NOT_FOUND",
			"No app account found for this mobile. Please register first.", nil)
	}
	if err != nil {
		return "", err
	}

	otp := generateOTP()
	expiry := time.Now().Add(10 * time.Minute)
	if err := s.repo.SetOTP(ctx, u.AppUserID, otp, expiry); err != nil {
		return "", domain.NewAppError("OTP_ERROR", "Failed to generate OTP", err)
	}

	// Cache OTP for extra lookup speed (10 min TTL)
	if s.cache != nil {
		_ = s.cache.Set(ctx, "otp:"+mobile, otp, 10*time.Minute)
	}
	return otp, nil
}

// VerifyOTP checks the OTP and marks user as verified.
func (s *MobileService) VerifyOTP(ctx context.Context, req *domain.AppVerifyOTPRequest) error {
	_, err := s.repo.VerifyOTP(ctx, req.Mobile, req.OTP)
	if err != nil {
		return err
	}
	u, _ := s.repo.GetAppUserByMobile(ctx, req.Mobile)
	if u != nil {
		_ = s.repo.MarkVerified(ctx, u.AppUserID)
	}
	return nil
}

func (s *MobileService) UpdateDevice(ctx context.Context, appUserID int, req *domain.AppUpdateDeviceRequest) error {
	return s.repo.UpdateDevice(ctx, appUserID, req.DeviceToken, req.DeviceType, req.AppVersion)
}

// ── Subscriber App data ───────────────────────────────────────────────────────

func (s *MobileService) GetDashboard(ctx context.Context, appUserID, subscriberID int) (*domain.AppDashboard, error) {
	profile, err := s.repo.GetSubscriberProfile(ctx, appUserID, subscriberID)
	if err != nil {
		return nil, err
	}
	dash, err := s.repo.GetSubscriberDashboard(ctx, subscriberID)
	if err != nil {
		return nil, err
	}
	dash.Profile = profile
	return dash, nil
}

func (s *MobileService) GetChits(ctx context.Context, subscriberID int) ([]*domain.AppChitSummary, error) {
	return s.repo.GetSubscriberChits(ctx, subscriberID)
}

func (s *MobileService) GetDues(ctx context.Context, subscriberID int) ([]*domain.AppInstallmentDue, error) {
	return s.repo.GetSubscriberDues(ctx, subscriberID)
}

func (s *MobileService) GetPaymentHistory(ctx context.Context, subscriberID, limit, offset int) ([]*domain.AppPaymentHistoryItem, int64, error) {
	return s.repo.GetSubscriberPaymentHistory(ctx, subscriberID, limit, offset)
}

func (s *MobileService) GetNotifications(ctx context.Context, subscriberID, limit, offset int) ([]*domain.AppNotification, int64, error) {
	return s.repo.GetNotifications(ctx, subscriberID, limit, offset)
}

func (s *MobileService) MarkNotificationRead(ctx context.Context, notificationID, subscriberID int) error {
	return s.repo.MarkNotificationRead(ctx, notificationID, subscriberID)
}

func (s *MobileService) MarkAllNotificationsRead(ctx context.Context, subscriberID int) error {
	return s.repo.MarkAllNotificationsRead(ctx, subscriberID)
}

// ── Agent App ─────────────────────────────────────────────────────────────────

// AgentLogin uses the existing staff JWT (same auth as admin panel) — the agent
// logs in with their staff username + password. We validate role is COLLECTION_AGENT.
// (Handler calls the existing AuthService.Login and checks role.)

func (s *MobileService) GetAgentDashboard(ctx context.Context, agentUserID, companyID int) (*domain.AgentCollectionSummary, error) {
	// Fetch agent name from users table
	var agentName string
	s.repo.DB().QueryRow(ctx,
		`SELECT full_name FROM users WHERE user_id = $1`, agentUserID,
	).Scan(&agentName)

	dash, err := s.repo.GetAgentDashboard(ctx, agentUserID, companyID)
	if err != nil {
		return nil, err
	}
	dash.AgentName = agentName
	return dash, nil
}

func (s *MobileService) GetSubscriberList(ctx context.Context, companyID int, filters map[string]interface{}) ([]*domain.AgentSubscriberListItem, int64, error) {
	return s.repo.GetAgentSubscriberList(ctx, companyID, filters)
}

func (s *MobileService) RecordCashPayment(ctx context.Context,
	req *domain.AgentRecordCashPaymentRequest,
	agentUserID, companyID, branchID int,
) (*domain.AgentRecentCollection, error) {
	return s.repo.RecordCashPayment(ctx, req, agentUserID, companyID, branchID)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func generateOTP() string {
	b := make([]byte, 3)
	rand.Read(b)
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", n)
}
