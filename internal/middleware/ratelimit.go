package middleware

import (
	"fmt"
	"net/http"
	"time"

	"chit-fund-api/internal/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var limiters = make(map[string]*rate.Limiter)

func RateLimit(cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.Next()
			return
		}

		// Create limiter key based on company_id
		key := fmt.Sprintf("company:%d", claims.CompanyID)

		limiter, exists := limiters[key]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(cfg.RequestsPerMinute)), cfg.RequestsPerMinute)
			limiters[key] = limiter
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": "1 minute",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
