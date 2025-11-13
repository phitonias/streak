package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/phitonias/streak/internal/utils"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.Printf("Error: %v", err)

			// Return error response
			utils.ErrorResponse(c, 500, "Internal server error")
		}
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				utils.ErrorResponse(c, 500, "Internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
