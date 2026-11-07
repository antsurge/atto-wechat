// Package oauth implements the Official Account web authorization flow (网页授权).
//
// It deliberately returns raw results instead of logging users in: obtaining a
// code, exchanging it for a token and reading the profile are platform calls,
// while "logging in" is application policy (which session, which cookie, which
// permissions). Callers compose the two.
package oauth

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/errors"
)

// Endpoint paths, relative to the client's BaseURL.
const (
	PathAccessToken  = "/sns/oauth2/access_token"
	PathRefreshToken = "/sns/oauth2/refresh_token"
	PathUserInfo     = "/sns/userinfo"
	PathCheckToken   = "/sns/auth"
)

// AuthorizationURL is the host that serves the consent page. It is not the API
// host, which is why the redirect is built here rather than by the client.
const AuthorizationURL = "https://open.weixin.qq.com/connect/oauth2/authorize"

// Scopes.
const (
	// ScopeBase silently obtains an openid.
	ScopeBase = "snsapi_base"
	// ScopeUserInfo additionally prompts the user and returns a profile.
	ScopeUserInfo = "snsapi_userinfo"
)

// Token is a web-authorization user token.
//
// It is not the API access_token: it authorizes calls on behalf of one user and
// is never valid as a query parameter for the management APIs.
type Token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid,omitempty"`

	// ExpiresAt is computed locally from ExpiresIn.
	ExpiresAt int64 `json:"-"`
}

// Expired reports whether the token has expired.
func (t *Token) Expired() bool {
	if t == nil || t.AccessToken == "" {
		return true
	}
	if t.ExpiresAt == 0 {
		return false
	}

	return time.Now().Unix() >= t.ExpiresAt
}

// UserInfo is the profile returned by /sns/userinfo.
type UserInfo struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid,omitempty"`
}

// Client performs the web authorization calls.
type Client struct {
	client *client.Client
	appID  string
	secret string
}

// NewClient builds an OAuth client.
//
// The supplied HTTP client must not carry the access_token middleware: OAuth
// calls use a user token in the query and would be rejected if an API token were
// appended under the same parameter name.
func NewClient(c *client.Client, appID, secret string) (*Client, error) {
	if c == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "oauth: client must not be nil")
	}
	if appID == "" || secret == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "oauth: appID and secret are required")
	}

	return &Client{client: c, appID: appID, secret: secret}, nil
}

// AuthorizationURL builds the URL to redirect the user to.
//
// state is echoed back unchanged; callers should put an unguessable value there
// and verify it on return, because the code is otherwise accepted from any
// browser that reaches the callback.
func (c *Client) AuthorizationURL(redirectURI, scope, state string) string {
	if scope == "" {
		scope = ScopeBase
	}

	q := url.Values{
		"appid":         {c.appID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {scope},
		"state":         {state},
	}

	// The fragment is part of the protocol; without it WeChat shows a warning
	// page instead of redirecting.
	return AuthorizationURL + "?" + q.Encode() + "#wechat_redirect"
}

// AccessToken exchanges an authorization code for a user token.
func (c *Client) AccessToken(ctx context.Context, code string) (*Token, error) {
	if code == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "oauth: code must not be empty")
	}

	return c.token(ctx, PathAccessToken, url.Values{
		"appid":      {c.appID},
		"secret":     {c.secret},
		"code":       {code},
		"grant_type": {"authorization_code"},
	})
}

// RefreshToken renews a user token.
//
// A refreshed token keeps the same openid but comes with a new refresh token, so
// callers must always store the returned one.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "oauth: refresh token must not be empty")
	}

	return c.token(ctx, PathRefreshToken, url.Values{
		"appid":         {c.appID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

// UserInfo reads the profile of the user behind token.
func (c *Client) UserInfo(ctx context.Context, token *Token) (*UserInfo, error) {
	if token == nil || token.AccessToken == "" || token.OpenID == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "oauth: token and openid are required")
	}

	resp, err := c.client.Get(ctx, PathUserInfo, url.Values{
		"access_token": {token.AccessToken},
		"openid":       {token.OpenID},
		"lang":         {"zh_CN"},
	})
	if err != nil {
		return nil, err
	}

	var raw struct {
		UserInfo
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := resp.DecodeJSON(&raw); err != nil {
		return nil, err
	}

	if raw.ErrCode != 0 {
		return nil, oauthError(raw.ErrCode, raw.ErrMsg)
	}

	if raw.OpenID == "" {
		return nil, errors.New(errors.CodeInvalidResponse, "oauth: userinfo response has no openid")
	}

	info := raw.UserInfo

	return &info, nil
}

// CheckToken reports whether token is still accepted.
func (c *Client) CheckToken(ctx context.Context, token *Token) error {
	if token == nil || token.AccessToken == "" || token.OpenID == "" {
		return errors.New(errors.CodeInvalidArgument, "oauth: token and openid are required")
	}

	resp, err := c.client.Get(ctx, PathCheckToken, url.Values{
		"access_token": {token.AccessToken},
		"openid":       {token.OpenID},
	})
	if err != nil {
		return err
	}

	var result oauthResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return oauthError(result.ErrCode, result.ErrMsg)
	}

	return nil
}

// token performs the shared exchange used by the two token endpoints.
func (c *Client) token(ctx context.Context, path string, query url.Values) (*Token, error) {
	resp, err := c.client.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Token
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := resp.DecodeJSON(&raw); err != nil {
		return nil, err
	}

	if raw.ErrCode != 0 {
		return nil, oauthError(raw.ErrCode, raw.ErrMsg)
	}

	if raw.AccessToken == "" {
		return nil, errors.New(errors.CodeInvalidResponse, "oauth: token response has no access_token")
	}

	tok := raw.Token
	if tok.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).Unix()
	}

	return &tok, nil
}

// oauthResponse is the minimal error-bearing reply.
type oauthResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// oauthError classifies a WeChat OAuth errcode.
//
// The mapping exists because these codes change what the caller must do: an
// invalid code means "send the user through authorization again", while a
// rejected token means "refresh", and a bad appid means "fix configuration".
func oauthError(code int, msg string) error {
	var class errors.Code

	switch code {
	case 40029, 40163, 42003, 41008:
		// code invalid, already used, expired, or missing.
		class = errors.CodeInvalidArgument
	case 40001, 40014, 41001, 42001:
		// token invalid or expired.
		class = errors.CodeTokenExpired
	case 40013, 40125:
		// bad appid or secret: a configuration error, not a user error.
		class = errors.CodeUnauthorized
	default:
		class = errors.CodeInvalidResponse
	}

	return errors.Newf(class, "oauth: errcode %s: %s", strconv.Itoa(code), msg)
}
