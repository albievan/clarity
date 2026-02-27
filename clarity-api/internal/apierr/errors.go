package apierr

import (
	"errors"
	"net/http"
)

// Error codes — match OpenAPI spec error.code field.
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeUnprocessable       = "UNPROCESSABLE_ENTITY"
	CodeTooManyRequests     = "TOO_MANY_REQUESTS"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeLockedAccount       = "ACCOUNT_LOCKED"
	CodeMFARequired         = "MFA_REQUIRED"
)

// APIError is the canonical error type returned from service and repository layers.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	Details    []FieldError
}

// FieldError carries per-field validation failures.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Message }

// Constructors.

func BadRequest(msg string, fields ...FieldError) *APIError {
	return &APIError{http.StatusBadRequest, CodeBadRequest, msg, fields}
}

func Unauthorized(msg string) *APIError {
	return &APIError{http.StatusUnauthorized, CodeUnauthorized, msg, nil}
}

func Forbidden(msg string) *APIError {
	return &APIError{http.StatusForbidden, CodeForbidden, msg, nil}
}

func NotFound(entity string) *APIError {
	return &APIError{http.StatusNotFound, CodeNotFound, entity + " not found", nil}
}

func Conflict(msg string) *APIError {
	return &APIError{http.StatusConflict, CodeConflict, msg, nil}
}

func Unprocessable(msg string, fields ...FieldError) *APIError {
	return &APIError{http.StatusUnprocessableEntity, CodeUnprocessable, msg, fields}
}

func TooManyRequests(msg string) *APIError {
	return &APIError{http.StatusTooManyRequests, CodeTooManyRequests, msg, nil}
}

func Internal(msg string) *APIError {
	return &APIError{http.StatusInternalServerError, CodeInternalServerError, msg, nil}
}

func AccountLocked(msg string) *APIError {
	return &APIError{http.StatusLocked, CodeLockedAccount, msg, nil}
}

func MFARequired(mfaToken string) *APIError {
	e := &APIError{http.StatusOK, CodeMFARequired, "MFA verification required", nil}
	_ = mfaToken // caller handles mfa_token separately
	return e
}

// As unwraps to *APIError.
func As(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
