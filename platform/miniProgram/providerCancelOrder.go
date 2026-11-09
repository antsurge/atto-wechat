package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerCancelOrderPath cancels an order the courier cannot fulfil.
const (
	providerCancelOrderPath          = "/cgi-bin/express/delivery/single_waybill/cancel_order"
	ProviderCancelOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderCancelOrderErrCodeMap holds the errcode descriptions WeChat documents
// for /cgi-bin/express/delivery/single_waybill/cancel_order, keyed by the errcode
// as it appears in the reply.
var ProviderCancelOrderErrCodeMap = logisticsErrCodes()

// ProviderCancelOrderParams is the cancellation to report.
type ProviderCancelOrderParams struct {
	// Token is the Token WeChat pushed with the order.
	Token string `json:"token"`
	// Reason explains the cancellation to the merchant.
	Reason string `json:"reason"`
}

// CheckProviderCancelOrderParams validates params before anything is sent to
// WeChat.
func CheckProviderCancelOrderParams(params ProviderCancelOrderParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
	}
	if params.Reason == "" {
		return invalidLogisticsField("reason is required")
	}

	return nil
}

// ProviderCancelOrderResponse is the reply of
// /cgi-bin/express/delivery/single_waybill/cancel_order.
type ProviderCancelOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the cancellation.
func (r *ProviderCancelOrderResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderCancelOrderErrMsg returns the description of a provider cancel order
// error.
func ProviderCancelOrderErrMsg(code int) string {
	return logisticsErrMsg(ProviderCancelOrderErrCodeMap, code, ProviderCancelOrderDefaultErrMsg)
}

// ProviderCancelOrderError turns the errcode from the reply into an error.
func ProviderCancelOrderError(r *ProviderCancelOrderResponse) error {
	return errors.New(ProviderCancelOrderErrMsg(r.ErrCode))
}

// ProviderCancelOrder cancels an order the courier cannot fulfil.
//
// This is not the merchant's CancelOrder: the courier cancels because it cannot
// pick the parcel up (address out of range, forbidden goods), the merchant
// cancels because it no longer wants the shipment.
func (a *Application) ProviderCancelOrder(
	ctx context.Context,
	params ProviderCancelOrderParams,
) (*ProviderCancelOrderResponse, error) {
	if err := CheckProviderCancelOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerCancelOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderCancelOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
