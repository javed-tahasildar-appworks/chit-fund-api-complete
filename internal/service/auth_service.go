package service

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"chit-fund-api/internal/config"
	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
	"chit-fund-api/pkg/cache"
	"chit-fund-api/pkg/jwt"
)

type AuthService struct {
	repo       *repository.UserRepository
	jwtManager *jwt.JWTManager
	cache      *cache.RedisClient
}

func NewAuthService(repo *repository.UserRepository, cfg config.JWTConfig, redisCache *cache.RedisClient) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtManager: jwt.NewJWTManager(cfg.Secret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL),
		cache:      redisCache,
	}
}

func (s *AuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, domain.NewAppError("AUTH_FAILED", "Invalid credentials", err)
	}

	if user.IsLocked {
		return nil, domain.NewAppError("ACCOUNT_LOCKED", "Account is locked due to too many failed attempts", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		fmt.Println("Password mismatch:", err)
		return nil, domain.NewAppError("AUTH_FAILED", "Invalid credentials", err)
	}

	claims := &domain.TokenClaims{
		UserID:    user.UserID,
		CompanyID: user.CompanyID,
		BranchID:  user.BranchID,
		RoleID:    user.RoleID,
		RoleName:  user.RoleName,
	}

	accessToken, err := s.jwtManager.GenerateToken(claims, false)
	if err != nil {
		return nil, domain.NewAppError("TOKEN_ERROR", "Failed to generate access token", err)
	}
	refreshToken, err := s.jwtManager.GenerateToken(claims, true)
	if err != nil {
		return nil, domain.NewAppError("TOKEN_ERROR", "Failed to generate refresh token", err)
	}

	user.PasswordHash = ""
	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    900,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResponse, error) {
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, domain.NewAppError("INVALID_TOKEN", "Invalid or expired refresh token", err)
	}
	accessToken, err := s.jwtManager.GenerateToken(claims, false)
	if err != nil {
		return nil, domain.NewAppError("TOKEN_ERROR", "Failed to generate token", err)
	}
	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if s.cache != nil {
		_ = s.cache.Set(ctx, "blacklisted:"+token, "1", 0)
	}
	return nil
}

func (s *AuthService) GetMe(ctx context.Context, userID, companyID int) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, req *domain.ChangePasswordRequest) error {
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return domain.NewAppError("HASH_ERROR", "Failed to process password", err)
	}
	return s.repo.UpdatePassword(ctx, userID, string(newHash))
}
