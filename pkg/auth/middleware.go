package auth

import (
	"context"
	"os"
	"strings"

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/gin-gonic/gin"
)

func ApiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.AbortWithError(401, cerr.NewUnauthorizedError("Missing Authorization header", nil))
			return
		}

		parts := strings.Split(authorization, " ")
		if len(parts) != 2 {
			c.AbortWithError(401, cerr.NewUnauthorizedError("Invalid Authorization header", nil))
			return
		}

		if parts[0] != "Bearer" {
			c.AbortWithError(401, cerr.NewUnauthorizedError("Invalid Authorization header", nil))
			return
		}

		bearerToken := parts[1]

		// TODO: update this to use real auth service
		if bearerToken != "chariot$123" {
			c.AbortWithError(401, cerr.NewUnauthorizedError("Invalid Authorization", nil))
			return
		}

		// TODO: source user from token
		c.Set("user_id", uint64(0))

		if entity, ok := os.LookupEnv("BANK_ENTITY"); ok {
			// TODO: add entity key to context for downstream data filtering
			// do not set this for admin users
			c.Set("entity", entity) // sandbox: sandbox_entity_qyn2yuztryrriavmjyza
		}

		c.Next()
	}
}

func UserId(ctx context.Context) uint64 {
	userId := ctx.Value("user_id")
	if userId != nil {
		return userId.(uint64)
	}
	return 0
}
