package cerr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HttpError struct {
	Timestamp time.Time
	Code      int
	Message   string
	ErrorMsg  string
	Details   error
}

func (he HttpError) MarshalJSON() ([]byte, error) {
	details := ""
	if he.Details != nil {
		details = he.Details.Error()
	}
	return json.Marshal(
		struct {
			Timestamp time.Time `json:"timestamp"`
			Code      int       `json:"code"`
			Message   string    `json:"message"`
			ErrorMsg  string    `json:"error"`
			Details   string    `json:"details,omitempty"`
		}{he.Timestamp, he.Code, he.Message, he.ErrorMsg, details})
}

func NewHttpError(code int, msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      code,
		ErrorMsg:  http.StatusText(code),
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewBadRequest(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusBadRequest,
		ErrorMsg:  "Bad Request",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewUnauthorizedError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusUnauthorized,
		ErrorMsg:  "Unauthorized",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewForbiddenError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusForbidden,
		ErrorMsg:  "Forbidden",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewNotFoundError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusNotFound,
		ErrorMsg:  "Not Found",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewConflictError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusConflict,
		ErrorMsg:  "Conflict",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewGoneError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusGone,
		ErrorMsg:  "Gone",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewInternalServerError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusInternalServerError,
		ErrorMsg:  "Internal server error",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func NewBadGatewayError(msg string, cause error) *HttpError {
	status, ok := status.FromError(cause)
	if ok {
		err := &HttpError{
			Timestamp: time.Now(),
			Code:      grpcToHTTPStatusCode(status),
			ErrorMsg:  status.Code().String(),
			Message:   msg,
		}
		if cause != nil {
			err.Details = status.Err()
		}
		return err
	} else {
		err := &HttpError{
			Timestamp: time.Now(),
			Code:      http.StatusBadGateway,
			ErrorMsg:  "Bad Gateway",
			Message:   msg,
		}
		if cause != nil {
			err.Details = cause
		}
		return err
	}
}

func NewDatabaseError(msg string, cause error) *HttpError {
	err := &HttpError{
		Timestamp: time.Now(),
		Code:      http.StatusInternalServerError,
		ErrorMsg:  "Internal server error",
		Message:   msg,
	}
	if cause != nil {
		err.Details = cause
	}
	return err
}

func (h *HttpError) Error() string {
	return fmt.Sprintf("HttpError [%d]: %s; caused by: %s", h.Code, h.Message, h.Details)
}

func grpcToHTTPStatusCode(s *status.Status) int {
	switch s.Code() {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusRequestedRangeNotSatisfiable
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
