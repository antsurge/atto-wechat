package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getOrderPath returns one waybill.
const (
	getOrderPath          = "/cgi-bin/express/business/order/get"
	GetOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// GetOrderErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/order/get, keyed by the errcode as it appears in the
// reply.
var GetOrderErrCodeMap = logisticsErrCodes()

// GetOrderParams identifies the waybill to fetch.
type GetOrderParams struct {
	// OrderID is the merchant order id.
	OrderID string `json:"order_id"`
	// OpenID is the buyer; it is needed when the waybill was created from the mini
	// program.
	OpenID string `json:"openid,omitempty"`
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// WaybillID is the waybill number; omit it to look the waybill up by order id.
	WaybillID string `json:"waybill_id,omitempty"`
	// PrintType is LogisticsPrintTypeTwoPart (0) or
	// LogisticsPrintTypeOnePart (1). The zero value is the two-part waybill, which
	// is also what an omitted field means.
	PrintType int `json:"print_type"`
}

// CheckGetOrderParams validates params before anything is sent to WeChat.
func CheckGetOrderParams(params GetOrderParams) error {
	if params.OrderID == "" {
		return invalidLogisticsField("order_id is required")
	}
	if params.DeliveryID == "" {
		return invalidLogisticsField("delivery_id is required")
	}

	switch params.PrintType {
	case LogisticsPrintTypeTwoPart, LogisticsPrintTypeOnePart:
	default:
		return invalidLogisticsField("print_type must be 0 or 1")
	}

	return nil
}

// GetOrderResponse is the reply of /cgi-bin/express/business/order/get.
type GetOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	LogisticsOrderData
}

// OK reports whether a waybill was returned.
func (r *GetOrderResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.WaybillID != ""
}

// GetOrderErrMsg returns the description of a get order error.
func GetOrderErrMsg(code int) string {
	return logisticsErrMsg(GetOrderErrCodeMap, code, GetOrderDefaultErrMsg)
}

// GetOrderError turns the errcode from the reply into an error.
func GetOrderError(r *GetOrderResponse) error {
	return errors.New(GetOrderErrMsg(r.ErrCode))
}

// GetOrder fetches one waybill, including the printable markup.
//
// PrintHTML is what the merchant prints; it is empty for couriers that do not
// offer electronic waybills.
func (a *Application) GetOrder(
	ctx context.Context,
	params GetOrderParams,
) (*GetOrderResponse, error) {
	if err := CheckGetOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result GetOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
