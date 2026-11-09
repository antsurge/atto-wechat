package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// testUpdateOrderPath fakes a tracking event, for testing GetPath without a real
// parcel.
const (
	testUpdateOrderPath          = "/cgi-bin/express/business/test_update_order"
	TestUpdateOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// TestUpdateOrderErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/test_update_order, keyed by the errcode as it appears
// in the reply.
var TestUpdateOrderErrCodeMap = logisticsErrCodes()

// The only account and courier the endpoint accepts.
const (
	// TestUpdateOrderBizID is the biz_id the endpoint requires.
	TestUpdateOrderBizID = "test_biz_id"
	// TestUpdateOrderDeliveryID is the delivery_id the endpoint requires.
	TestUpdateOrderDeliveryID = "TEST"
)

// TestUpdateOrderParams is the tracking event to inject.
type TestUpdateOrderParams struct {
	// OrderID is the merchant order id.
	OrderID string `json:"order_id"`
	// OpenID is the buyer.
	OpenID string `json:"openid,omitempty"`
	// BizID must be test_biz_id: the endpoint only works against the test account.
	BizID string `json:"biz_id"`
	// DeliveryID must be TEST: the endpoint only works against the test courier.
	DeliveryID string `json:"delivery_id"`
	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
	// ActionTime is when the event happened, as a Unix timestamp in seconds.
	ActionTime int64 `json:"action_time"`
	// ActionType is the courier's event type, e.g. 100001 for 已揽件.
	ActionType int `json:"action_type"`
	// ActionMsg is the event description.
	ActionMsg string `json:"action_msg"`
}

// CheckTestUpdateOrderParams validates params before anything is sent to WeChat.
func CheckTestUpdateOrderParams(params TestUpdateOrderParams) error {
	// The endpoint only works against the test account and courier. Rejecting
	// anything else keeps a mistaken call from injecting a fake event into a real
	// parcel, which the buyer would see.
	if params.BizID != TestUpdateOrderBizID {
		return invalidLogisticsField("biz_id must be " + TestUpdateOrderBizID)
	}
	if params.DeliveryID != TestUpdateOrderDeliveryID {
		return invalidLogisticsField("delivery_id must be " + TestUpdateOrderDeliveryID)
	}
	if params.OrderID == "" {
		return invalidLogisticsField("order_id is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}

	return nil
}

// TestUpdateOrderResponse is the reply of
// /cgi-bin/express/business/test_update_order.
type TestUpdateOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the fake event.
func (r *TestUpdateOrderResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// TestUpdateOrderErrMsg returns the description of a test update order error.
func TestUpdateOrderErrMsg(code int) string {
	return logisticsErrMsg(TestUpdateOrderErrCodeMap, code, TestUpdateOrderDefaultErrMsg)
}

// TestUpdateOrderError turns the errcode from the reply into an error.
func TestUpdateOrderError(r *TestUpdateOrderResponse) error {
	return errors.New(TestUpdateOrderErrMsg(r.ErrCode))
}

// TestUpdateOrder injects a fake tracking event into a waybill.
//
// It only exists to test the notification and GetPath flows before a courier
// goes live; the injected event is indistinguishable from a real one to the
// buyer, so never call it in production.
func (a *Application) TestUpdateOrder(
	ctx context.Context,
	params TestUpdateOrderParams,
) (*TestUpdateOrderResponse, error) {
	if err := CheckTestUpdateOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, testUpdateOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result TestUpdateOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
