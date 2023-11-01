package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// New initializes the Logging middleware.
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.Named("http").With(
			zap.String("path", c.Request.URL.Path),
			zap.String(requestID, c.GetString(requestID)),
		)
		log.Debug("processing request")
		c.Set("logger", log)
		newC := context.WithValue(c, "logger", log)
		c.Request = c.Request.WithContext(newC)
		c.Next()
	}
}
