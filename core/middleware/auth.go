package middleware

import (
	"context"
	"net/url"

	"github.com/antsurge/atto-wechat/core/codec"
)

// AccessTokenParam is the query parameter WeChat expects for the API access
// token.
const AccessTokenParam = "access_token"

// tokenExpiryCodes are the errcode values WeChat returns when the access_token
// was rejected: expired, invalid, missing, or revoked.
var tokenExpiryCodes = map[int]struct{}{
	40001: {},
	40014: {},
	41001: {},
	42001: {},
	42007: {},
}

// TokenSource supplies access tokens to WithToken.
//
// The interface is declared here rather than imported from core/auth because
// core/auth depends on core/client, which depends on core/middleware; importing
// it back would create a cycle. auth.AccessTokenManager satisfies it
// structurally, so callers pass it directly.
type TokenSource interface {
	// Get returns a currently valid token, fetching one if necessary.
	Get(ctx context.Context) (string, error)
	// Refresh forces a new token, invalidating the cached one.
	Refresh(ctx context.Context) (string, error)
}

// WithToken injects the access token into every request's query string and
// recovers from a rejected token by refreshing it and replaying the request
// exactly once.
//
// The replay lives here, in the auth layer, because only this layer knows the
// token lifecycle and can guarantee a single replay. A request rejected for an
// expired token was not processed, so replaying it cannot double-apply.
//
// Do not use this middleware for WeChat Pay v3: those requests carry no
// access_token, and injecting one would change the URL covered by the request
// signature.
func WithToken(src TokenSource) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			// A request marked public must neither carry the token nor trigger
			// a fetch: an image download should not fail because credentials
			// are misconfigured.
			if req.NoAuth {
				return next(ctx, req)
			}

			token, err := src.Get(ctx)
			if err != nil {
				return nil, err
			}

			// Work on a copy: the caller may reuse the Request, and mutating
			// Query in place would carry a stale token into later calls.
			resp, err := next(ctx, withQueryParam(req, AccessTokenParam, token))
			if err != nil || !tokenRejected(resp) {
				return resp, err
			}

			// The token looked valid locally but WeChat refused it -- revoked,
			// or expired between the cached expiry check and the remote check.
			newToken, refreshErr := src.Refresh(ctx)
			if refreshErr != nil {
				// Report the original rejection: the token is the problem the
				// caller needs to see, and the refresh failure is a consequence.
				return resp, nil
			}

			return next(ctx, withQueryParam(req, AccessTokenParam, newToken))
		}
	}
}

// tokenRejected reports whether resp is a WeChat reply rejecting the token.
//
// The client deliberately does not turn errcodes into errors -- a business
// rejection is a response -- so the auth layer inspects the reply itself.
func tokenRejected(resp *Response) bool {
	if resp == nil || len(resp.Body) == 0 {
		return false
	}

	var probe struct {
		ErrCode int `json:"errcode"`
	}
	if err := codec.Unmarshal(resp.Body, &probe); err != nil {
		return false
	}

	_, ok := tokenExpiryCodes[probe.ErrCode]

	return ok
}

// withQueryParam returns a shallow copy of req whose query has key set to
// value, leaving the original Request and its Query untouched.
func withQueryParam(req *Request, key, value string) *Request {
	cp := *req

	q := make(url.Values, len(req.Query)+1)
	for k, vs := range req.Query {
		q[k] = append([]string(nil), vs...)
	}
	q.Set(key, value)
	cp.Query = q

	return &cp
}
