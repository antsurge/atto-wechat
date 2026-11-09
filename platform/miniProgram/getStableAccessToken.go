package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// stableTokenPath issues the stable app access token.
const (
	stableTokenPath                = "/cgi-bin/stable_token"
	StableAccessTokenDefaultErrMsg = "系统错误，请稍后再试"
)

// StableAccessTokenErrCodeMap holds the errcode descriptions WeChat documents
// for /cgi-bin/stable_token, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var StableAccessTokenErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40002: "grant_type 不合法",
	40013: "appid 无效",
	40125: "appsecret 无效",
	40164: "调用 IP 不在白名单内",
	41002: "缺少 appid",
	41004: "缺少 appsecret",
	43002: "需要使用 POST 请求",
	45009: "接口调用超过天级别上限",
	45011: "接口调用太频繁，请稍候再试",
	89503: "此次调用需要管理员确认，请耐心等候",
}

// StableAccessTokenResponse is the reply of /cgi-bin/stable_token.
type StableAccessTokenResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// OK reports whether the reply carries a usable token.
func (r *StableAccessTokenResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.AccessToken != ""
}

// StableAccessTokenErrMsg returns the description of a stable_token error.
func StableAccessTokenErrMsg(code int) string {
	if msg, ok := StableAccessTokenErrCodeMap[code]; ok {
		return msg
	}

	return StableAccessTokenDefaultErrMsg
}

// StableAccessTokenError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// StableAccessTokenErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func StableAccessTokenError(r *StableAccessTokenResponse) error {
	return errors.New(StableAccessTokenErrMsg(r.ErrCode))
}

// stableAccessTokenBody is the JSON body of /cgi-bin/stable_token.
type stableAccessTokenBody struct {
	GrantType    string `json:"grant_type"`
	AppID        string `json:"appid"`
	Secret       string `json:"secret"`
	ForceRefresh bool   `json:"force_refresh,omitempty"`
}

// GetStableAccessToken fetches an access token from the stable endpoint.
//
// It is the endpoint WeChat recommends over the /cgi-bin/token one behind
// AccessToken(): the two are fully isolated, so a token obtained here is neither
// stored in the token manager nor used by the other API calls of this package.
// Call it only when a token is needed outside this package.
//
// forceRefresh invalidates the token issued by the previous call, so it is a
// last resort: WeChat allows 20 forced refreshes per day with at least 30 seconds
// between them, and a forced refresh breaks every service still holding the old
// token. Leave it false and call again — in normal mode a token stays valid for
// its full 7200 seconds and repeated calls return the same value.
func (a *Application) GetStableAccessToken(
	ctx context.Context,
	forceRefresh bool,
) (*StableAccessTokenResponse, error) {
	// The endpoint authenticates with appid/secret, so it goes through the client
	// that does not attach a token.
	resp, err := a.base.PostJSON(ctx, stableTokenPath, nil, &stableAccessTokenBody{
		GrantType:    "client_credential",
		AppID:        a.cfg.AppID,
		Secret:       a.cfg.AppSecret,
		ForceRefresh: forceRefresh,
	})
	if err != nil {
		return nil, err
	}

	var result StableAccessTokenResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
