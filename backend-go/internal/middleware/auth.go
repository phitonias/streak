package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phitonias/streak/internal/config"
	"github.com/phitonias/streak/internal/models"
	"github.com/phitonias/streak/internal/utils"
)

const UserContextKey = "user"

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, 401, "No token provided")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, 401, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := utils.ValidateToken(token, cfg.JWT.Secret)
		if err != nil {
			utils.ErrorResponse(c, 401, "Invalid or expired token")
			c.Abort()
			return
		}

		// Set user in context
		c.Set(UserContextKey, claims)
		c.Next()
	}
}

func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		claims, err := utils.ValidateToken(token, cfg.JWT.Secret)
		if err == nil {
			c.Set(UserContextKey, claims)
		}

		c.Next()
	}
}

func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userClaims, exists := c.Get(UserContextKey)
		if !exists {
			utils.ErrorResponse(c, 401, "Authentication required")
			c.Abort()
			return
		}

		claims := userClaims.(*utils.Claims)

		// Check if user has required role
		hasRole := false
		for _, role := range roles {
			if claims.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			utils.ErrorResponse(c, 403, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper function to get user claims from context
func GetUserClaims(c *gin.Context) (*utils.Claims, bool) {
	userClaims, exists := c.Get(UserContextKey)
	if !exists {
		return nil, false
	}
	claims, ok := userClaims.(*utils.Claims)
	return claims, ok
}
