// Package miniProgram wires the core packages into a mini program application:
// a ready HTTP client with token injection, a token manager, a session manager
// and an optional callback server.
//
// It adds configuration and composition only. Every mechanism -- transport,
// token lifecycle, sessions, callbacks -- lives in core and stays usable on its
// own.
package miniProgram

import (
	"context"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/event"
	"github.com/antsurge/atto-wechat/core/middleware"
)

// DefaultBaseURL is the WeChat API root used by mini programs.
const DefaultBaseURL = "https://api.weixin.qq.com"

// Config describes one mini program application.
//
// AppID and AppSecret are required; everything else has a working default. Token
// and AESKey are optional and only enable the callback endpoint.
type Config struct {
	// AppID is the mini program's AppID.
	AppID string
	// AppSecret is the mini program's AppSecret.
	AppSecret string

	// Token is the callback token configured in the console.
	Token string
	// AESKey is the 43-character EncodingAESKey. Empty means plaintext mode.
	AESKey string

	// CoreOptions tune the shared HTTP infrastructure: timeout, retry, logger,
	// base URL override, custom *http.Client.
	CoreOptions []config.Option

	// Cache backs token, ticket and session storage. Defaults to a private
	// in-memory cache, which is fine for a single process but loses state on
	// restart and is not shared across instances.
	Cache cache.Cache

	// Middleware replaces the default HTTP middleware chain. When nil, logging
	// and retry are installed. Token injection is always appended last and cannot
	// be removed: the application's API client exists to carry it.
	Middleware []middleware.Middleware

	// SessionTTL is the lifetime of a login session. Defaults to
	// session.DefaultTTL.
	SessionTTL time.Duration

	// DedupStore remembers processed callbacks. Defaults to Cache when it
	// implements cache.SetNXer, otherwise deduplication is disabled.
	DedupStore cache.SetNXer
	// DedupTTL is how long a processed callback is remembered.
	DedupTTL time.Duration

	// OnHandlerError observes callback handler failures without changing the
	// reply WeChat receives.
	OnHandlerError func(ctx context.Context, msg *event.Message, err error)
}
