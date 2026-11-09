package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setUserNotifyExtPath updates the extended fields of a service card.
const (
	setUserNotifyExtPath          = "/wxa/set_user_notifyext"
	SetUserNotifyExtDefaultErrMsg = "系统错误，请稍后再试"
)

// SetUserNotifyExtErrCodeMap holds the errcode descriptions WeChat documents
// for /wxa/set_user_notifyext, keyed by the errcode as it appears in the reply.
//
// The 854xx range is specific to service cards: 85470/85471 mean the caller
// built ext_json wrongly, and 85444-85447 mean a nested structure inside it
// failed validation.
var SetUserNotifyExtErrCodeMap = map[int]string{
	40003: "openid 不正确",
	85431: "系统错误，请稍后再试",
	85434: "notify_type 不合法",
	85436: "notify_code 异常（被封禁）",
	85437: "notify_code 不存在",
	85444: "pay_info 校验不通过",
	85445: "store_info 校验不通过",
	85446: "shipping_list 校验不通过",
	85447: "product_list 校验不通过",
	85461: "notify_type 准入驳回",
	85462: "小程序准入驳回",
	85470: "ext_json 不合法",
	85471: "ext_json 包含非 utf8 字符",
}

// SetUserNotifyExtParams describes the extended fields to write.
type SetUserNotifyExtParams struct {
	// OpenID is the user the card belongs to.
	OpenID string

	// NotifyType is the card id, such as 1003 for an order card.
	NotifyType int

	// NotifyCode is the token that authorises the update.
	NotifyCode string

	// ExtJSON is the extended content, serialised as a JSON string. Each card
	// type defines its own shape, which is why it stays a string here.
	ExtJSON string
}

// CheckSetUserNotifyExtParams validates params before anything is sent to
// WeChat.
func CheckSetUserNotifyExtParams(params SetUserNotifyExtParams) error {
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

	if params.ExtJSON == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ext_json is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SetUserNotifyExtParams) body() *setUserNotifyExtBody {
	return &setUserNotifyExtBody{
		OpenID:     p.OpenID,
		NotifyType: p.NotifyType,
		NotifyCode: p.NotifyCode,
		ExtJSON:    p.ExtJSON,
	}
}

// setUserNotifyExtBody is the JSON body of /wxa/set_user_notifyext.
type setUserNotifyExtBody struct {
	OpenID     string `json:"openid"`
	NotifyType int    `json:"notify_type"`
	NotifyCode string `json:"notify_code"`
	ExtJSON    string `json:"ext_json"`
}

// SetUserNotifyExtResponse is the reply of /wxa/set_user_notifyext.
//
// It carries no business payload: success is reported by errcode alone.
type SetUserNotifyExtResponse struct {
	response.WechatResponse
}

// SetUserNotifyExtErrMsg returns the description of a setUserNotifyExt error.
func SetUserNotifyExtErrMsg(code int) string {
	if msg, ok := SetUserNotifyExtErrCodeMap[code]; ok {
		return msg
	}
	return SetUserNotifyExtDefaultErrMsg
}

// SetUserNotifyExtError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SetUserNotifyExtErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func SetUserNotifyExtError(r *SetUserNotifyExtResponse) error {
	return errors.New(SetUserNotifyExtErrMsg(r.ErrCode))
}

// SetUserNotifyExt updates the extended fields of a service card.
//
// It writes the part of the card that is too large to resend on every update,
// such as the order lines, the shipping list or the store address. The card has
// to exist already: the first activation goes through SetUserNotify, which owns
// notify_code.
func (a *Application) SetUserNotifyExt(
	ctx context.Context,
	params SetUserNotifyExtParams,
) (*SetUserNotifyExtResponse, error) {
	if err := CheckSetUserNotifyExtParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, setUserNotifyExtPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SetUserNotifyExtResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
