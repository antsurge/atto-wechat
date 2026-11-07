package session

import "github.com/antsurge/atto-wechat/core/errors"

var (
	errInvalidCache = errors.New(
		errors.CodeInvalidArgument,
		"cache must not be nil",
	)

	errSessionNotFound = errors.New(
		errors.CodeTokenExpired,
		"session not found or expired",
	)
)
