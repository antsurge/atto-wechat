package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// clearQuotaPath resets the daily call count of every server API.
const clearQuotaPath = "/cgi-bin/clear_quota"

// ClearQuotaParams identifies the account whose quota is reset.
type ClearQuotaParams struct {
	// AppID is the account to clear. It is usually the mini program itself, but
	// a third-party platform may clear a merchant's quota with the merchant's
	// appid while calling with a component token.
	AppID string
}

// CheckClearQuotaParams validates params before anything is sent to WeChat.
func CheckClearQuotaParams(params ClearQuotaParams) error {
	if params.AppID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ClearQuota appid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p ClearQuotaParams) body() *clearQuotaBody {
	return &clearQuotaBody{AppID: p.AppID}
}

// clearQuotaBody is the JSON body of /cgi-bin/clear_quota.
type clearQuotaBody struct {
	AppID string `json:"appid"`
}

// ClearQuotaResponse is the reply of /cgi-bin/clear_quota.
//
// It carries no business payload: success is reported by errcode alone.
type ClearQuotaResponse struct {
	response.WechatResponse
}

// OK reports whether the quota was reset.
func (r *ClearQuotaResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ClearQuotaErrMsg returns the description of a clearQuota error.
func ClearQuotaErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// ClearQuotaError turns the errcode from the reply into an error.
func ClearQuotaError(r *ClearQuotaResponse) error {
	return errors.New(ClearQuotaErrMsg(r.ErrCode))
}

// ClearQuota resets the daily call count of the account's server APIs.
//
// This is the blunt instrument: it clears every API at once, and WeChat caps how
// often it may be used (48006). Prefer ClearAPIQuota to reset a single API.
func (a *Application) ClearQuota(
	ctx context.Context,
	params ClearQuotaParams,
) (*ClearQuotaResponse, error) {
	if err := CheckClearQuotaParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, clearQuotaPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result ClearQuotaResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
