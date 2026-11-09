package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerUpdateOrderFeePath reports the freight the buyer has to pay.
const (
	providerUpdateOrderFeePath          = "/cgi-bin/express/delivery/single_waybill/fee"
	ProviderUpdateOrderFeeDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderUpdateOrderFeeErrCodeMap holds the errcode descriptions WeChat
// documents for /cgi-bin/express/delivery/single_waybill/fee, keyed by the
// errcode as it appears in the reply.
var ProviderUpdateOrderFeeErrCodeMap = logisticsErrCodes()

// ProviderNeedPay values carried in ProviderUpdateOrderFeeParams.NeedPay.
const (
	// ProviderNeedPayNo means the merchant already paid.
	ProviderNeedPayNo = 0
	// ProviderNeedPayWechat means the buyer pays with WeChat Pay.
	ProviderNeedPayWechat = 1
	// ProviderNeedPayScore means the buyer pays with WeChat Pay Score.
	ProviderNeedPayScore = 2
)

// ProviderUpdateOrderFeeParams is the freight breakdown to report.
//
// Every amount is in fen, not yuan.
type ProviderUpdateOrderFeeParams struct {
	// Token is the Token WeChat pushed with the order.
	Token string `json:"token"`
	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
	// NeedPay is one of the ProviderNeedPay* values.
	NeedPay int `json:"need_pay"`
	// Fee is what the buyer actually pays, usually equal to OriginalFee.
	Fee int64 `json:"fee"`
	// OriginalFee is BaseFee + InsuredFee + OtherFee.
	OriginalFee int64 `json:"original_fee"`
	// BaseFee is the freight.
	BaseFee int64 `json:"base_fee"`
	// InsuredFee is the insurance fee.
	InsuredFee int64 `json:"insured_fee,omitempty"`
	// OtherFee is any other charge.
	OtherFee int64 `json:"other_fee,omitempty"`
	// Remark describes OtherFee. WeChat documents the field as a number, but it
	// reads as free text on the payment page, so it is a string here.
	Remark string `json:"remark,omitempty"`
	// PayGoodsName is the product name shown on the WeChat Pay page.
	PayGoodsName string `json:"pay_goods_name,omitempty"`
}

// CheckProviderUpdateOrderFeeParams validates params before anything is sent to
// WeChat.
func CheckProviderUpdateOrderFeeParams(params ProviderUpdateOrderFeeParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}

	switch params.NeedPay {
	case ProviderNeedPayNo, ProviderNeedPayWechat, ProviderNeedPayScore:
	default:
		return invalidLogisticsField("need_pay must be 0, 1 or 2")
	}

	if params.Fee < 0 {
		return invalidLogisticsField("fee must not be negative")
	}
	if params.OriginalFee < 0 {
		return invalidLogisticsField("original_fee must not be negative")
	}
	if params.BaseFee < 0 {
		return invalidLogisticsField("base_fee must not be negative")
	}
	// WeChat shows the breakdown to the buyer, so a total that does not add up
	// would be visible as a wrong price.
	if params.OriginalFee != params.BaseFee+params.InsuredFee+params.OtherFee {
		return invalidLogisticsField("original_fee must equal base_fee + insured_fee + other_fee")
	}
	if params.Fee > params.OriginalFee {
		return invalidLogisticsField("fee must not exceed original_fee")
	}

	return nil
}

// ProviderUpdateOrderFeeResponse is the reply of
// /cgi-bin/express/delivery/single_waybill/fee.
type ProviderUpdateOrderFeeResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the freight.
func (r *ProviderUpdateOrderFeeResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderUpdateOrderFeeErrMsg returns the description of an update order fee
// error.
func ProviderUpdateOrderFeeErrMsg(code int) string {
	return logisticsErrMsg(ProviderUpdateOrderFeeErrCodeMap, code, ProviderUpdateOrderFeeDefaultErrMsg)
}

// ProviderUpdateOrderFeeError turns the errcode from the reply into an error.
func ProviderUpdateOrderFeeError(r *ProviderUpdateOrderFeeResponse) error {
	return errors.New(ProviderUpdateOrderFeeErrMsg(r.ErrCode))
}

// ProviderUpdateOrderFee reports the freight the buyer has to pay.
//
// Report it once the parcel is weighed, before it is picked up: WeChat asks the
// buyer to pay with this number, so a later change means a second payment flow.
func (a *Application) ProviderUpdateOrderFee(
	ctx context.Context,
	params ProviderUpdateOrderFeeParams,
) (*ProviderUpdateOrderFeeResponse, error) {
	if err := CheckProviderUpdateOrderFeeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerUpdateOrderFeePath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderUpdateOrderFeeResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
