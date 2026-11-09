package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveRoleListPath lists the member roles of the live broadcast account.
const (
	getLiveRoleListPath          = "/wxaapi/broadcast/role/getrolelist"
	GetLiveRoleListDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveRoleListErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/role/getrolelist, keyed by the errcode as it appears in the
// reply.
var GetLiveRoleListErrCodeMap = liveErrCodes(
	200002,
)

// GetLiveRoleListParams selects the members to return.
type GetLiveRoleListParams struct {
	// Offset is the number of members to skip, from 0.
	Offset int

	// Limit is the page size.
	Limit int

	// Keyword filters by WeChat id or nickname; empty lists every member.
	Keyword string
}

// CheckGetLiveRoleListParams validates params before anything is sent to WeChat.
func CheckGetLiveRoleListParams(params GetLiveRoleListParams) error {
	switch {
	case params.Offset < 0:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: offset must not be negative",
		)
	case params.Limit <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: limit is required")
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetLiveRoleListParams) query() url.Values {
	q := url.Values{
		"offset": {strconv.Itoa(p.Offset)},
		"limit":  {strconv.Itoa(p.Limit)},
	}
	if p.Keyword != "" {
		q.Set("keyword", p.Keyword)
	}

	return q
}

// LiveRoleMember is one member with a role on the account.
type LiveRoleMember struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     int    `json:"role"`
}

// GetLiveRoleListResponse is the reply of /wxaapi/broadcast/role/getrolelist.
type GetLiveRoleListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	List  []LiveRoleMember `json:"list"`
	Total int              `json:"total"`
}

// OK reports whether WeChat answered with a usable list.
func (r *GetLiveRoleListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveRoleListErrMsg returns the description of a getrolelist error.
func GetLiveRoleListErrMsg(code int) string {
	if msg, ok := GetLiveRoleListErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveRoleListDefaultErrMsg
}

// GetLiveRoleListError turns the errcode from the reply into an error.
func GetLiveRoleListError(r *GetLiveRoleListResponse) error {
	return errors.New(GetLiveRoleListErrMsg(r.ErrCode))
}

// GetLiveRoleList lists the members with a role on the account.
//
// The list covers the whole account rather than one room, so an admin shows up
// once however many rooms it manages.
func (a *Application) GetLiveRoleList(
	ctx context.Context,
	params GetLiveRoleListParams,
) (*GetLiveRoleListResponse, error) {
	if err := CheckGetLiveRoleListParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getLiveRoleListPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetLiveRoleListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
