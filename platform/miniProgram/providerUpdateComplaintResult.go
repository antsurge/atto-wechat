package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerUpdateComplaintResultPath reports how a complaint was resolved.
const (
	providerUpdateComplaintResultPath          = "/cgi-bin/express/delivery/scatter/update_complaint_result"
	ProviderUpdateComplaintResultDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderUpdateComplaintResultErrCodeMap holds the errcode descriptions WeChat
// documents for /cgi-bin/express/delivery/scatter/update_complaint_result, keyed
// by the errcode as it appears in the reply.
var ProviderUpdateComplaintResultErrCodeMap = logisticsErrCodes()

// ProviderUpdateComplaintResultParams is the complaint resolution to report.
type ProviderUpdateComplaintResultParams struct {
	// Token is the Token WeChat pushed with the order.
	Token string `json:"token"`
	// WaybillID is the waybill number the complaint is about.
	WaybillID string `json:"waybill_id"`
	// Result is the resolution, e.g. 已赔付 or 已解释.
	Result string `json:"result"`
	// Desc explains the resolution to the buyer.
	Desc string `json:"desc"`
}

// CheckProviderUpdateComplaintResultParams validates params before anything is
// sent to WeChat.
func CheckProviderUpdateComplaintResultParams(params ProviderUpdateComplaintResultParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}
	if params.Result == "" {
		return invalidLogisticsField("result is required")
	}
	if params.Desc == "" {
		return invalidLogisticsField("desc is required")
	}

	return nil
}

// ProviderUpdateComplaintResultResponse is the reply of
// /cgi-bin/express/delivery/scatter/update_complaint_result.
type ProviderUpdateComplaintResultResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat recorded the resolution.
func (r *ProviderUpdateComplaintResultResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ProviderUpdateComplaintResultErrMsg returns the description of an update
// complaint result error.
func ProviderUpdateComplaintResultErrMsg(code int) string {
	return logisticsErrMsg(
		ProviderUpdateComplaintResultErrCodeMap,
		code,
		ProviderUpdateComplaintResultDefaultErrMsg,
	)
}

// ProviderUpdateComplaintResultError turns the errcode from the reply into an
// error.
func ProviderUpdateComplaintResultError(r *ProviderUpdateComplaintResultResponse) error {
	return errors.New(ProviderUpdateComplaintResultErrMsg(r.ErrCode))
}

// ProviderUpdateComplaintResult reports how a complaint was resolved.
//
// The buyer opened the complaint in the waybill page; this closes the loop.
// Desc is shown to the buyer, so write it for them rather than for the courier's
// internal notes.
func (a *Application) ProviderUpdateComplaintResult(
	ctx context.Context,
	params ProviderUpdateComplaintResultParams,
) (*ProviderUpdateComplaintResultResponse, error) {
	if err := CheckProviderUpdateComplaintResultParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerUpdateComplaintResultPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderUpdateComplaintResultResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
