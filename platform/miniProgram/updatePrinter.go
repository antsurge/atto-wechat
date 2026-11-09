package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// updatePrinterPath grants or revokes an employee's waybill printing permission.
const (
	updatePrinterPath          = "/cgi-bin/express/business/printer/update"
	UpdatePrinterDefaultErrMsg = "系统错误，请稍后再试"
)

// UpdatePrinterErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/printer/update, keyed by the errcode as it appears in
// the reply.
var UpdatePrinterErrCodeMap = logisticsErrCodes()

// Printer permission values carried in UpdatePrinterParams.UpdateType.
const (
	// LogisticsPrinterUpdateBind grants the printing permission.
	LogisticsPrinterUpdateBind = "bind"
	// LogisticsPrinterUpdateUnbind revokes the printing permission.
	LogisticsPrinterUpdateUnbind = "unbind"
)

// UpdatePrinterParams is the permission change to apply.
type UpdatePrinterParams struct {
	// OpenID is the employee who prints waybills.
	OpenID string `json:"openid"`
	// TagIDList is the comma separated list of waybill templates the employee may
	// print; it comes from the courier's tagid_list.
	TagIDList string `json:"tagid_list,omitempty"`
	// UpdateType is LogisticsPrinterUpdateBind or
	// LogisticsPrinterUpdateUnbind.
	UpdateType string `json:"update_type"`
}

// CheckUpdatePrinterParams validates params before anything is sent to WeChat.
func CheckUpdatePrinterParams(params UpdatePrinterParams) error {
	if params.OpenID == "" {
		return invalidLogisticsField("openid is required")
	}

	switch params.UpdateType {
	case LogisticsPrinterUpdateBind, LogisticsPrinterUpdateUnbind:
	default:
		return invalidLogisticsField("update_type must be bind or unbind")
	}

	return nil
}

// UpdatePrinterResponse is the reply of
// /cgi-bin/express/business/printer/update.
type UpdatePrinterResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the change.
func (r *UpdatePrinterResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdatePrinterErrMsg returns the description of an update printer error.
func UpdatePrinterErrMsg(code int) string {
	return logisticsErrMsg(UpdatePrinterErrCodeMap, code, UpdatePrinterDefaultErrMsg)
}

// UpdatePrinterError turns the errcode from the reply into an error.
func UpdatePrinterError(r *UpdatePrinterResponse) error {
	return errors.New(UpdatePrinterErrMsg(r.ErrCode))
}

// UpdatePrinter grants or revokes an employee's waybill printing permission.
//
// The employee must be a member of the mini program first; this only decides
// whether GetOrder's print_html may be rendered for them.
func (a *Application) UpdatePrinter(
	ctx context.Context,
	params UpdatePrinterParams,
) (*UpdatePrinterResponse, error) {
	if err := CheckUpdatePrinterParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updatePrinterPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result UpdatePrinterResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
