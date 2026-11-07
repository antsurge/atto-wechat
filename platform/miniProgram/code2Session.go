package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// code2SessionPath is the login endpoint: it exchanges a temporary code issued
// by wx.login for the user's identifiers and a session key.
const (
	code2SessionPath          = "/sns/jscode2session"
	Code2SessionDefaultErrMsg = "系统错误，请稍后再试"
)

// Code2SessionErrCodeMap holds the errcode descriptions WeChat documents for
// /sns/jscode2session, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var Code2SessionErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40029: "code 无效",
	40226: "高风险等级用户，小程序登录拦截",
	45011: "API 调用太频繁，请稍候再试",
}

// Code2SessionResponse is the reply of /sns/jscode2session.
//
// The structure keeps the two kinds of parameters apart: the embedded common
// header is what every WeChat API returns (errcode/errmsg), while the fields
// below are specific to this endpoint.
type Code2SessionResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid,omitempty"`
	SessionKey string `json:"session_key"`
}

// Code2Session exchanges a login code for the user's identifiers and session key.
//
// The session_key must never be returned to the client: it is what makes the
// encrypted user data decryptable, so leaking it hands over the user's phone
// number to anyone who can read the response.
func (a *Application) Code2Session(
	ctx context.Context,
	code string,
) (*Code2SessionResponse, error) {
	if code == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: code must not be empty")
	}

	resp, err := a.base.Get(ctx, code2SessionPath, url.Values{
		"appid":      {a.cfg.AppID},
		"secret":     {a.cfg.AppSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	})
	if err != nil {
		return nil, err
	}

	var result Code2SessionResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// OK reports whether the response contains the required session credentials.
func (r *Code2SessionResponse) OK() bool {
	return r != nil &&
		r.ErrCode == 0 &&
		r.OpenID != "" &&
		r.SessionKey != ""
}

// Code2SessionErrMsg returns the description of a WeChat code2session error.
func Code2SessionErrMsg(code int) string {
	if msg, ok := Code2SessionErrCodeMap[code]; ok {
		return msg
	}
	return Code2SessionDefaultErrMsg
}

// code2SessionError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// Code2SessionErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func Code2SessionError(r *Code2SessionResponse) error {
	return errors.New(Code2SessionErrMsg(r.ErrCode))
}
