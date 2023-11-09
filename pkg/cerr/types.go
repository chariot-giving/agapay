package cerr

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HttpError struct {
	Type   string
	Title  string
	Status int
	Cause  error
}

func (he HttpError) MarshalJSON() ([]byte, error) {
	details := ""
	if he.Cause != nil {
		details = he.Cause.Error()
	}
	return json.Marshal(
		struct {
			Status int    `json:"status"`
			Type   string `json:"type"`
			Title  string `json:"title"`
			Detail string `json:"detail,omitempty"`
		}{he.Status, he.Type, he.Title, details})
}

func NewHttpError(code int, msg string, cause error) *HttpError {
	err := &HttpError{
		Status: code,
		Type:   http.StatusText(code),
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewBadRequest(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusBadRequest,
		Type:   "Bad Request",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewUnauthorizedError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusUnauthorized,
		Type:   "Unauthorized",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewForbiddenError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusForbidden,
		Type:   "Forbidden",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewNotFoundError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusNotFound,
		Type:   "Not Found",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewConflictError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusConflict,
		Type:   "Conflict",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewGoneError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusGone,
		Type:   "Gone",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewInternalServerError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusInternalServerError,
		Type:   "Internal server error",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func NewBadGatewayError(msg string, cause error) *HttpError {
	status, ok := status.FromError(cause)
	if ok {
		err := &HttpError{
			Status: grpcToHTTPStatusCode(status),
			Type:   status.Code().String(),
			Title:  msg,
		}
		if cause != nil {
			err.Cause = status.Err()
		}
		return err
	} else {
		err := &HttpError{
			Status: http.StatusBadGateway,
			Type:   "Bad Gateway",
			Title:  msg,
		}
		if cause != nil {
			err.Cause = cause
		}
		return err
	}
}

func NewDatabaseError(msg string, cause error) *HttpError {
	err := &HttpError{
		Status: http.StatusInternalServerError,
		Type:   "Internal server error",
		Title:  msg,
	}
	if cause != nil {
		err.Cause = cause
	}
	return err
}

func (h *HttpError) Error() string {
	return fmt.Sprintf("HttpError [%d]: %s; caused by: %s", h.Status, h.Title, h.Cause)
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
