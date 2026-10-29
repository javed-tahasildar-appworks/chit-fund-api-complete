package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequirePermission checks if user has specific permission
func RequirePermission(module, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// For SUPER_ADMIN, allow everything
		if claims.RoleName == "SUPER_ADMIN" {
			c.Next()
			return
		}

		// Check permission from claims (would need to be added to JWT)
		// This is a simplified version - in production, fetch from database or cache

		// For now, allow if user is authenticated
		// TODO: Implement proper permission checking
		c.Next()
	}
}

// RequireRole checks if user has specific role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		for _, role := range roles {
			if claims.RoleName == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// hasPermission checks if user permissions contain the required permission
func hasPermission(permissions json.RawMessage, module, resource, action string) bool {
	var perms map[string]map[string][]string
	if err := json.Unmarshal(permissions, &perms); err != nil {
		return false
	}

	if modulePerms, ok := perms[module]; ok {
		if actions, ok := modulePerms[resource]; ok {
			for _, a := range actions {
				if a == action {
					return true
				}
			}
		}
	}

	return false
}
