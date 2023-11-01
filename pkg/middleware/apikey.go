package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func ApiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Missing Authorization header"})
			return
		}

		parts := strings.Split(authorization, " ")
		if len(parts) != 2 {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid Authorization header"})
			return
		}

		if parts[0] != "Bearer" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid Authorization header"})
			return
		}

		bearerToken := parts[1]

		// TODO: update this to use real auth service
		if bearerToken != "chariot$123" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid Authorization"})
			return
		}

		// TODO: add entity key to context for downstream data filtering
		// do not set this for admin users

		c.Next()
	}
}
