package middleware

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/antsurge/atto-wechat/core/config"
)

// redactedParams are query parameters whose values must never be logged. An
// access_token or a session_key in a log file is a credential leak: log
// aggregation outlives the token and is read by more people than the code.
var redactedParams = map[string]struct{}{
	"access_token":       {},
	"secret":             {},
	"session_key":        {},
	"js_code":            {},
	"encodingaeskey":     {},
	"encoding_aes_key":   {},
	"ticket":             {},
	"signature":          {},
	"msg_signature":      {},
}

// Logging returns a Middleware that logs each request and its outcome through
// the configured logger.
//
// Records are emitted only when cfg.LogLevel is slog.LevelInfo or more verbose,
// so the SDK's diagnostics can be silenced with
// config.WithLogLevel(slog.LevelWarn). The logger's own handler level still
// applies on top of that threshold.
func Logging(cfg *config.CoreConfig) Middleware {
	if cfg == nil {
		cfg = config.New()
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	verbose := cfg.LogLevel <= slog.LevelInfo

	return func(next Handler) Handler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			start := time.Now()

			if verbose {
				logger.InfoContext(ctx, "atto-wechat request",
					"method", req.Method,
					"path", req.Path,
					"query", redactQuery(req.Query),
				)
			}

			resp, err := next(ctx, req)

			status := 0
			if resp != nil {
				status = resp.StatusCode
			}

			if verbose {
				logger.InfoContext(ctx, "atto-wechat response",
					"method", req.Method,
					"path", req.Path,
					"status", status,
					"elapsed", time.Since(start).String(),
					"error", err,
				)
			}

			return resp, err
		}
	}
}

// redactQuery renders a query string with sensitive values replaced, so the log
// line stays useful (which keys were sent) without leaking the values.
func redactQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}

	safe := make(url.Values, len(q))
	for k, vs := range q {
		if _, secret := redactedParams[k]; secret {
			safe[k] = []string{"***"}
			continue
		}
		safe[k] = vs
	}

	return safe.Encode()
}
