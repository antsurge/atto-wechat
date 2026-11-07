package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPluginOpenPIDPath is the plugin openpid endpoint: it exchanges the
// plugin login code for the unique identifier of the plugin's user.
const (
	getPluginOpenPIDPath          = "/wxa/getpluginopenpid"
	GetPluginOpenPIDDefaultErrMsg = "系统错误，请稍后再试"
)

// GetPluginOpenPIDErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/getpluginopenpid, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GetPluginOpenPIDErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40029: "code 无效",
	45011: "API 调用太频繁，请稍候再试",
	1000:  "系统错误",
	1001:  "请求参数非法",
	1003:  "请求频率过快",
	1005:  "插件 appid 与数据不匹配",
	1007:  "openpid 数据不存在",
	1022:  "json 数据解析错误",
}

// GetPluginOpenPIDResponse is the reply of /wxa/getpluginopenpid.
//
// The structure keeps the two kinds of parameters apart: the embedded common
// header is what every WeChat API returns (errcode/errmsg), while the field
// below is specific to this endpoint.
type GetPluginOpenPIDResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OpenPID string `json:"openpid"`
}

// OK reports whether the response contains the plugin user identifier.
func (r *GetPluginOpenPIDResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.OpenPID != ""
}

// GetPluginOpenPIDErrMsg returns the description of a getPluginOpenPId error.
func GetPluginOpenPIDErrMsg(code int) string {
	if msg, ok := GetPluginOpenPIDErrCodeMap[code]; ok {
		return msg
	}
	return GetPluginOpenPIDDefaultErrMsg
}

// GetPluginOpenPIDError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetPluginOpenPIDErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetPluginOpenPIDError(r *GetPluginOpenPIDResponse) error {
	return errors.New(GetPluginOpenPIDErrMsg(r.ErrCode))
}

// GetPluginOpenPID exchanges the code returned by wx.pluginLogin for the unique
// identifier of the plugin's user.
//
// The code comes from wx.pluginLogin inside the plugin, not from wx.login; it
// is valid for five minutes and can only be used once. openpid identifies the
// user within the plugin, so it is not interchangeable with openid.
func (a *Application) GetPluginOpenPID(
	ctx context.Context,
	code string,
) (*GetPluginOpenPIDResponse, error) {
	if code == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: code must not be empty")
	}

	resp, err := a.api.PostJSON(ctx, getPluginOpenPIDPath, nil, map[string]string{
		"code": code,
	})
	if err != nil {
		return nil, err
	}

	var result GetPluginOpenPIDResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
