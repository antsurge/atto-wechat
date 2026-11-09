package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerUpdatePathPath appends a tracking event to a waybill.
const (
	providerUpdatePathPath          = "/cgi-bin/express/delivery/path/update"
	ProviderUpdatePathDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderUpdatePathErrCodeMap holds the errcode descriptions WeChat documents
// for /cgi-bin/express/delivery/path/update, keyed by the errcode as it appears
// in the reply.
var ProviderUpdatePathErrCodeMap = logisticsErrCodes()

// ProviderUpdatePathParams is the tracking event to append.
type ProviderUpdatePathParams struct {
	// Token is the Token WeChat pushed with the order; it replaces order_id on the
	// provider side.
	Token string `json:"token"`
	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
	// ActionTime is when the event happened, as a Unix timestamp in seconds.
	ActionTime int64 `json:"action_time"`
	// ActionType is the event type, one of the ProviderAction* values.
	ActionType int `json:"action_type"`
	// ActionMsg is the event description shown to the buyer. Put a mobile number
	// on its own when the courier exposes the courier's phone, and keep it UTF-8.
	ActionMsg string `json:"action_msg"`
}

// CheckProviderUpdatePathParams validates params before anything is sent to
// WeChat.
func CheckProviderUpdatePathParams(params ProviderUpdatePathParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}
	if params.ActionTime <= 0 {
		return invalidLogisticsField("action_time must be a positive Unix timestamp")
	}
	if params.ActionType <= 0 {
		return invalidLogisticsField("action_type is required")
	}
	if params.ActionMsg == "" {
		return invalidLogisticsField("action_msg is required")
	}

	return nil
}

// ProviderUpdatePathResponse is the reply of
// /cgi-bin/express/delivery/path/update.
type ProviderUpdatePathResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat stored the event.
func (r *ProviderUpdatePathResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderUpdatePathErrMsg returns the description of an update path error.
func ProviderUpdatePathErrMsg(code int) string {
	return logisticsErrMsg(ProviderUpdatePathErrCodeMap, code, ProviderUpdatePathDefaultErrMsg)
}

// ProviderUpdatePathError turns the errcode from the reply into an error.
func ProviderUpdatePathError(r *ProviderUpdatePathResponse) error {
	return errors.New(ProviderUpdatePathErrMsg(r.ErrCode))
}

// ProviderUpdatePath appends a tracking event to a waybill.
//
// Events are appended in call order, not sorted by ActionTime, so report them as
// they happen. The buyer sees ActionMsg verbatim in the tracking page.
func (a *Application) ProviderUpdatePath(
	ctx context.Context,
	params ProviderUpdatePathParams,
) (*ProviderUpdatePathResponse, error) {
	if err := CheckProviderUpdatePathParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerUpdatePathPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderUpdatePathResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
