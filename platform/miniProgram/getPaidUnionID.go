package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPaidUnionIDPath is the paid unionid endpoint: it returns the UnionID of a
// user who has just paid in this mini program.
const (
	getPaidUnionIDPath          = "/wxa/getpaidunionid"
	GetPaidUnionIDDefaultErrMsg = "系统错误，请稍后再试"
)

// GetPaidUnionIDErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/getpaidunionid, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GetPaidUnionIDErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40003: "openid 错误",
	45011: "API 调用太频繁，请稍候再试",
	89002: "没有绑定开放平台账号",
	89300: "订单无效",
}

// PaidUnionIDParams identifies the paid user and proves the payment.
//
// The payment can be proven in either of two ways, but not both:
// TransactionID alone, or MchID together with OutTradeNo. The same data the
// payment callback carries is what belongs here; the mini program never sees
// the order identifiers themselves.
type PaidUnionIDParams struct {
	// OpenID is the paid user's openid in this mini program.
	OpenID string

	// TransactionID is the transaction id of the WeChat payment order.
	TransactionID string

	// MchID and OutTradeNo are the merchant id and merchant order number.
	MchID      string
	OutTradeNo string
}

// query validates the parameters and renders them as query values.
func (p PaidUnionIDParams) query() (url.Values, error) {
	if p.OpenID == "" {
		return nil, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: openid must not be empty",
		)
	}

	values := url.Values{"openid": {p.OpenID}}

	switch {
	case p.TransactionID != "":
		values.Set("transaction_id", p.TransactionID)
	case p.MchID != "" && p.OutTradeNo != "":
		values.Set("mch_id", p.MchID)
		values.Set("out_trade_no", p.OutTradeNo)
	default:
		return nil, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: either transaction_id or mch_id with out_trade_no is required",
		)
	}

	return values, nil
}

// GetPaidUnionIDResponse is the reply of /wxa/getpaidunionid.
//
// The structure keeps the two kinds of parameters apart: the embedded common
// header is what every WeChat API returns (errcode/errmsg), while the field
// below is specific to this endpoint.
type GetPaidUnionIDResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	UnionID string `json:"unionid"`
}

// OK reports whether the response carries the user's UnionID.
func (r *GetPaidUnionIDResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.UnionID != ""
}

// GetPaidUnionIDErrMsg returns the description of a getPaidUnionId error.
func GetPaidUnionIDErrMsg(code int) string {
	if msg, ok := GetPaidUnionIDErrCodeMap[code]; ok {
		return msg
	}
	return GetPaidUnionIDDefaultErrMsg
}

// GetPaidUnionIDError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetPaidUnionIDErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetPaidUnionIDError(r *GetPaidUnionIDResponse) error {
	return errors.New(GetPaidUnionIDErrMsg(r.ErrCode))
}

// GetPaidUnionID returns the UnionID of a user who has paid in this mini program.
//
// The user must have finished paying, and the call is only valid within five
// minutes of the payment. Without an open platform binding the call fails with
// 89002, and a trade id that WeChat cannot match fails with 89300.
//
// UnionID is the cross-application identifier of the user: it is what lets the
// same person be recognised across the apps bound to the same open platform
// account, which a mini program openid cannot do on its own.
func (a *Application) GetPaidUnionID(
	ctx context.Context,
	params PaidUnionIDParams,
) (*GetPaidUnionIDResponse, error) {
	query, err := params.query()
	if err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getPaidUnionIDPath, query)
	if err != nil {
		return nil, err
	}

	var result GetPaidUnionIDResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
