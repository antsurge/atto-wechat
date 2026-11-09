package miniProgram

import (
	"context"
	"errors"
	"strings"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getAPIQuotaPath queries the daily quota of one server API.
const getAPIQuotaPath = "/cgi-bin/openapi/quota/get"

// GetAPIQuotaParams identifies the API to inspect.
type GetAPIQuotaParams struct {
	// CGIPath is the API path, for example "/cgi-bin/message/custom/send".
	//
	// It must start with "/" and must not carry the "https://api.weixin.qq.com"
	// prefix; either mistake answers 76003.
	CGIPath string
}

// CheckGetAPIQuotaParams validates params before anything is sent to WeChat.
func CheckGetAPIQuotaParams(params GetAPIQuotaParams) error {
	return checkCGIPath("GetAPIQuota", params.CGIPath)
}

// body renders params as the JSON body the endpoint expects.
func (p GetAPIQuotaParams) body() *getAPIQuotaBody {
	return &getAPIQuotaBody{CGIPath: p.CGIPath}
}

// getAPIQuotaBody is the JSON body of /cgi-bin/openapi/quota/get.
type getAPIQuotaBody struct {
	CGIPath string `json:"cgi_path"`
}

// GetAPIQuotaResponse is the reply of /cgi-bin/openapi/quota/get.
type GetAPIQuotaResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Quota *APIQuota `json:"quota"`
	// RateLimit is the ordinary call-frequency limit.
	RateLimit *APIRateLimit `json:"rate_limit"`
	// ComponentRateLimit is the call-frequency limit when the call is made on
	// behalf of a merchant by a third-party platform. It is absent otherwise.
	ComponentRateLimit *APIRateLimit `json:"component_rate_limit"`
}

// OK reports whether the reply carries a quota to read.
func (r *GetAPIQuotaResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Quota != nil
}

// GetAPIQuotaErrMsg returns the description of a getApiQuota error.
func GetAPIQuotaErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// GetAPIQuotaError turns the errcode from the reply into an error.
func GetAPIQuotaError(r *GetAPIQuotaResponse) error {
	return errors.New(GetAPIQuotaErrMsg(r.ErrCode))
}

// GetAPIQuota queries how much of one server API's daily quota is left.
//
// The quota belongs to the account that owns the access token, so the answer is
// only meaningful for an API that account can call: 76022 means the token and the
// API belong to different accounts.
func (a *Application) GetAPIQuota(
	ctx context.Context,
	params GetAPIQuotaParams,
) (*GetAPIQuotaResponse, error) {
	if err := CheckGetAPIQuotaParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getAPIQuotaPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetAPIQuotaResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// checkCGIPath validates the cgi_path every quota endpoint takes.
//
// WeChat answers 76003 for a path with the host prefix or without the leading
// slash, and the mistake is easy to make because the docs show the full URL next
// to the path. Rejecting it here turns a round trip into a local error.
func checkCGIPath(operation, cgiPath string) error {
	if cgiPath == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+operation+" cgi_path is required",
		)
	}

	if strings.HasPrefix(cgiPath, "http://") || strings.HasPrefix(cgiPath, "https://") {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+operation+" cgi_path must not carry the host prefix",
		)
	}

	if !strings.HasPrefix(cgiPath, "/") {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+operation+" cgi_path must start with \"/\"",
		)
	}

	return nil
}
