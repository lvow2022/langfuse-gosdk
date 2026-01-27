package langfuse

import (
	"fmt"
	"net/http"
)

// LangfuseError represents a Langfuse-specific error with retry information
type LangfuseError struct {
	Code       string
	Message    string
	StatusCode int
	retryable  bool
}

// Error implements the error interface
func (e *LangfuseError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("langfuse: %s (HTTP %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("langfuse: %s: %s", e.Code, e.Message)
}

// IsRetryable returns whether this error is retryable
func (e *LangfuseError) IsRetryable() bool {
	return e.retryable
}

// Unwrap returns the underlying error for use with errors.Is and errors.As
func (e *LangfuseError) Unwrap() error {
	if e.Message != "" {
		return fmt.Errorf(e.Message)
	}
	return nil
}

// NewHTTPError creates a new LangfuseError from an HTTP status code and body
// HTTP 429 (Too Many Requests) and 5xx errors are considered retryable
// HTTP 4xx errors (except 429) are not retryable
func NewHTTPError(statusCode int, body string) *LangfuseError {
	retryable := statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode < 600)

	code := fmt.Sprintf("HTTP_%d", statusCode)
	if statusCode >= 500 && statusCode < 600 {
		code = "SERVER_ERROR"
	} else if statusCode == http.StatusTooManyRequests {
		code = "RATE_LIMITED"
	} else if statusCode >= 400 && statusCode < 500 {
		code = "CLIENT_ERROR"
	}

	return &LangfuseError{
		Code:       code,
		Message:    body,
		StatusCode: statusCode,
		retryable:  retryable,
	}
}

// NewNetworkError creates a new retryable LangfuseError for network failures
func NewNetworkError(err error) *LangfuseError {
	return &LangfuseError{
		Code:      "NETWORK_ERROR",
		Message:   err.Error(),
		retryable: true,
	}
}

// NewConfigError creates a new non-retryable LangfuseError for configuration issues
func NewConfigError(message string) *LangfuseError {
	return &LangfuseError{
		Code:      "CONFIG_ERROR",
		Message:   message,
		retryable: false,
	}
}

// IsRetryableError checks if an error is retryable
// Returns true if err is a LangfuseError and IsRetryable() returns true
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if langfuseErr, ok := err.(*LangfuseError); ok {
		return langfuseErr.IsRetryable()
	}
	return false
}
