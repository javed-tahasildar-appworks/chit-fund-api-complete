package middleware

import (
	"github.com/gin-gonic/gin"
)

// TenantContext ensures tenant isolation by setting company context
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.Next()
			return
		}

		// Set tenant context in request context for database queries
		c.Set("company_id", claims.CompanyID)

		if claims.BranchID != nil {
			c.Set("branch_id", *claims.BranchID)
		}

		c.Next()
	}
}
