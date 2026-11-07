package auth

import (
	"context"
	"net/url"
	"time"

	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/errors"
)

// errEmptyToken indicates an empty access_token in a successful response.
var errEmptyToken = errors.New(
	errors.CodeInvalidResponse,
	"empty access_token in token response",
)

// TokenProvider defines the source of a fresh access token.
type TokenProvider interface {
	// FetchToken fetches a fresh token from the remote platform.
	FetchToken(ctx context.Context) (*Token, error)
}

// tokenResponse represents the common WeChat token API response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// HTTPTokenProvider fetches access tokens through HTTP.
type HTTPTokenProvider struct {
	client *client.Client
	path   string
	params url.Values
}

// NewHTTPTokenProvider creates a generic HTTP token provider.
//
// It supports platforms whose token endpoints return the standard
// access_token and expires_in fields.
//
// params contains fixed query parameters, such as:
//   - OfficialAccount / MiniProgram: appid, secret, grant_type
//   - Work: corpid, corpsecret
func NewHTTPTokenProvider(
	c *client.Client,
	path string,
	params url.Values,
) *HTTPTokenProvider {
	return &HTTPTokenProvider{
		client: c,
		path:   path,
		params: cloneValues(params),
	}
}

// FetchToken performs an HTTP request and parses the token response.
func (p *HTTPTokenProvider) FetchToken(ctx context.Context) (*Token, error) {
	if p == nil || p.client == nil {
		return nil, errInvalidTokenProvider
	}

	if p.path == "" {
		return nil, errInvalidTokenProvider
	}

	resp, err := p.client.Get(ctx, p.path, p.params)
	if err != nil {
		return nil, err
	}

	var result tokenResponse

	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	if result.AccessToken == "" {
		// A rejected credential surfaces here as an empty token. Reporting the
		// errcode in the message is the only way the caller can tell a bad
		// secret (40125) from an IP whitelist problem (40164).
		if result.ErrCode != 0 {
			return nil, errors.Newf(
				errors.CodeInvalidResponse,
				"token request failed: errcode %d: %s",
				result.ErrCode,
				result.ErrMsg,
			)
		}

		return nil, errEmptyToken
	}

	// A non-positive expiry is not a usable token lifetime.
	if result.ExpiresIn <= 0 {
		return nil, errInvalidTokenExpiry
	}

	return &Token{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
		ExpiresAt: time.Now().Add(
			time.Duration(result.ExpiresIn) * time.Second,
		).Unix(),
	}, nil
}

// cloneValues deep-copies a url.Values.
//
// The copy is what keeps a caller's map from being mutated later: the provider
// holds its credentials for the lifetime of the client, and a shared map would
// let any holder rewrite them.
func cloneValues(v url.Values) url.Values {
	if v == nil {
		return nil
	}

	out := make(url.Values, len(v))
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}

	return out
}
