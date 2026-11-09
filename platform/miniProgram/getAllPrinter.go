package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getAllPrinterPath lists the employees allowed to print waybills.
const (
	getAllPrinterPath          = "/cgi-bin/express/business/printer/getall"
	GetAllPrinterDefaultErrMsg = "系统错误，请稍后再试"
)

// GetAllPrinterErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/printer/getall, keyed by the errcode as it appears in
// the reply.
var GetAllPrinterErrCodeMap = logisticsErrCodes()

// GetAllPrinterResponse is the reply of
// /cgi-bin/express/business/printer/getall.
type GetAllPrinterResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// Count is the number of printers.
	Count int `json:"count"`
	// OpenID lists the employees allowed to print.
	OpenID []string `json:"openid"`
}

// OK reports whether WeChat answered with a usable list.
func (r *GetAllPrinterResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetAllPrinterErrMsg returns the description of a getall printer error.
func GetAllPrinterErrMsg(code int) string {
	return logisticsErrMsg(GetAllPrinterErrCodeMap, code, GetAllPrinterDefaultErrMsg)
}

// GetAllPrinterError turns the errcode from the reply into an error.
func GetAllPrinterError(r *GetAllPrinterResponse) error {
	return errors.New(GetAllPrinterErrMsg(r.ErrCode))
}

// GetAllPrinter lists the employees allowed to print waybills.
//
// An empty list means no permission has been granted yet; pair it with
// UpdatePrinter to grant one.
func (a *Application) GetAllPrinter(ctx context.Context) (*GetAllPrinterResponse, error) {
	resp, err := a.api.Get(ctx, getAllPrinterPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetAllPrinterResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
