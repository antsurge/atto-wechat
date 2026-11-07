// Package errors defines the unified error type used across atto-wechat.
//
// It is part of Tier 0: it depends only on the standard library, so every other
// package may import it without creating a cycle.
//
// A WeChat reply that carries a business errcode is a response, not an error:
// callers decode it into one of the core/response types and decide what to do.
// This package only describes failures of the call itself -- transport,
// serialization, credentials, cryptography.
package errors

import (
	stderrors "errors"
	"fmt"
	"log/slog"
)

// Error represents an SDK error.
type Error struct {
	// Code is the SDK classification of the failure.
	Code Code

	// Message is a human-readable description.
	Message string

	// HTTPStatus is the HTTP status of the response, or 0 when unknown.
	HTTPStatus int

	// Cause is the underlying error, if any.
	Cause error
}

// New creates a new SDK error.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new SDK error with a formatted message.
func Newf(code Code, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap creates an SDK error with an underlying cause.
//
// It returns nil when cause is nil: wrapping nothing would fabricate a failure
// that never happened.
func Wrap(code Code, message string, cause error) *Error {
	if cause == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf is like Wrap but formats the message.
func Wrapf(code Code, cause error, format string, args ...any) *Error {
	if cause == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// Error implements the standard error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	msg := string(e.Code) + ": " + e.Message
	if e.HTTPStatus != 0 {
		msg = fmt.Sprintf("%s (http %d)", msg, e.HTTPStatus)
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}

	return msg
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

// LogValue implements slog.LogValuer so a *Error expands into structured fields
// instead of a single opaque string. The package still does not depend on any
// logger: the method only describes the value.
func (e *Error) LogValue() slog.Value {
	if e == nil {
		return slog.GroupValue()
	}

	attrs := []slog.Attr{
		slog.String("code", string(e.Code)),
		slog.String("message", e.Message),
	}
	if e.HTTPStatus != 0 {
		attrs = append(attrs, slog.Int("http_status", e.HTTPStatus))
	}
	if e.Cause != nil {
		attrs = append(attrs, slog.String("cause", e.Cause.Error()))
	}

	return slog.GroupValue(attrs...)
}

// As extracts an *Error from err, unwrapping as needed.
func As(err error) (*Error, bool) {
	var target *Error

	if !stderrors.As(err, &target) {
		return nil, false
	}

	return target, true
}

// Is reports whether the error has the specified SDK error code.
func Is(err error, code Code) bool {
	var target *Error

	if !stderrors.As(err, &target) {
		return false
	}

	return target.Code == code
}

// CodeOf extracts the SDK error code.
// Returns CodeUnknown when the error is not an SDK error.
func CodeOf(err error) Code {
	var target *Error

	if !stderrors.As(err, &target) {
		return CodeUnknown
	}

	return target.Code
}

// IsRetryable reports whether err is a transient failure worth replaying: the
// remote side was busy or rate limited.
//
// A timeout is deliberately excluded. Replaying is only safe for requests that
// were not processed, and a timed-out request may well have been.
func IsRetryable(err error) bool {
	switch CodeOf(err) {
	case CodeSystemBusy, CodeRateLimited:
		return true
	default:
		return false
	}
}

// IsTokenExpired reports whether err indicates that the access token was
// rejected, which the auth layer recovers from by refreshing and replaying once.
func IsTokenExpired(err error) bool {
	switch CodeOf(err) {
	case CodeTokenExpired, CodeTokenInvalid:
		return true
	default:
		return false
	}
}

// IsAuthError reports whether err is a credential or permission problem that
// retrying cannot fix.
func IsAuthError(err error) bool {
	switch CodeOf(err) {
	case CodeUnauthorized, CodeForbidden:
		return true
	default:
		return false
	}
}

// IsParamError reports whether err is caused by malformed input that the caller
// must fix before trying again.
func IsParamError(err error) bool {
	return CodeOf(err) == CodeInvalidArgument
}
