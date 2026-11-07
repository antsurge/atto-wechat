package auth

import (
	"github.com/antsurge/atto-wechat/core/errors"
)

var (
	errInvalidTokenProvider = errors.New(
		errors.CodeInvalidArgument,
		"invalid token provider",
	)

	errInvalidCache = errors.New(
		errors.CodeInvalidArgument,
		"cache must not be nil",
	)

	errInvalidTokenExpiry = errors.New(
		errors.CodeInvalidResponse,
		"invalid token expiry in response",
	)

	errInvalidTicketProvider = errors.New(
		errors.CodeInvalidArgument,
		"invalid ticket provider",
	)

	errEmptyTicket = errors.New(
		errors.CodeInvalidResponse,
		"empty ticket in response",
	)
)
