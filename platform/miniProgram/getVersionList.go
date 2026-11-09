package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getVersionListPath lists the client versions the log endpoints can filter by.
const getVersionListPath = "/wxaapi/log/get_client_version"

// Values of the type field of a version group.
const (
	// ClientVersionTypeClient groups the WeChat client versions, 1.
	ClientVersionTypeClient = 1
	// ClientVersionTypeServiceDirect groups the 服务直达 versions, 2.
	ClientVersionTypeServiceDirect = 2
)

// ClientVersionGroup is one kind of version list.
type ClientVersionGroup struct {
	// Type is one of the ClientVersionType* values.
	Type int `json:"type"`
	// ClientVersionList holds the version strings of that kind, e.g. "7.0.20".
	//
	// WeChat does not document the element type: every example comes back as an
	// empty array. It is declared as strings because that is what the filter
	// parameters take -- appVersion, sdkVersion and clientVersion are all
	// strings -- and because a wrong guess fails loudly at decode time rather
	// than silently dropping entries.
	ClientVersionList []string `json:"client_version_list"`
}

// GetVersionListResponse is the reply of /wxaapi/log/get_client_version.
type GetVersionListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// CVList is the list of version groups.
	CVList []ClientVersionGroup `json:"cvlist"`
}

// OK reports whether the list was read.
func (r *GetVersionListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Versions returns the version strings of one group.
//
// The reply always carries both groups, and either may be empty; looking a group
// up by type is how a caller picks the right one without relying on order.
func (r *GetVersionListResponse) Versions(versionType int) []string {
	if r == nil {
		return nil
	}

	for _, group := range r.CVList {
		if group.Type == versionType {
			return group.ClientVersionList
		}
	}

	return nil
}

// GetVersionListError turns the errcode from the reply into an error.
func GetVersionListError(r *GetVersionListResponse) error {
	return operationError(r.ErrCode)
}

// GetVersionList lists the client versions to filter logs and errors by.
//
// Like GetSceneList it is a reference table: the versions do not change often,
// so the reply is worth caching instead of fetching before every query.
func (a *Application) GetVersionList(ctx context.Context) (*GetVersionListResponse, error) {
	resp, err := a.api.Get(ctx, getVersionListPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetVersionListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
