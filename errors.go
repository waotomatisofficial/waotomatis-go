package waotomatis

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, snake_case error code — the public contract shared by
// the server, the SDK, and the MCP tools (see apps/api/src/lib/errors.ts).
// Branch on the code (via Code(err)) for stable handling across API versions.
type ErrorCode string

// Server-returned codes.
const (
	CodeUnauthorized              ErrorCode = "unauthorized"
	CodeForbiddenScope            ErrorCode = "forbidden_scope"
	CodeInsufficientPermissions   ErrorCode = "insufficient_permissions"
	CodeValidationFailed          ErrorCode = "validation_failed"
	CodeOrgNotFound               ErrorCode = "org_not_found"
	CodeTeamNotFound              ErrorCode = "team_not_found"
	CodeMemberNotFound            ErrorCode = "member_not_found"
	CodeChatNotFound              ErrorCode = "chat_not_found"
	CodeContactNotFound           ErrorCode = "contact_not_found"
	CodeSessionNotFound           ErrorCode = "session_not_found"
	CodeSessionDisconnected       ErrorCode = "session_disconnected"
	CodeOnboardingFailed          ErrorCode = "onboarding_failed"
	CodeMessageNotFound           ErrorCode = "message_not_found"
	CodeMediaNotFound             ErrorCode = "media_not_found"
	CodeUnsupportedMessageType    ErrorCode = "unsupported_message_type"
	CodeSendFailed                ErrorCode = "send_failed"
	CodeWebhookNotFound           ErrorCode = "webhook_not_found"
	CodeMetaAPIError              ErrorCode = "meta_api_error"
	CodeStorageQuotaExceeded      ErrorCode = "storage_quota_exceeded"
	CodeStorageObjectNotFound     ErrorCode = "storage_object_not_found"
	CodeStorageConnectionNotFound ErrorCode = "storage_connection_not_found"
	CodeStorageConnectionFailed   ErrorCode = "storage_connection_failed"
	CodeNotFound                  ErrorCode = "not_found"
	CodeRateLimited               ErrorCode = "rate_limited"
	CodeInternalError             ErrorCode = "internal_error"

	// Client-side codes (never returned by the server).
	CodeTimeout         ErrorCode = "timeout"
	CodeConnectionError ErrorCode = "connection_error"
)

// Error is the base error for every failure surfaced by the SDK. It mirrors the
// server's uniform error model { error: { code, message, requestId } } plus the
// HTTP status. Every typed error in this package wraps an *Error, so callers can
// use errors.As to recover it, or the package helpers Code/Status/RequestID.
//
//	_, err := wao.Sessions(id).Messages.SendText(msg)
//	if err != nil {
//	    var rl *waotomatis.RateLimitError
//	    if errors.As(err, &rl) {
//	        time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
//	    }
//	    if waotomatis.Code(err) == waotomatis.CodeSessionDisconnected {
//	        // ...
//	    }
//	}
type Error struct {
	// Code is the stable, machine-readable error code.
	Code ErrorCode
	// Message is the human-readable error message from the server.
	Message string
	// RequestID correlates the failure with server logs (empty if absent).
	RequestID string
	// Status is the HTTP status code (0 for client-side network/timeout errors).
	Status int
}

func (e *Error) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("waotomatis: %s (code=%s, status=%d, request_id=%s)", e.Message, e.Code, e.Status, e.RequestID)
	}
	return fmt.Sprintf("waotomatis: %s (code=%s, status=%d)", e.Message, e.Code, e.Status)
}

// The typed errors below each carry a *Error (accessible via Base) and satisfy
// the error interface by delegating to it. They Unwrap to the *Error so
// errors.As(err, **Error) and errors.As(err, **AuthenticationError) both work.

// AuthenticationError is returned on 401 — missing or invalid API key.
type AuthenticationError struct{ err *Error }

func (e *AuthenticationError) Error() string { return e.err.Error() }
func (e *AuthenticationError) Unwrap() error { return e.err }

// Base returns the underlying *Error (code, message, request id, status).
func (e *AuthenticationError) Base() *Error { return e.err }

