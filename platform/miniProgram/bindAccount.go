package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// bindAccountPath binds or unbinds a courier account.
const (
	bindAccountPath          = "/cgi-bin/express/business/account/bind"
	BindAccountDefaultErrMsg = "系统错误，请稍后再试"
)

// BindAccountErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/account/bind, keyed by the errcode as it appears in
// the reply.
var BindAccountErrCodeMap = logisticsErrCodes()

// BindAccountParams is what to do with the courier account.
//
// Binding is what makes a courier usable at all: the biz_id/password pair is the
// account the merchant signed with that courier, and AddOrder refers to it by
// the same BizID.
type BindAccountParams struct {
	// Type is LogisticsBindTypeBind or LogisticsBindTypeUnbind.
	Type string `json:"type"`
	// BizID is the merchant's customer code at the courier.
	BizID string `json:"biz_id"`
	// DeliveryID is the courier id, see GetAllDelivery.
	DeliveryID string `json:"delivery_id"`
	// Password is the customer password; it is optional for EMS, SF and JD.
	Password string `json:"password,omitempty"`
	// RemarkContent is the note EMS requires when it reviews the account:
	// 电话/联系人/服务类型/发货地址.
	RemarkContent string `json:"remark_content,omitempty"`
}

// Bind type values carried in BindAccountParams.Type.
const (
	// LogisticsBindTypeBind binds the account.
	LogisticsBindTypeBind = "bind"
	// LogisticsBindTypeUnbind unbinds the account.
	LogisticsBindTypeUnbind = "unbind"
)

// CheckBindAccountParams validates params before anything is sent to WeChat.
func CheckBindAccountParams(params BindAccountParams) error {
	if params.Type != LogisticsBindTypeBind && params.Type != LogisticsBindTypeUnbind {
		return invalidLogisticsField("type must be bind or unbind")
	}
	if params.BizID == "" {
		return invalidLogisticsField("biz_id is required")
	}
	if params.DeliveryID == "" {
		return invalidLogisticsField("delivery_id is required")
	}

	return nil
}

// BindAccountResponse is the reply of /cgi-bin/express/business/account/bind.
//
// The endpoint returns no payload: success is only the absence of an errcode.
type BindAccountResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the change.
func (r *BindAccountResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// BindAccountErrMsg returns the description of a bind_account error.
func BindAccountErrMsg(code int) string {
	return logisticsErrMsg(BindAccountErrCodeMap, code, BindAccountDefaultErrMsg)
}

// BindAccountError turns the errcode from the reply into an error.
func BindAccountError(r *BindAccountResponse) error {
	return errors.New(BindAccountErrMsg(r.ErrCode))
}

// BindAccount binds or unbinds a courier account.
//
// Unbinding deletes the credentials WeChat stored, so waybills already created
// with that account keep working but no new one can be created.
func (a *Application) BindAccount(
	ctx context.Context,
	params BindAccountParams,
) (*BindAccountResponse, error) {
	if err := CheckBindAccountParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, bindAccountPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result BindAccountResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
