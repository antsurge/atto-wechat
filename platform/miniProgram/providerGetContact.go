package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerGetContactPath reads the sender and receiver of a waybill.
const (
	providerGetContactPath          = "/cgi-bin/express/delivery/contact/get"
	ProviderGetContactDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderGetContactErrCodeMap holds the errcode descriptions WeChat documents
// for /cgi-bin/express/delivery/contact/get, keyed by the errcode as it appears
// in the reply.
var ProviderGetContactErrCodeMap = logisticsErrCodes()

// ProviderGetContactParams identifies the waybill whose contacts to read.
type ProviderGetContactParams struct {
	// Token is the Token WeChat pushed with the order.
	Token string `json:"token"`
	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
}

// CheckProviderGetContactParams validates params before anything is sent to
// WeChat.
func CheckProviderGetContactParams(params ProviderGetContactParams) error {
	if params.Token == "" {
		return invalidLogisticsField("token is required")
	}
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}

	return nil
}

// ProviderGetContactResponse is the reply of
// /cgi-bin/express/delivery/contact/get.
type ProviderGetContactResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
	// Sender is who ships the parcel.
	Sender ProviderContact `json:"sender"`
	// Receiver is who receives the parcel.
	Receiver ProviderContact `json:"receiver"`
}

// OK reports whether WeChat returned the contacts.
func (r *ProviderGetContactResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.WaybillID != ""
}

// ProviderGetContactErrMsg returns the description of a get contact error.
func ProviderGetContactErrMsg(code int) string {
	return logisticsErrMsg(ProviderGetContactErrCodeMap, code, ProviderGetContactDefaultErrMsg)
}

// ProviderGetContactError turns the errcode from the reply into an error.
func ProviderGetContactError(r *ProviderGetContactResponse) error {
	return errors.New(ProviderGetContactErrMsg(r.ErrCode))
}

// ProviderGetContact reads the sender and receiver of a waybill.
//
// The order event already carries both, but it is pushed only once and may be
// missed; this is how a courier recovers them, and how it gets them when the
// event was sent without the contact fields.
func (a *Application) ProviderGetContact(
	ctx context.Context,
	params ProviderGetContactParams,
) (*ProviderGetContactResponse, error) {
	if err := CheckProviderGetContactParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerGetContactPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderGetContactResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
