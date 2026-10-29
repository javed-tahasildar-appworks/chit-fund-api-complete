package jwt

import (
	"fmt"
	"time"

	"chit-fund-api/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTManager(secret string, accessTTL, refreshTTL int) *JWTManager {
	return &JWTManager{
		secret:          secret,
		accessTokenTTL:  time.Duration(accessTTL) * time.Minute,
		refreshTokenTTL: time.Duration(refreshTTL) * time.Hour,
	}
}

// GenerateToken generates a JWT for staff users (admin panel).
func (m *JWTManager) GenerateToken(claims *domain.TokenClaims, isRefresh bool) (string, error) {
	exp := m.accessTokenTTL
	if isRefresh {
		exp = m.refreshTokenTTL
	}

	jwtClaims := jwt.MapClaims{
		"user_id":    claims.UserID,
		"company_id": claims.CompanyID,
		"role_id":    claims.RoleID,
		"role_name":  claims.RoleName,
		"exp":        time.Now().Add(exp).Unix(),
		"iat":        time.Now().Unix(),
	}
	if claims.BranchID != nil {
		jwtClaims["branch_id"] = *claims.BranchID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(m.secret))
}

// GenerateAppToken generates a JWT for the subscriber mobile app.
// Embeds subscriber_id and app_type="subscriber" in addition to standard claims.
func (m *JWTManager) GenerateAppToken(claims *domain.TokenClaims, subscriberID int, isRefresh bool) (string, error) {
	exp := m.accessTokenTTL
	if isRefresh {
		exp = m.refreshTokenTTL
	}

	jwtClaims := jwt.MapClaims{
		"user_id":       claims.UserID,
		"company_id":    claims.CompanyID,
		"role_id":       claims.RoleID,
		"role_name":     claims.RoleName,
		"subscriber_id": subscriberID,
		"app_type":      "subscriber",
		"exp":           time.Now().Add(exp).Unix(),
		"iat":           time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(m.secret))
}

// ValidateToken validates a staff JWT and returns typed TokenClaims.
func (m *JWTManager) ValidateToken(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	tokenClaims := &domain.TokenClaims{
		UserID:    int(claims["user_id"].(float64)),
		CompanyID: int(claims["company_id"].(float64)),
		RoleID:    int(claims["role_id"].(float64)),
		RoleName:  claims["role_name"].(string),
	}
	if branchID, ok := claims["branch_id"].(float64); ok {
		bid := int(branchID)
		tokenClaims.BranchID = &bid
	}
	return tokenClaims, nil
}

// ValidateAppTokenRaw validates a subscriber app JWT and returns raw MapClaims
// so middleware can extract app_type and subscriber_id without a fixed struct.
func (m *JWTManager) ValidateAppTokenRaw(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
