package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerUpdateOrderStatusPath drives the order through its lifecycle, including
// the stages before pickup.
const (
	providerUpdateOrderStatusPath          = "/cgi-bin/express/delivery/single_waybill/update"
	ProviderUpdateOrderStatusDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderUpdateOrderStatusErrCodeMap holds the errcode descriptions WeChat
// documents for /cgi-bin/express/delivery/single_waybill/update, keyed by the
// errcode as it appears in the reply.
var ProviderUpdateOrderStatusErrCodeMap = logisticsErrCodes()

// ProviderUpdateOrderStatusParams is the status change to report.
type ProviderUpdateOrderStatusParams struct {
	// Token is the Token WeChat pushed with the order.
	Token string `json:"token"`
	// WaybillID is the waybill number; optional because the order may not have one
	// yet at the pre-pickup stages.
	WaybillID string `json:"waybill_id,omitempty"`
	// ActionTime is when the change happened, as a Unix timestamp in seconds.
	ActionTime int64 `json:"action_time"`
	// ActionType is one of the ProviderAction* values.
	ActionType int `json:"action_type"`
	// ActionMsg is the description shown in the tracking page. If it contains a
	// mobile number, write it as 11 digits, and keep it UTF-8.
	ActionMsg string `json:"action_msg"`
	// PickupCourierName is the courier who collects the parcel.
	PickupCourierName string `json:"pickup_courier_name,omitempty"`
	// PickupCourierPhone is that courier's phone.
	PickupCourierPhone string `json:"pickup_courier_phone,omitempty"`
	// DeliveryCourierName is the courier who delivers the parcel.
	DeliveryCourierName string `json:"delivery_courier_name,omitempty"`
	// DeliveryCourierPhone is that courier's phone.
	DeliveryCourierPhone string `json:"delivery_courier_phone,omitempty"`
}

// CheckProviderUpdateOrderStatusParams validates params before anything is sent
// to WeChat.
//
// WeChat marks the four courier fields as required, but which of them apply
// depends on the stage: there is no delivery courier before the parcel reaches
// the destination city. Requiring them all here would reject legitimate
// pre-pickup updates, so they are only rejected when half filled.
func CheckProviderUpdateOrderStatusParams(params ProviderUpdateOrderStatusParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
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

	if err := checkCourierPair(
		"pickup_courier_name",
		params.PickupCourierName,
		"pickup_courier_phone",
		params.PickupCourierPhone,
	); err != nil {
		return err
	}

	return checkCourierPair(
		"delivery_courier_name",
		params.DeliveryCourierName,
		"delivery_courier_phone",
		params.DeliveryCourierPhone,
	)
}

// checkCourierPair rejects a courier whose name and phone are not both set or
// both empty.
func checkCourierPair(nameKey, name, phoneKey, phone string) error {
	if (name == "") == (phone == "") {
		return nil
	}

	return invalidLogisticsField(nameKey + " and " + phoneKey + " must be set together")
}

// ProviderUpdateOrderStatusResponse is the reply of
// /cgi-bin/express/delivery/single_waybill/update.
type ProviderUpdateOrderStatusResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the status change.
func (r *ProviderUpdateOrderStatusResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderUpdateOrderStatusErrMsg returns the description of an update order
// status error.
func ProviderUpdateOrderStatusErrMsg(code int) string {
	return logisticsErrMsg(ProviderUpdateOrderStatusErrCodeMap, code, ProviderUpdateOrderStatusDefaultErrMsg)
}

// ProviderUpdateOrderStatusError turns the errcode from the reply into an error.
func ProviderUpdateOrderStatusError(r *ProviderUpdateOrderStatusResponse) error {
	return errors.New(ProviderUpdateOrderStatusErrMsg(r.ErrCode))
}

// ProviderUpdateOrderStatus drives the order through its lifecycle.
//
// This is what tells the merchant and the buyer that the order was accepted,
// assigned, paid or cancelled before pickup — the stages ProviderUpdatePath does
// not cover. Send the states in order: the pre-pickup ones (9xxxx) first, then
// the path events.
func (a *Application) ProviderUpdateOrderStatus(
	ctx context.Context,
	params ProviderUpdateOrderStatusParams,
) (*ProviderUpdateOrderStatusResponse, error) {
	if err := CheckProviderUpdateOrderStatusParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerUpdateOrderStatusPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderUpdateOrderStatusResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
