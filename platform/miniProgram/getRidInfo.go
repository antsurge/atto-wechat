package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getRIDInfoPath returns the details behind an rid.
const getRIDInfoPath = "/cgi-bin/openapi/rid/get"

// GetRIDInfoParams identifies the failed call to inspect.
type GetRIDInfoParams struct {
	// RID is the rid WeChat returned with the error.
	RID string
}

// CheckGetRIDInfoParams validates params before anything is sent to WeChat.
func CheckGetRIDInfoParams(params GetRIDInfoParams) error {
	if params.RID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetRIDInfo rid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetRIDInfoParams) body() *getRIDInfoBody {
	return &getRIDInfoBody{RID: p.RID}
}

// getRIDInfoBody is the JSON body of /cgi-bin/openapi/rid/get.
type getRIDInfoBody struct {
	RID string `json:"rid"`
}

// RIDRequest is what WeChat recorded for the rid.
type RIDRequest struct {
	// InvokeTime is when the call started, as a Unix timestamp.
	InvokeTime int64 `json:"invoke_time"`
	// CostInMS is how long it took, in milliseconds.
	CostInMS int `json:"cost_in_ms"`
	// RequestURL is the URL with its query parameters.
	RequestURL string `json:"request_url"`
	// RequestBody is the POST body, empty for a GET call.
	RequestBody string `json:"request_body"`
	// ResponseBody is what the endpoint answered.
	ResponseBody string `json:"response_body"`
	// ClientIP is the address the call came from.
	ClientIP string `json:"client_ip"`
}

// GetRIDInfoResponse is the reply of /cgi-bin/openapi/rid/get.
type GetRIDInfoResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Request *RIDRequest `json:"request"`
}

// OK reports whether the reply carries the request details.
func (r *GetRIDInfoResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Request != nil
}

// GetRIDInfoErrMsg returns the description of a getRidInfo error.
func GetRIDInfoErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// GetRIDInfoError turns the errcode from the reply into an error.
func GetRIDInfoError(r *GetRIDInfoResponse) error {
	return errors.New(GetRIDInfoErrMsg(r.ErrCode))
}

// GetRIDInfo returns what WeChat recorded for a failed call.
//
// The rid belongs to the account that produced it, and only that account's token
// can read it (76003 otherwise); rids older than 7 days are gone (76001). The
// reply includes the request and response bodies, so it is the fastest way to see
// why a call failed on WeChat's side rather than guessing from the errcode.
func (a *Application) GetRIDInfo(
	ctx context.Context,
	params GetRIDInfoParams,
) (*GetRIDInfoResponse, error) {
	if err := CheckGetRIDInfoParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getRIDInfoPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetRIDInfoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
