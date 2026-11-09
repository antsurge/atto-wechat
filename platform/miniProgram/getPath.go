package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getPathPath returns the tracking events of a waybill.
const (
	getPathPath          = "/cgi-bin/express/business/path/get"
	GetPathDefaultErrMsg = "系统错误，请稍后再试"
)

// GetPathErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/path/get, keyed by the errcode as it appears in the
// reply.
var GetPathErrCodeMap = logisticsErrCodes()

// GetPathParams identifies the waybill whose tracking to fetch.
type GetPathParams struct {
	// OpenID is the buyer; it is needed when the waybill was created from the mini
	// program.
	OpenID string `json:"openid,omitempty"`
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
}

// CheckGetPathParams validates params before anything is sent to WeChat.
func CheckGetPathParams(params GetPathParams) error {
	if params.DeliveryID == "" {
		return invalidLogisticsField("delivery_id is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}

	return nil
}

// LogisticsPathItem is one tracking event.
type LogisticsPathItem struct {
	// ActionTime is when the event happened, as a Unix timestamp in seconds.
	ActionTime int64 `json:"action_time"`
	// ActionType is the courier's event type, e.g. 100001 for 已揽件；the values
	// are defined by the courier.
	ActionType int `json:"action_type"`
	// ActionMsg is the event description shown to the buyer.
	ActionMsg string `json:"action_msg"`
}

// GetPathResponse is the reply of /cgi-bin/express/business/path/get.
type GetPathResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OpenID     string `json:"openid"`
	DeliveryID string `json:"delivery_id"`
	WaybillID  string `json:"waybill_id"`
	// PathItemNum is the number of tracking events.
	PathItemNum int `json:"path_item_num"`
	// PathItemList is ordered oldest first.
	PathItemList []LogisticsPathItem `json:"path_item_list"`
}

// OK reports whether WeChat answered with a usable tracking list.
//
// An empty list is valid: the courier may not have scanned the parcel yet, which
// is different from the call failing.
func (r *GetPathResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Latest returns the most recent tracking event, or nil when there is none.
func (r *GetPathResponse) Latest() *LogisticsPathItem {
	if r == nil || len(r.PathItemList) == 0 {
		return nil
	}

	return &r.PathItemList[len(r.PathItemList)-1]
}

// GetPathErrMsg returns the description of a get path error.
func GetPathErrMsg(code int) string {
	return logisticsErrMsg(GetPathErrCodeMap, code, GetPathDefaultErrMsg)
}

// GetPathError turns the errcode from the reply into an error.
func GetPathError(r *GetPathResponse) error {
	return errors.New(GetPathErrMsg(r.ErrCode))
}

// GetPath fetches the tracking events of a waybill.
//
// The reply only covers events the courier reported to WeChat; couriers that do
// not push tracking return an empty list.
func (a *Application) GetPath(
	ctx context.Context,
	params GetPathParams,
) (*GetPathResponse, error) {
	if err := CheckGetPathParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getPathPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result GetPathResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
