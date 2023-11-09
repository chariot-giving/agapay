package telemetry

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// headerXRequestID is the name of the HTTP header which contains the request ID.
	RequestID = "x-request-id"
	OriginIP  = "origin_ip"
)

// New initializes the RequestID middleware.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get id from request
		rid := c.GetHeader(RequestID)
		if rid == "" {
			rid = uuid.New().String()
			c.Request.Header.Add(RequestID, rid)
		}
		// Set the id to ensure that the requestid is in the response
		c.Header(RequestID, rid)
		c.Set(RequestID, rid)
		c.Set("origin_ip", c.ClientIP())
		c.Next()
	}
}
