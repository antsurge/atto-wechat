package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// clearAPIQuotaPath resets the daily call count of one server API.
const clearAPIQuotaPath = "/cgi-bin/openapi/quota/clear"

// ClearAPIQuotaParams identifies the API whose quota is reset.
type ClearAPIQuotaParams struct {
	// CGIPath is the API path, for example "/channels/ec/basics/info/get".
	//
	// WeChat documents this endpoint for the 微信小店 APIs, whose paths start with
	// "/channels/ec/"; the prefix is not enforced here because the same call is
	// accepted for other account types.
	CGIPath string
}

// CheckClearAPIQuotaParams validates params before anything is sent to WeChat.
func CheckClearAPIQuotaParams(params ClearAPIQuotaParams) error {
	return checkCGIPath("ClearAPIQuota", params.CGIPath)
}

// body renders params as the JSON body the endpoint expects.
func (p ClearAPIQuotaParams) body() *clearAPIQuotaBody {
	return &clearAPIQuotaBody{CGIPath: p.CGIPath}
}

// clearAPIQuotaBody is the JSON body of /cgi-bin/openapi/quota/clear.
type clearAPIQuotaBody struct {
	CGIPath string `json:"cgi_path"`
}

// ClearAPIQuotaResponse is the reply of /cgi-bin/openapi/quota/clear.
//
// It carries no business payload: success is reported by errcode alone.
type ClearAPIQuotaResponse struct {
	response.WechatResponse
}

// OK reports whether the quota was reset.
func (r *ClearAPIQuotaResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ClearAPIQuotaErrMsg returns the description of a clearApiQuota error.
func ClearAPIQuotaErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// ClearAPIQuotaError turns the errcode from the reply into an error.
func ClearAPIQuotaError(r *ClearAPIQuotaResponse) error {
	return errors.New(ClearAPIQuotaErrMsg(r.ErrCode))
}

// ClearAPIQuota resets the daily call count of one server API.
//
// Unlike ClearQuota it touches a single API, so a service that exhausted the quota
// of one endpoint does not have to reset all of them.
func (a *Application) ClearAPIQuota(
	ctx context.Context,
	params ClearAPIQuotaParams,
) (*ClearAPIQuotaResponse, error) {
	if err := CheckClearAPIQuotaParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, clearAPIQuotaPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result ClearAPIQuotaResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
