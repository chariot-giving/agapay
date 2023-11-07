package auth

import (
	"os"
	"strings"

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/gin-gonic/gin"
)

func ApiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.Error(cerr.NewUnauthorizedError("Missing Authorization header", nil))
			c.Abort()
			return
		}

		parts := strings.Split(authorization, " ")
		if len(parts) != 2 {
			c.Error(cerr.NewUnauthorizedError("Invalid Authorization header", nil))
			c.Abort()
			return
		}

		if parts[0] != "Bearer" {
			c.Error(cerr.NewUnauthorizedError("Invalid Authorization header", nil))
			c.Abort()
			return
		}

		bearerToken := parts[1]

		// TODO: update this to use real auth service
		if bearerToken != "chariot$123" {
			c.Error(cerr.NewUnauthorizedError("Invalid Authorization", nil))
			c.Abort()
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
