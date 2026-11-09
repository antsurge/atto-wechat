package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// providerPreviewTemplatePath renders a waybill template without creating a
// waybill.
const (
	providerPreviewTemplatePath          = "/cgi-bin/express/delivery/template/preview"
	ProviderPreviewTemplateDefaultErrMsg = "系统错误，请稍后再试"
)

// ProviderPreviewTemplateErrCodeMap holds the errcode descriptions WeChat
// documents for /cgi-bin/express/delivery/template/preview, keyed by the errcode
// as it appears in the reply.
var ProviderPreviewTemplateErrCodeMap = logisticsErrCodes()

// ProviderPreviewTemplateParams is the template and data to render.
type ProviderPreviewTemplateParams struct {
	// WaybillID is the waybill number the preview is for.
	WaybillID string `json:"waybill_id"`
	// WaybillTemplate is the HTML template, base64 encoded.
	WaybillTemplate string `json:"waybill_template"`
	// WaybillData is the waybill data as a JSON string, use WaybillDataJSON to
	// build it from the WaybillData returned by the order event.
	WaybillData string `json:"waybill_data"`
	// Custom is the merchant order that produced the waybill, in the exact shape
	// of AddOrderParams.
	Custom AddOrderParams `json:"custom"`
}

// CheckProviderPreviewTemplateParams validates params before anything is sent to
// WeChat.
func CheckProviderPreviewTemplateParams(params ProviderPreviewTemplateParams) error {
	if params.WaybillID == "" {
		return invalidLogisticsField("waybill_id is required")
	}
	if params.WaybillTemplate == "" {
		return invalidLogisticsField("waybill_template is required")
	}
	if params.WaybillData == "" {
		return invalidLogisticsField("waybill_data is required")
	}
	// The template is rendered against the order, so an unusable order would only
	// produce a blank sheet.
	if err := CheckAddOrderParams(params.Custom); err != nil {
		return err
	}

	return nil
}

// ProviderPreviewTemplateResponse is the reply of
// /cgi-bin/express/delivery/template/preview.
type ProviderPreviewTemplateResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// WaybillID is the waybill number the preview is for.
	WaybillID string `json:"waybill_id"`
	// RenderedWaybillTemplate is the rendered HTML, base64 encoded.
	RenderedWaybillTemplate string `json:"rendered_waybill_template"`
}

// OK reports whether WeChat rendered the template.
func (r *ProviderPreviewTemplateResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.RenderedWaybillTemplate != ""
}

// ProviderPreviewTemplateErrMsg returns the description of a preview template
// error.
func ProviderPreviewTemplateErrMsg(code int) string {
	return logisticsErrMsg(ProviderPreviewTemplateErrCodeMap, code, ProviderPreviewTemplateDefaultErrMsg)
}

// ProviderPreviewTemplateError turns the errcode from the reply into an error.
func ProviderPreviewTemplateError(r *ProviderPreviewTemplateResponse) error {
	return errors.New(ProviderPreviewTemplateErrMsg(r.ErrCode))
}

// ProviderPreviewTemplate renders a waybill template without creating a waybill.
//
// It is how a courier checks a template before shipping it: render a sample,
// look at the HTML, only then use the template in production. The reply is
// base64 encoded, so decode it before writing the file.
func (a *Application) ProviderPreviewTemplate(
	ctx context.Context,
	params ProviderPreviewTemplateParams,
) (*ProviderPreviewTemplateResponse, error) {
	if err := CheckProviderPreviewTemplateParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, providerPreviewTemplatePath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ProviderPreviewTemplateResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
