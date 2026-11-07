// Package config holds core-level configuration shared across all WeChat
// platform modules.
//
// It is part of Tier 0 (zero internal dependencies): it only relies on the
// standard library, so any other package — client, auth, middleware, platform
// modules — can safely import it without creating import cycles.
package config

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Default infrastructure values applied by New when no Option overrides them.
const (
	// DefaultBaseURL is the WeChat API root.
	DefaultBaseURL = "https://api.weixin.qq.com"

	// DefaultHTTPTimeout is the per-request timeout.
	DefaultHTTPTimeout = 30 * time.Second

	// DefaultRetryAttempts is how many times a retryable error (see
	// core/errors.IsRetryable) is retried. 0 disables retries.
	DefaultRetryAttempts = 2

	// DefaultRetryBackoff is the base delay between retries.
	DefaultRetryBackoff = 200 * time.Millisecond

	// DefaultRetryMaxBackoff is the maximum delay between retries.
	DefaultRetryMaxBackoff = 10 * time.Second

	// DefaultUserAgent is sent on every request unless overridden.
	DefaultUserAgent = "atto-wechat/0.1"

	// DefaultLogLevel is the minimum level for the SDK's own diagnostics.
	DefaultLogLevel = slog.LevelInfo
)

// CoreConfig holds shared infrastructure settings used by the HTTP client and
// other core components. Platform-specific credentials (app_id, secret, token,
// aes_key) live in the respective platform/*/config.go instead.
type CoreConfig struct {
	// BaseURL is the WeChat API root. Defaults to DefaultBaseURL.
	BaseURL string

	// UserAgent is sent on every request. Defaults to DefaultUserAgent.
	UserAgent string

	// HTTPTimeout is the per-request timeout passed to http.Client.
	// Defaults to DefaultHTTPTimeout.
	HTTPTimeout time.Duration

	// RetryAttempts is the maximum number of retries on a retryable error
	// (see core/errors.IsRetryable). 0 disables retries.
	// Defaults to DefaultRetryAttempts.
	RetryAttempts int

	// RetryBackoff is the base delay between retries.
	// The actual delay may increase exponentially with each retry.
	// Defaults to DefaultRetryBackoff.
	RetryBackoff time.Duration

	// RetryMaxBackoff is the maximum delay between retries when using
	// exponential backoff. Defaults to DefaultRetryMaxBackoff.
	RetryMaxBackoff time.Duration

	// LogLevel is the verbosity threshold for the SDK's own request/response
	// diagnostics. Logging is emitted when it is slog.LevelInfo or more
	// verbose; set it to slog.LevelWarn or higher to silence the SDK.
	// The injected Logger's handler level still applies on top of this.
	// Defaults to DefaultLogLevel.
	LogLevel slog.Level

	// Logger is used by middleware for request/response logging.
	// Defaults to slog.Default().
	Logger *slog.Logger

	// HTTPClient, when non-nil, overrides the default client. It lets callers
	// inject a custom transport, proxy, or TLS configuration.
	HTTPClient *http.Client
}

// Option mutates a CoreConfig. Functional options keep construction clean and
// let new fields be added later without breaking callers.
type Option func(*CoreConfig)

// WithBaseURL sets the WeChat API root URL.
func WithBaseURL(url string) Option {
	return func(c *CoreConfig) {
		c.BaseURL = url
	}
}

// WithHTTPTimeout sets the per-request timeout.
func WithHTTPTimeout(d time.Duration) Option {
	return func(c *CoreConfig) {
		c.HTTPTimeout = d
	}
}

// WithUserAgent sets the User-Agent header value.
func WithUserAgent(ua string) Option {
	return func(c *CoreConfig) {
		c.UserAgent = ua
	}
}

// WithRetry configures retry behaviour for retryable errors.
func WithRetry(attempts int, backoff time.Duration) Option {
	return func(c *CoreConfig) {
		c.RetryAttempts = attempts
		c.RetryBackoff = backoff
	}
}

// WithRetryMaxBackoff sets the maximum delay between retries.
func WithRetryMaxBackoff(d time.Duration) Option {
	return func(c *CoreConfig) {
		c.RetryMaxBackoff = d
	}
}

// WithLogger sets the logger used by middleware.
func WithLogger(l *slog.Logger) Option {
	return func(c *CoreConfig) {
		if l != nil {
			c.Logger = l
		}
	}
}

// WithLogLevel sets the verbosity threshold for the SDK's own diagnostics.
func WithLogLevel(level slog.Level) Option {
	return func(c *CoreConfig) {
		c.LogLevel = level
	}
}

// WithHTTPClient injects a custom *http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *CoreConfig) {
		if client != nil {
			c.HTTPClient = client
		}
	}
}

// defaults returns a CoreConfig populated with default values.
func defaults() *CoreConfig {
	return &CoreConfig{
		BaseURL:         DefaultBaseURL,
		UserAgent:       DefaultUserAgent,
		HTTPTimeout:     DefaultHTTPTimeout,
		RetryAttempts:   DefaultRetryAttempts,
		RetryBackoff:    DefaultRetryBackoff,
		RetryMaxBackoff: DefaultRetryMaxBackoff,
		LogLevel:        DefaultLogLevel,
		Logger:          slog.Default(),
	}
}

// New builds a CoreConfig from the given options, applying defaults first.
func New(opts ...Option) *CoreConfig {
	c := defaults()

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	return c
}

// Validate checks required fields and normalizes the configuration. It returns
// a standard error (not a core/errors value) to keep this package free of
// internal dependencies.
func (c *CoreConfig) Validate() error {
	if c == nil {
		return errors.New("config: CoreConfig must not be nil")
	}

	if c.BaseURL == "" {
		return errors.New("config: BaseURL must not be empty")
	}

	if c.HTTPTimeout <= 0 {
		return errors.New("config: HTTPTimeout must be positive")
	}

	if c.RetryAttempts < 0 {
		return errors.New("config: RetryAttempts must not be negative")
	}

	if c.RetryBackoff < 0 {
		return errors.New("config: RetryBackoff must not be negative")
	}

	if c.RetryMaxBackoff < 0 {
		return errors.New("config: RetryMaxBackoff must not be negative")
	}

	if c.RetryMaxBackoff > 0 &&
		c.RetryBackoff > c.RetryMaxBackoff {
		return errors.New("config: RetryBackoff must not exceed RetryMaxBackoff")
	}

	return nil
}

// Clone returns a shallow copy of the config so platform modules can derive
// their own settings without mutating the shared instance. Logger and
// HTTPClient are intentionally shared (pointers), which is safe for reuse.
func (c *CoreConfig) Clone() *CoreConfig {
	if c == nil {
		return nil
	}

	cp := *c
	return &cp
}
