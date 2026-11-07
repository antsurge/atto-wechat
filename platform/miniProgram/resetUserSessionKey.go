package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// Path, signature algorithm and fallback description of the session reset.
const (
	resetUserSessionKeyPath          = "/wxa/resetusersessionkey"
	resetUserSessionKeySigMethod     = "hmac_sha256"
	ResetUserSessionKeyDefaultErrMsg = "系统错误，请稍后再试"
)

// ResetUserSessionKeyErrCodeMap holds the errcode descriptions WeChat documents
// for /wxa/resetusersessionkey, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var ResetUserSessionKeyErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40097: "参数错误",
	45011: "API 调用太频繁，请稍候再试",
	87007: "session_key 不存在或已过期",
	87008: "不支持的签名算法",
	87009: "签名错误",
}

// ResetUserSessionKeyResponse is the reply of /wxa/resetusersessionkey.
//
// On success WeChat returns the openid and the new session_key, which inherits
// the expiry of the session_key it replaces: resetting never extends a login.
type ResetUserSessionKeyResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OpenID     string `json:"openid,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
}

// OK reports whether the session key was reset.
func (r *ResetUserSessionKeyResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ResetUserSessionKeyErrMsg returns the description of a resetusersessionkey
// error.
func ResetUserSessionKeyErrMsg(code int) string {
	if msg, ok := ResetUserSessionKeyErrCodeMap[code]; ok {
		return msg
	}
	return ResetUserSessionKeyDefaultErrMsg
}

// ResetUserSessionKeyError turns the errcode from the reply into an error.
func ResetUserSessionKeyError(r *ResetUserSessionKeyResponse) error {
	return errors.New(ResetUserSessionKeyErrMsg(r.ErrCode))
}

// ResetUserSessionKey calls /wxa/resetusersessionkey for openID.
//
// The old session_key stops working immediately; the reply carries a new one
// with the same expiry, and the local session is updated to hold it so later
// checks and decryptions keep working. A call is successful when the reply
// reports errcode 0; use ResetUserSessionKeyError(result) for any other.
// signature is the value produced by CheckSessionSignature with the session_key
// stored server side, not one sent by the client.
func (a *Application) ResetUserSessionKey(
	ctx context.Context,
	openID string,
	signature string,
) (*ResetUserSessionKeyResponse, error) {
	if openID == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: openid must not be empty")
	}
	if signature == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: signature must not be empty")
	}

	resp, err := a.api.Get(ctx, resetUserSessionKeyPath, url.Values{
		"openid":     {openID},
		"signature":  {signature},
		"sig_method": {resetUserSessionKeySigMethod},
	})
	if err != nil {
		return nil, err
	}

	var result ResetUserSessionKeyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	if result.OK() && result.SessionKey != "" {
		// The session still holds the replaced key, which no longer decrypts
		// anything: swap it without touching the expiry WeChat kept.
		resetOpenID := result.OpenID
		if resetOpenID == "" {
			resetOpenID = openID
		}

		a.sessions.SetSessionKey(resetOpenID, result.SessionKey)
	}

	return &result, nil
}
