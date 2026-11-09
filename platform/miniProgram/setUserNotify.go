package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setUserNotifyPath activates or updates a service card.
const (
	setUserNotifyPath          = "/wxa/set_user_notify"
	SetUserNotifyDefaultErrMsg = "系统错误，请稍后再试"
)

// SetUserNotifyErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/set_user_notify, keyed by the errcode as it appears in the reply.
//
// The 854xx range is specific to service cards: 85470/85471 mean the caller
// built content_json wrongly, and 85444-85447 mean a nested structure inside it
// failed validation.
var SetUserNotifyErrCodeMap = map[int]string{
	40003: "openid 不正确",
	85431: "系统错误，请稍后再试",
	85434: "notify_type 不合法",
	85436: "notify_code 异常（被封禁）",
	85437: "notify_code 不存在",
	85444: "pay_info 校验不通过",
	85445: "store_info 校验不通过",
	85446: "shipping_list 校验不通过",
	85447: "product_list 校验不通过",
	85448: "服务卡片消息已过期",
	85449: "服务卡片消息已失效",
	85461: "notify_type 准入驳回",
	85462: "小程序准入驳回",
	85470: "content_json 不合法",
	85471: "content_json 包含非 utf8 字符",
}

// SetUserNotifyParams describes the card to activate or update.
type SetUserNotifyParams struct {
	// OpenID is the user the card belongs to.
	OpenID string

	// NotifyType is the card id, such as 1003 for an order card.
	NotifyType int

	// NotifyCode is the token that authorises the update. When it comes from a
	// WeChat Pay transaction id, WeChat may not have indexed the payment yet:
	// an 85437 right after paying is worth retrying a minute later.
	NotifyCode string

	// ContentJSON is the card content, serialised as a JSON string. Its shape
	// is defined per NotifyType, which is why it stays a string here.
	ContentJSON string

	// CheckJSON is the optional verification payload, serialised as a JSON
	// string the same way.
	CheckJSON string
}

// CheckSetUserNotifyParams validates params before anything is sent to WeChat.
func CheckSetUserNotifyParams(params SetUserNotifyParams) error {
	if params.OpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: openid is required",
		)
	}

	if params.NotifyType == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: notify_type is required",
		)
	}

	if params.NotifyCode == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: notify_code is required",
		)
	}

	if params.ContentJSON == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: content_json is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SetUserNotifyParams) body() *setUserNotifyBody {
	return &setUserNotifyBody{
		OpenID:      p.OpenID,
		NotifyType:  p.NotifyType,
		NotifyCode:  p.NotifyCode,
		ContentJSON: p.ContentJSON,
		CheckJSON:   p.CheckJSON,
	}
}

// setUserNotifyBody is the JSON body of /wxa/set_user_notify.
type setUserNotifyBody struct {
	OpenID      string `json:"openid"`
	NotifyType  int    `json:"notify_type"`
	NotifyCode  string `json:"notify_code"`
	ContentJSON string `json:"content_json"`
	CheckJSON   string `json:"check_json,omitempty"`
}

// SetUserNotifyResponse is the reply of /wxa/set_user_notify.
//
// It carries no business payload: success is reported by errcode alone.
type SetUserNotifyResponse struct {
	response.WechatResponse
}

// SetUserNotifyErrMsg returns the description of a setUserNotify error.
func SetUserNotifyErrMsg(code int) string {
	if msg, ok := SetUserNotifyErrCodeMap[code]; ok {
		return msg
	}
	return SetUserNotifyDefaultErrMsg
}

// SetUserNotifyError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SetUserNotifyErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func SetUserNotifyError(r *SetUserNotifyResponse) error {
	return errors.New(SetUserNotifyErrMsg(r.ErrCode))
}

// SetUserNotify activates or updates a service card.
//
// A service card lives inside the WeChat service notification panel rather than
// in a chat: the user opens it long after the message that announced it, so its
// state has to be refreshed. Both the first activation and every later update
// go through this endpoint, and the card keeps its latest ContentJSON.
//
// The endpoint is restricted to the e-commerce categories, and 85461/85462 mean
// the account or the card type was not admitted.
func (a *Application) SetUserNotify(
	ctx context.Context,
	params SetUserNotifyParams,
) (*SetUserNotifyResponse, error) {
	if err := CheckSetUserNotifyParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, setUserNotifyPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SetUserNotifyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
