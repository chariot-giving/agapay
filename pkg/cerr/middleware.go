package cerr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger := c.Value("logger").(*zap.Logger)

		c.Header("Content-Type", "application/json")

		var sentError error
		for _, err := range c.Errors {
			// log, handle, etc.
			httpError := new(HttpError)
			if ok := errors.As(err, &httpError); ok {
				logger.Error(fmt.Sprintf("%s: %s", httpError.Type, httpError.Title),
					zap.Error(httpError.Cause),
					zap.Int("code", httpError.Status))
				if sentError == nil {
					// status -1 doesn't overwrite existing status code
					status := httpError.Status
					if c.IsAborted() {
						status = c.Writer.Status()
					}
					c.JSON(status, httpError)
					sentError = httpError
				}
			} else {
				logger.Error(err.Error())
				if sentError == nil {
					httpError := &HttpError{
						Status: http.StatusInternalServerError,
						Type:   "Internal server error",
						Title:  err.Error(),
						Cause:  nil,
					}
					// status -1 doesn't overwrite existing status code
					status := httpError.Status
					if c.IsAborted() {
						status = c.Writer.Status()
					}
					c.JSON(status, httpError)
					sentError = err
				}
			}
		}
	}
}
