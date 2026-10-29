package middleware

import (
	"net/http"
	"strings"

	"chit-fund-api/internal/domain"
	"chit-fund-api/pkg/jwt"

	"github.com/gin-gonic/gin"
)

const (
	ClaimsKey = "claims"
)

func AuthRequired(secret string) gin.HandlerFunc {
	jwtManager := jwt.NewJWTManager(secret, 15, 24)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func GetClaims(c *gin.Context) *domain.TokenClaims {
	claims, exists := c.Get(ClaimsKey)
	if !exists {
		return nil
	}
	return claims.(*domain.TokenClaims)
}