// PermissionError is returned on 403 — the key/user is not permitted to perform
// this action.
type PermissionError struct{ err *Error }

func (e *PermissionError) Error() string { return e.err.Error() }
func (e *PermissionError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *PermissionError) Base() *Error { return e.err }

// NotFoundError is returned on 404 — the addressed resource does not exist (or
// isn't visible to this key).
type NotFoundError struct{ err *Error }

func (e *NotFoundError) Error() string { return e.err.Error() }
func (e *NotFoundError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *NotFoundError) Base() *Error { return e.err }

// ValidationError is returned on 409 / 422 — the request was understood but
// rejected (validation failure or bad state).
type ValidationError struct{ err *Error }

func (e *ValidationError) Error() string { return e.err.Error() }
func (e *ValidationError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *ValidationError) Base() *Error { return e.err }

// RateLimitError is returned on 429. RetryAfter (seconds) is parsed from the
// Retry-After response header when present (0 otherwise).
type RateLimitError struct {
	err *Error
	// RetryAfter is the number of seconds to wait before retrying, or 0.
	RetryAfter int
}

func (e *RateLimitError) Error() string { return e.err.Error() }
func (e *RateLimitError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *RateLimitError) Base() *Error { return e.err }

// APIError is returned on 5xx — an unexpected server-side failure. It is safe
// to retry idempotent calls.
type APIError struct{ err *Error }

func (e *APIError) Error() string { return e.err.Error() }
func (e *APIError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *APIError) Base() *Error { return e.err }

// ConnectionError wraps a network/transport failure that occurred before a
// response was received.
type ConnectionError struct{ err *Error }

func (e *ConnectionError) Error() string { return e.err.Error() }
func (e *ConnectionError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *ConnectionError) Base() *Error { return e.err }

// TimeoutError indicates the request exceeded the configured timeout or the
// caller's context deadline.
type TimeoutError struct{ err *Error }

func (e *TimeoutError) Error() string { return e.err.Error() }
func (e *TimeoutError) Unwrap() error { return e.err }

// Base returns the underlying *Error.
func (e *TimeoutError) Base() *Error { return e.err }

// Code returns the ErrorCode of any error produced by this SDK, or an empty
// ErrorCode if err is nil or not a waotomatis error.
func Code(err error) ErrorCode {
	if e := asBase(err); e != nil {
		return e.Code
	}
	return ""
}

// Status returns the HTTP status of a waotomatis error (0 if none / client-side).
func Status(err error) int {
	if e := asBase(err); e != nil {
		return e.Status
	}
	return 0
}

// RequestID returns the server request id of a waotomatis error ("" if none).
func RequestID(err error) string {
	if e := asBase(err); e != nil {
		return e.RequestID
	}
	return ""
}

func asBase(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// errorFromStatus constructs the right typed error for an HTTP error response,
// mirroring errorFromStatus in packages/sdk/src/errors.ts.
func errorFromStatus(status int, code ErrorCode, message, requestID string, retryAfter int) error {
	if code == "" {
		code = CodeInternalError
	}
	if message == "" {
		message = "Request failed."
	}
	base := &Error{Code: code, Message: message, RequestID: requestID, Status: status}
	switch {
	case status == 401:
		return &AuthenticationError{base}
	case status == 403:
		return &PermissionError{base}
	case status == 404:
		return &NotFoundError{base}
	case status == 408:
		base.Code = CodeTimeout
		return &TimeoutError{base}
	case status == 409 || status == 422:
		return &ValidationError{base}
	case status == 429:
		return &RateLimitError{err: base, RetryAfter: retryAfter}
	case status >= 500:
		return &APIError{base}
	default:
		// Other 4xx — surface as the base error with its code.
		return base
	}
}

func newConnectionError(message string) *ConnectionError {
	return &ConnectionError{&Error{Code: CodeConnectionError, Message: message}}
}

func newTimeoutError(message string) *TimeoutError {
	if message == "" {
		message = "Request timed out."
	}
	return &TimeoutError{&Error{Code: CodeTimeout, Message: message, Status: 408}}
}
