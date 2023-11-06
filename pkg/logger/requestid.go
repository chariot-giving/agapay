package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// headerXRequestID is the name of the HTTP header which contains the request ID.
	requestID = "x-request-id"
)

// New initializes the RequestID middleware.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get id from request
		rid := c.GetHeader(requestID)
		if rid == "" {
			rid = uuid.New().String()
			c.Request.Header.Add(requestID, rid)
		}
		// Set the id to ensure that the requestid is in the response
		c.Header(requestID, rid)
		c.Set(requestID, rid)
		c.Next()
	}
}
