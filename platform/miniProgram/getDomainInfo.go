package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getDomainInfoPath reads the domains configured for the mini program.
const getDomainInfoPath = "/wxa/getwxadevinfo"

// Values of the action parameter: which group of domains to read.
const (
	// DomainActionAll reads every group; it is what an empty action does.
	DomainActionAll = ""
	// DomainActionBiz reads the business domains only.
	DomainActionBiz = "getbizdomain"
	// DomainActionServer reads the server domains only.
	DomainActionServer = "getserverdomain"
)

// GetDomainInfoParams selects the group of domains to read.
type GetDomainInfoParams struct {
	// Action is one of the DomainAction* values; empty reads everything.
	Action string
}

// CheckGetDomainInfoParams validates params before anything is sent.
func CheckGetDomainInfoParams(params GetDomainInfoParams) error {
	switch params.Action {
	case DomainActionAll, DomainActionBiz, DomainActionServer:
		return nil
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetDomainInfo action must be getbizdomain or getserverdomain",
		)
	}
}

// body renders params as the JSON body the endpoint expects.
func (p GetDomainInfoParams) body() *getDomainInfoBody {
	return &getDomainInfoBody{Action: p.Action}
}

// getDomainInfoBody is the JSON body of a domain query.
type getDomainInfoBody struct {
	Action string `json:"action,omitempty"`
}

// GetDomainInfoResponse is the reply of /wxa/getwxadevinfo.
//
// Only the groups the action asked for are filled: a nil slice means "not
// requested", an empty non-nil slice means "requested and none configured".
type GetDomainInfoResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// RequestDomain is the list of request 合法域名.
	RequestDomain []string `json:"requestdomain"`
	// WSRequestDomain is the list of socket 合法域名.
	WSRequestDomain []string `json:"wsrequestdomain"`
	// UploadDomain is the list of uploadFile 合法域名.
	UploadDomain []string `json:"uploaddomain"`
	// DownloadDomain is the list of downloadFile 合法域名.
	DownloadDomain []string `json:"downloaddomain"`
	// UDPDomain is the list of udp 合法域名.
	UDPDomain []string `json:"udpdomain"`
	// BizDomain is the list of 业务域名.
	BizDomain []string `json:"bizdomain"`
}

// OK reports whether the domains were read.
func (r *GetDomainInfoResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ServerDomains returns the five server-side lists keyed by kind.
//
// The kind names are the ones WeChat uses, so a caller can report which list a
// missing domain was expected in.
func (r *GetDomainInfoResponse) ServerDomains() map[string][]string {
	if r == nil {
		return nil
	}

	return map[string][]string{
		"request":  r.RequestDomain,
		"socket":   r.WSRequestDomain,
		"upload":   r.UploadDomain,
		"download": r.DownloadDomain,
		"udp":      r.UDPDomain,
	}
}

// ContainsServerDomain reports whether domain appears in any server-side list.
//
// It is the usual reason to call the endpoint: a request域名 missing from the
// list fails in the client without ever reaching the server, and the error there
// does not name the domain that was refused.
func (r *GetDomainInfoResponse) ContainsServerDomain(domain string) bool {
	for _, domains := range r.ServerDomains() {
		for _, configured := range domains {
			if configured == domain {
				return true
			}
		}
	}

	return false
}

// GetDomainInfoError turns the errcode from the reply into an error.
func GetDomainInfoError(r *GetDomainInfoResponse) error {
	return operationError(r.ErrCode)
}

// GetDomainInfo reads the domains configured for the mini program.
//
// The endpoint is a POST even though it only reads: the group to read travels in
// the JSON body, not in the query.
func (a *Application) GetDomainInfo(
	ctx context.Context,
	params GetDomainInfoParams,
) (*GetDomainInfoResponse, error) {
	if err := CheckGetDomainInfoParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getDomainInfoPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetDomainInfoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
