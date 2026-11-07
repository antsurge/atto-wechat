package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
	"github.com/antsurge/atto-wechat/core/security"
)

// Path, signature algorithm and fallback description of the session check.
const (
	checkSessionKeyPath          = "/wxa/checksession"
	checkSessionKeySigMethod     = "hmac_sha256"
	CheckSessionKeyDefaultErrMsg = "系统错误，请稍后再试"
)

// CheckSessionKeyErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/checksession, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var CheckSessionKeyErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40003: "openid 无效",
	45011: "API 调用太频繁，请稍候再试",
	87009: "签名错误",
}

// CheckSessionKeyResponse is the reply of /wxa/checksession.
//
// The endpoint only answers with the common header: errcode 0 means the login
// state is still valid, anything else means it is not.
type CheckSessionKeyResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether the login state is still valid.
func (r *CheckSessionKeyResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// CheckSessionKeyErrMsg returns the description of a checksession error.
func CheckSessionKeyErrMsg(code int) string {
	if msg, ok := CheckSessionKeyErrCodeMap[code]; ok {
		return msg
	}
	return CheckSessionKeyDefaultErrMsg
}

// CheckSessionKeyError turns the errcode from the reply into an error.
func CheckSessionKeyError(r *CheckSessionKeyResponse) error {
	return errors.New(CheckSessionKeyErrMsg(r.ErrCode))
}

// CheckSessionSignature computes the signature /wxa/checksession expects: an
// HMAC-SHA256 keyed by the session key over an empty message, hex-encoded.
//
// An application that keeps the session_key server side (session.Manager does)
// can call this instead of trusting a signature sent by the client.
func CheckSessionSignature(sessionKey string) string {
	return security.HMACSHA256Hex(nil, []byte(sessionKey))
}

// CheckSessionKey calls /wxa/checksession for openID.
//
// A successful call is one where the reply reports errcode 0; use
// CheckSessionKeyError(result) to turn any other errcode into an error. signature
// is the value produced by CheckSessionSignature.
func (a *Application) CheckSessionKey(
	ctx context.Context,
	openID string,
	signature string,
) (*CheckSessionKeyResponse, error) {
	if openID == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: openid must not be empty")
	}
	if signature == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: signature must not be empty")
	}

	resp, err := a.api.Get(ctx, checkSessionKeyPath, url.Values{
		"openid":     {openID},
		"signature":  {signature},
		"sig_method": {checkSessionKeySigMethod},
	})
	if err != nil {
		return nil, err
	}

	var result CheckSessionKeyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
