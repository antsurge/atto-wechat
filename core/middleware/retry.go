package middleware

import (
	"context"
	"time"

	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/errors"
)

// Retry returns a Middleware that replays the request on transient failures.
//
// Only errors for which errors.IsRetryable is true are retried: the remote side
// was busy or rate limited. A timeout is not retried, because a timed-out
// request may already have been processed.
//
// Attempts and backoff come from cfg. A negative RetryAttempts counts as zero.
// The wait is exponential and capped by cfg.RetryMaxBackoff, and it returns
// immediately when the context is cancelled.
func Retry(cfg *config.CoreConfig) Middleware {
	if cfg == nil {
		cfg = config.New()
	}

	attempts := cfg.RetryAttempts
	if attempts < 0 {
		attempts = 0
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			backoff := cfg.RetryBackoff
			if backoff < 0 {
				backoff = 0
			}

			var (
				resp *Response
				err  error
			)

			for attempt := 0; ; attempt++ {
				resp, err = next(ctx, req)
				if err == nil || !errors.IsRetryable(err) || attempt >= attempts {
					return resp, err
				}

				if err := sleep(ctx, backoff); err != nil {
					return resp, err
				}

				backoff = nextBackoff(backoff, cfg.RetryMaxBackoff)
			}
		}
	}
}

// nextBackoff doubles d, capped by max. A non-positive max means no cap.
func nextBackoff(d, max time.Duration) time.Duration {
	next := d * 2
	if next <= 0 {
		return d
	}
	if max > 0 && next > max {
		return max
	}

	return next
}

// sleep waits for d, returning early with the context error when cancelled.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return errors.Wrap(errors.CodeCancelled, "request cancelled", ctx.Err())
		default:
			return nil
		}
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return errors.Wrap(errors.CodeCancelled, "request cancelled", ctx.Err())
	case <-timer.C:
		return nil
	}
}
