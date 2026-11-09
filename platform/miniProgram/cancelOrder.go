package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// cancelOrderPath cancels a waybill.
const (
	cancelOrderPath          = "/cgi-bin/express/business/order/cancel"
	CancelOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// CancelOrderErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/order/cancel, keyed by the errcode as it appears in
// the reply.
var CancelOrderErrCodeMap = logisticsErrCodes()

// CancelOrderParams identifies the waybill to cancel.
type CancelOrderParams struct {
	// OrderID is the merchant order id.
	OrderID string `json:"order_id"`
	// OpenID is the buyer; it is needed when the waybill was created from the mini
	// program.
	OpenID string `json:"openid,omitempty"`
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// WaybillID is the waybill number returned by AddOrder.
	WaybillID string `json:"waybill_id"`
}

// CheckCancelOrderParams validates params before anything is sent to WeChat.
func CheckCancelOrderParams(params CancelOrderParams) error {
	if params.OrderID == "" {
		return invalidLogisticsField("order_id is required")
	}
	if params.DeliveryID == "" {
		return invalidLogisticsField("delivery_id is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}

	return nil
}

// CancelOrderResponse is the reply of /cgi-bin/express/business/order/cancel.
type CancelOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// DeliveryResultCode is the courier's own result code, 0 on success.
	DeliveryResultCode int `json:"delivery_resultcode"`
	// DeliveryResultMsg explains a non-zero DeliveryResultCode.
	DeliveryResultMsg string `json:"delivery_resultmsg"`
}

// OK reports whether the courier actually cancelled the waybill.
func (r *CancelOrderResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.DeliveryResultCode == 0
}

// CourierError returns the courier's rejection as an error, or nil when the
// courier accepted the cancellation.
func (r *CancelOrderResponse) CourierError() error {
	if r == nil || r.DeliveryResultCode == 0 {
		return nil
	}

	msg := r.DeliveryResultMsg
	if msg == "" {
		msg = "运力方拒绝取消运单"
	}

	return errors.New(msg)
}

// CancelOrderErrMsg returns the description of a cancel order error.
func CancelOrderErrMsg(code int) string {
	return logisticsErrMsg(CancelOrderErrCodeMap, code, CancelOrderDefaultErrMsg)
}

// CancelOrderError returns nil when the courier cancelled the waybill, the
// courier's message when it refused, and the errcode description otherwise.
//
// It covers both failure layers so a caller needs one call instead of checking
// ErrCode and CourierError itself.
func CancelOrderError(r *CancelOrderResponse) error {
	if r == nil {
		return errors.New(CancelOrderDefaultErrMsg)
	}
	if r.ErrCode != 0 {
		return errors.New(CancelOrderErrMsg(r.ErrCode))
	}

	return r.CourierError()
}

// CancelOrder cancels a waybill.
//
// A courier may refuse to cancel a parcel that was already picked up, which
// arrives as a non-zero DeliveryResultCode with errcode 0, so check OK().
func (a *Application) CancelOrder(
	ctx context.Context,
	params CancelOrderParams,
) (*CancelOrderResponse, error) {
	if err := CheckCancelOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, cancelOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result CancelOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
