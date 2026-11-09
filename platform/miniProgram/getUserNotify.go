package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getUserNotifyPath reads the current state of a service card.
const (
	getUserNotifyPath          = "/wxa/get_user_notify"
	GetUserNotifyDefaultErrMsg = "系统错误，请稍后再试"
)

// GetUserNotifyErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/get_user_notify, keyed by the errcode as it appears in the reply.
var GetUserNotifyErrCodeMap = map[int]string{
	40003: "openid 不正确",
	85431: "系统错误，请稍后再试",
	85434: "notify_type 不合法",
	85436: "notify_code 异常（被封禁）",
	85437: "notify_code 不存在",
	85461: "notify_type 准入驳回",
	85462: "小程序准入驳回",
}

// GetUserNotifyParams identifies the card to read.
type GetUserNotifyParams struct {
	// OpenID is the user the card belongs to.
	OpenID string

	// NotifyCode is the token of the card. Leave it empty to read the user's
	// cards in the order WeChat stores them.
	NotifyCode string
}

// CheckGetUserNotifyParams validates params before anything is sent to WeChat.
func CheckGetUserNotifyParams(params GetUserNotifyParams) error {
	if params.OpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: openid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetUserNotifyParams) body() *getUserNotifyBody {
	return &getUserNotifyBody{
		OpenID:     p.OpenID,
		NotifyCode: p.NotifyCode,
	}
}

// getUserNotifyBody is the JSON body of /wxa/get_user_notify.
type getUserNotifyBody struct {
	OpenID     string `json:"openid"`
	NotifyCode string `json:"notify_code,omitempty"`
}

// UserNotify is the state of one service card.
type UserNotify struct {
	// NotifyType is the card id.
	NotifyType int `json:"notify_type"`

	// NotifyCode is the token of the card.
	NotifyCode string `json:"notify_code"`

	// ContentJSON is the card content last written by SetUserNotify.
	ContentJSON string `json:"content_json"`

	// CheckJSON is the verification payload last written by SetUserNotify.
	CheckJSON string `json:"check_json"`
}

// GetUserNotifyResponse is the reply of /wxa/get_user_notify.
type GetUserNotifyResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Data []UserNotify `json:"data"`
}

// GetUserNotifyErrMsg returns the description of a getUserNotify error.
func GetUserNotifyErrMsg(code int) string {
	if msg, ok := GetUserNotifyErrCodeMap[code]; ok {
		return msg
	}
	return GetUserNotifyDefaultErrMsg
}

// GetUserNotifyError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetUserNotifyErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetUserNotifyError(r *GetUserNotifyResponse) error {
	return errors.New(GetUserNotifyErrMsg(r.ErrCode))
}

// GetUserNotify reads the current state of a service card.
//
// It is the read side of the service card: useful to rebuild a card after a
// deployment, and to tell "the user never opened it" from "the update was
// rejected". WeChat answers 85437 when notify_code is unknown, which is also
// what a card that was never activated looks like.
func (a *Application) GetUserNotify(
	ctx context.Context,
	params GetUserNotifyParams,
) (*GetUserNotifyResponse, error) {
	if err := CheckGetUserNotifyParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getUserNotifyPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetUserNotifyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
