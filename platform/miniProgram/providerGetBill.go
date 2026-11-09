package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerGetBillPath downloads the daily bill of waybills.
const (
	providerGetBillPath          = "/cgi-bin/express/delivery/single_waybill/get_bill"
	ProviderGetBillDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderGetBillErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/delivery/single_waybill/get_bill, keyed by the errcode as it
// appears in the reply.
var ProviderGetBillErrCodeMap = logisticsErrCodes()

// ProviderGetBillParams selects the bill to download.
type ProviderGetBillParams struct {
	// Date is the day to bill, formatted as YYYYMMDD.
	Date string `json:"date"`
	// Type is one of the ProviderBillType* values.
	Type string `json:"type"`
}

// CheckProviderGetBillParams validates params before anything is sent to WeChat.
func CheckProviderGetBillParams(params ProviderGetBillParams) error {
	if len(params.Date) != 8 {
		return invalidLogisticsField("date must be formatted as YYYYMMDD")
	}
	for _, r := range params.Date {
		if r < '0' || r > '9' {
			return invalidLogisticsField("date must be formatted as YYYYMMDD")
		}
	}

	switch params.Type {
	case ProviderBillTypeAll, ProviderBillTypeSuccess, ProviderBillTypeRefund:
	default:
		return invalidLogisticsField("type must be ALL, SUCCESS or REFUND")
	}

	return nil
}

// ProviderGetBillResponse is the reply of
// /cgi-bin/express/delivery/single_waybill/get_bill.
//
// The endpoint answers with the bill file itself on success and with JSON only on
// failure, so the body cannot be decoded as JSON unconditionally.
type ProviderGetBillResponse struct {
	// Common reply parameters, set only when the call failed.
	response.WechatResponse

	// Content is the bill file on success, empty on failure.
	Content []byte
	// ContentType is the Content-Type of the reply, e.g. text/csv.
	ContentType string
}

// OK reports whether WeChat returned a bill file.
func (r *ProviderGetBillResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.Content) > 0
}

// ProviderGetBillErrMsg returns the description of a get bill error.
func ProviderGetBillErrMsg(code int) string {
	return logisticsErrMsg(ProviderGetBillErrCodeMap, code, ProviderGetBillDefaultErrMsg)
}

// ProviderGetBillError returns nil when a bill file came back and the errcode
// description when WeChat answered with JSON instead.
//
// It never returns an error for a successful call because the reply is a file,
// not an errcode.
func ProviderGetBillError(r *ProviderGetBillResponse) error {
	if r == nil {
		return errors.New(ProviderGetBillDefaultErrMsg)
	}
	if r.ErrCode == 0 {
		return nil
	}

	return errors.New(ProviderGetBillErrMsg(r.ErrCode))
}

// ProviderGetBill downloads the daily bill of waybills.
//
// WeChat prepares the bill once per day, so today's bill is usually empty; ask
// for yesterday and earlier. The content is the file itself, worth storing as is
// rather than parsing in memory.
func (a *Application) ProviderGetBill(
	ctx context.Context,
	params ProviderGetBillParams,
) (*ProviderGetBillResponse, error) {
	if err := CheckProviderGetBillParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerGetBillPath, nil, params)
	if err != nil {
		return nil, err
	}

	result := ProviderGetBillResponse{ContentType: resp.Header.Get("Content-Type")}
	if isJSONBody(resp.Body) {
		if err := resp.DecodeJSON(&result); err != nil {
			return nil, err
		}

		return &result, nil
	}

	result.Content = resp.Body

	return &result, nil
}

// isJSONBody reports whether body starts with a JSON object.
//
// WeChat answers this endpoint with either a CSV file or {"errcode":...}, and
// only the latter is JSON.
func isJSONBody(body []byte) bool {
	for _, b := range body {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}

	return false
}
