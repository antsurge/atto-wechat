package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addLiveRolePath adds a member role to the live broadcast account.
const (
	addLiveRolePath          = "/wxaapi/broadcast/role/addrole"
	AddLiveRoleDefaultErrMsg = "系统错误，请稍后再试"
)

// AddLiveRoleErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/role/addrole, keyed by the errcode as it appears in the
// reply.
var AddLiveRoleErrCodeMap = liveErrCodes(
	200002,
	400001,
	400002,
	400003,
	400004,
)

// AddLiveRoleParams grants one role to one account.
type AddLiveRoleParams struct {
	// Username is the WeChat id receiving the role.
	Username string `json:"username"`

	// Role is LiveRoleAdmin, LiveRoleAnchor or LiveRoleOperator.
	Role int `json:"role"`
}

// CheckAddLiveRoleParams validates params before anything is sent to WeChat.
func CheckAddLiveRoleParams(params AddLiveRoleParams) error {
	switch {
	case params.Username == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: username is required")
	case params.Role == LiveRoleSuperAdmin:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: the super admin cannot be granted",
		)
	case params.Role != LiveRoleAdmin &&
		params.Role != LiveRoleAnchor &&
		params.Role != LiveRoleOperator:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: role must be LiveRoleAdmin, LiveRoleAnchor or LiveRoleOperator",
		)
	}

	return nil
}

// AddLiveRoleResponse is the reply of /wxaapi/broadcast/role/addrole.
type AddLiveRoleResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	CodeURL string `json:"codeurl"`
}

// OK reports whether WeChat granted the role.
//
// CodeURL is only returned for an anchor who has not completed real-name
// verification: it is the link the anchor opens to finish it, so an empty value
// is a valid success.
func (r *AddLiveRoleResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// AddLiveRoleErrMsg returns the description of an addrole error.
func AddLiveRoleErrMsg(code int) string {
	if msg, ok := AddLiveRoleErrCodeMap[code]; ok {
		return msg
	}
	return AddLiveRoleDefaultErrMsg
}

// AddLiveRoleError turns the errcode from the reply into an error.
func AddLiveRoleError(r *AddLiveRoleResponse) error {
	return errors.New(AddLiveRoleErrMsg(r.ErrCode))
}

// AddLiveRole grants a role to an account.
//
// The role is account wide: an anchor can open in every room, an admin manages
// the rooms, and an operator moderates them. When the account is an anchor who
// has not done real-name verification, WeChat returns a CodeURL instead of
// refusing, and the anchor has to verify before the role works.
func (a *Application) AddLiveRole(
	ctx context.Context,
	params AddLiveRoleParams,
) (*AddLiveRoleResponse, error) {
	if err := CheckAddLiveRoleParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addLiveRolePath, nil, params)
	if err != nil {
		return nil, err
	}

	var result AddLiveRoleResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
