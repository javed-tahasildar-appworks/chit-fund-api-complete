package middleware

import (
	"net/http"
	"strings"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/jwt"

	"github.com/gin-gonic/gin"
)

const AppClaimsKey = "app_claims"

// AppSubscriberClaims holds what is extracted from a subscriber app JWT.
type AppSubscriberClaims struct {
	AppUserID    int
	SubscriberID int
	CompanyID    int
}

// GetAppClaims extracts subscriber app claims from gin context.
func GetAppClaims(c *gin.Context) *AppSubscriberClaims {
	if v, exists := c.Get(AppClaimsKey); exists {
		if claims, ok := v.(*AppSubscriberClaims); ok {
			return claims
		}
	}
	return nil
}

// AppSubscriberAuth validates a JWT issued for the subscriber mobile app.
// It expects app_type="subscriber" and subscriber_id in the token.
func AppSubscriberAuth(secret string) gin.HandlerFunc {
	jwtManager := jwt.NewJWTManager(secret, 15, 24*7)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
				Error: "Authorization header required", Code: "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
				Error: "Invalid authorization format", Code: "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		rawClaims, err := jwtManager.ValidateAppTokenRaw(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
				Error: "Invalid or expired token", Code: "TOKEN_INVALID",
			})
			c.Abort()
			return
		}

		appType, _ := rawClaims["app_type"].(string)
		if appType != "subscriber" {
			c.JSON(http.StatusForbidden, domain.ErrorResponse{
				Error: "This endpoint is for subscriber app only", Code: "FORBIDDEN",
			})
			c.Abort()
			return
		}

		subscriberID := 0
		if sid, ok := rawClaims["subscriber_id"].(float64); ok {
			subscriberID = int(sid)
		}
		appUserID := 0
		if uid, ok := rawClaims["user_id"].(float64); ok {
			appUserID = int(uid)
		}
		companyID := 0
		if cid, ok := rawClaims["company_id"].(float64); ok {
			companyID = int(cid)
		}

		c.Set(AppClaimsKey, &AppSubscriberClaims{
			AppUserID:    appUserID,
			SubscriberID: subscriberID,
			CompanyID:    companyID,
		})
		c.Next()
	}
}

// RequireAgentRole ensures the authenticated staff user has COLLECTION_AGENT role.
func RequireAgentRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{Error: "Unauthorized"})
			c.Abort()
			return
		}
		if claims.RoleName != "COLLECTION_AGENT" &&
			claims.RoleName != "BRANCH_MANAGER" &&
			claims.RoleName != "ADMIN" &&
			claims.RoleName != "SUPER_ADMIN" {
			c.JSON(http.StatusForbidden, domain.ErrorResponse{
				Error: "This endpoint is for collection agents only", Code: "FORBIDDEN",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
