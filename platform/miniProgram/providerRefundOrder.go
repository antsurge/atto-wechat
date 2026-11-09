package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerRefundOrderPath refunds freight the courier overcharged.
const (
	providerRefundOrderPath          = "/cgi-bin/express/delivery/single_waybill/refund_order"
	ProviderRefundOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderRefundOrderErrCodeMap holds the errcode descriptions WeChat documents
// for /cgi-bin/express/delivery/single_waybill/refund_order, keyed by the errcode
// as it appears in the reply.
var ProviderRefundOrderErrCodeMap = logisticsErrCodes()

// ProviderRefundOrderParams is the refund to issue.
type ProviderRefundOrderParams struct {
	// Token is the Token WeChat pushed with the order.
	Token string `json:"token"`
	// Fee is the amount to refund, in fen.
	Fee int64 `json:"fee"`
}

// CheckProviderRefundOrderParams validates params before anything is sent to
// WeChat.
func CheckProviderRefundOrderParams(params ProviderRefundOrderParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
	}
	if params.Fee <= 0 {
		return invalidLogisticsField("fee must be greater than 0")
	}

	return nil
}

// ProviderRefundOrderResponse is the reply of
// /cgi-bin/express/delivery/single_waybill/refund_order.
type ProviderRefundOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the refund.
func (r *ProviderRefundOrderResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderRefundOrderErrMsg returns the description of a refund order error.
func ProviderRefundOrderErrMsg(code int) string {
	return logisticsErrMsg(ProviderRefundOrderErrCodeMap, code, ProviderRefundOrderDefaultErrMsg)
}

// ProviderRefundOrderError turns the errcode from the reply into an error.
func ProviderRefundOrderError(r *ProviderRefundOrderResponse) error {
	return errors.New(ProviderRefundOrderErrMsg(r.ErrCode))
}

// ProviderRefundOrder refunds freight the courier overcharged.
//
// The money goes back the way it came, so a refund can only follow a payment:
// an order the merchant paid directly has nothing to refund here.
func (a *Application) ProviderRefundOrder(
	ctx context.Context,
	params ProviderRefundOrderParams,
) (*ProviderRefundOrderResponse, error) {
	if err := CheckProviderRefundOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerRefundOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderRefundOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
