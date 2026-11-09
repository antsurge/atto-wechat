package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getAPIDomainIPPath returns the addresses of the WeChat API servers.
const getAPIDomainIPPath = "/cgi-bin/get_api_domain_ip"

// GetAPIDomainIPResponse is the reply of /cgi-bin/get_api_domain_ip.
type GetAPIDomainIPResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// IPList holds the addresses; both single addresses and CIDR ranges appear.
	IPList []string `json:"ip_list"`
}

// OK reports whether the reply carries addresses.
func (r *GetAPIDomainIPResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.IPList) > 0
}

// GetAPIDomainIPErrMsg returns the description of a getApiDomainIp error.
func GetAPIDomainIPErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// GetAPIDomainIPError turns the errcode from the reply into an error.
func GetAPIDomainIPError(r *GetAPIDomainIPResponse) error {
	return errors.New(GetAPIDomainIPErrMsg(r.ErrCode))
}

// GetAPIDomainIP lists the addresses this server reaches api.weixin.qq.com on.
//
// It is the outbound direction, for firewalls and egress allowlists: the answer
// changes as WeChat adds machines, so call it periodically instead of pinning the
// list. For the inbound direction — the addresses WeChat pushes callbacks from —
// use GetCallbackIP.
func (a *Application) GetAPIDomainIP(
	ctx context.Context,
) (*GetAPIDomainIPResponse, error) {
	resp, err := a.api.Get(ctx, getAPIDomainIPPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetAPIDomainIPResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
