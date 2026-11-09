package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerUpdateBusinessPath reports the review result of a merchant account.
const (
	providerUpdateBusinessPath          = "/cgi-bin/express/delivery/service/business/update"
	ProviderUpdateBusinessDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderUpdateBusinessErrCodeMap holds the errcode descriptions WeChat
// documents for /cgi-bin/express/delivery/service/business/update, keyed by the
// errcode as it appears in the reply.
var ProviderUpdateBusinessErrCodeMap = logisticsErrCodes()

// ProviderUpdateBusinessParams is the review result to report.
type ProviderUpdateBusinessParams struct {
	// ShopAppID is the merchant mini program appid, the ShopAppID of the review
	// event.
	ShopAppID string `json:"shop_app_id"`
	// BizID is the merchant account at the courier.
	BizID string `json:"biz_id"`
	// ResultCode is 0 when the account was approved, anything else when it was
	// rejected.
	ResultCode int `json:"result_code"`
	// ResultMsg is the rejection reason; only read when ResultCode is not 0.
	ResultMsg string `json:"result_msg,omitempty"`
}

// CheckProviderUpdateBusinessParams validates params before anything is sent to
// WeChat.
func CheckProviderUpdateBusinessParams(params ProviderUpdateBusinessParams) error {
	if params.ShopAppID == "" {
		return invalidLogisticsField("shop_app_id is required")
	}
	if params.BizID == "" {
		return invalidLogisticsField("biz_id is required")
	}
	// A rejection without a reason leaves the merchant with nothing to fix.
	if params.ResultCode != 0 && params.ResultMsg == "" {
		return invalidLogisticsField("result_msg is required when result_code is not 0")
	}

	return nil
}

// ProviderUpdateBusinessResponse is the reply of
// /cgi-bin/express/delivery/service/business/update.
type ProviderUpdateBusinessResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat recorded the review result.
func (r *ProviderUpdateBusinessResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderUpdateBusinessErrMsg returns the description of an update business
// error.
func ProviderUpdateBusinessErrMsg(code int) string {
	return logisticsErrMsg(ProviderUpdateBusinessErrCodeMap, code, ProviderUpdateBusinessDefaultErrMsg)
}

// ProviderUpdateBusinessError turns the errcode from the reply into an error.
func ProviderUpdateBusinessError(r *ProviderUpdateBusinessResponse) error {
	return errors.New(ProviderUpdateBusinessErrMsg(r.ErrCode))
}

// ProviderUpdateBusiness reports the review result of a merchant account.
//
// Only the merchant mini program that requested the review gets the review
// event, so this is the courier's answer to that request. Until it reports 0,
// the merchant cannot create waybills with this account.
func (a *Application) ProviderUpdateBusiness(
	ctx context.Context,
	params ProviderUpdateBusinessParams,
) (*ProviderUpdateBusinessResponse, error) {
	if err := CheckProviderUpdateBusinessParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerUpdateBusinessPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderUpdateBusinessResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
