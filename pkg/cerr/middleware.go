package cerr

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger := c.Value("logger").(*zap.Logger)

		var sentError error
		for _, err := range c.Errors {
			// log, handle, etc.
			httpError := new(HttpError)
			if ok := errors.As(err, &httpError); ok {
				logger.Error(fmt.Sprintf("%s: %s", httpError.ErrorMsg, httpError.Message),
					zap.Error(httpError.Details),
					zap.Int("code", httpError.Code))
				if sentError == nil && !c.IsAborted() {
					c.JSON(httpError.Code, httpError)
					sentError = httpError
				}
			} else {
				logger.Error(err.Error())
				if sentError == nil && !c.IsAborted() {
					httpError := &HttpError{
						Timestamp: time.Now(),
						Code:      http.StatusInternalServerError,
						Message:   err.Error(),
						ErrorMsg:  "Internal server error",
						Details:   nil,
					}
					// status -1 doesn't overwrite existing status code
					c.JSON(http.StatusInternalServerError, httpError)
					sentError = err
				}
			}
		}
	}
}
