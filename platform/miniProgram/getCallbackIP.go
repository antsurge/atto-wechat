package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getCallbackIPPath returns the addresses WeChat pushes callbacks from.
const getCallbackIPPath = "/cgi-bin/getcallbackip"

// GetCallbackIPResponse is the reply of /cgi-bin/getcallbackip.
type GetCallbackIPResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// IPList holds the addresses; both single addresses and CIDR ranges appear.
	IPList []string `json:"ip_list"`
}

// OK reports whether the reply carries addresses.
func (r *GetCallbackIPResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.IPList) > 0
}

// GetCallbackIPErrMsg returns the description of a getCallbackIp error.
func GetCallbackIPErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// GetCallbackIPError turns the errcode from the reply into an error.
func GetCallbackIPError(r *GetCallbackIPResponse) error {
	return errors.New(GetCallbackIPErrMsg(r.ErrCode))
}

// GetCallbackIP lists the addresses WeChat pushes callbacks from.
//
// It is the inbound direction, for a firewall that only lets the callback
// endpoint be reached from WeChat. The list changes over time, so refresh it
// instead of pinning it, and remember that the callback is authenticated by the
// signature in the body — the source address is a second line of defence, not the
// first.
func (a *Application) GetCallbackIP(
	ctx context.Context,
) (*GetCallbackIPResponse, error) {
	resp, err := a.api.Get(ctx, getCallbackIPPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetCallbackIPResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
