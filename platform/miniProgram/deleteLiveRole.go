package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteLiveRolePath removes a member role from the live broadcast account.
const (
	deleteLiveRolePath          = "/wxaapi/broadcast/role/deleterole"
	DeleteLiveRoleDefaultErrMsg = "系统错误，请稍后再试"
)

// DeleteLiveRoleErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/role/deleterole, keyed by the errcode as it appears in the
// reply.
var DeleteLiveRoleErrCodeMap = liveErrCodes(
	200002,
	400005,
)

// DeleteLiveRoleParams removes one role from one account.
type DeleteLiveRoleParams struct {
	// Username is the WeChat id losing the role.
	Username string `json:"username"`

	// Role is LiveRoleAdmin, LiveRoleAnchor or LiveRoleOperator.
	Role int `json:"role"`
}

// CheckDeleteLiveRoleParams validates params before anything is sent to WeChat.
func CheckDeleteLiveRoleParams(params DeleteLiveRoleParams) error {
	switch {
	case params.Username == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: username is required")
	case params.Role == LiveRoleSuperAdmin:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: the super admin cannot be removed",
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

// DeleteLiveRoleResponse is the reply of /wxaapi/broadcast/role/deleterole.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteLiveRoleResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat removed the role.
func (r *DeleteLiveRoleResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// DeleteLiveRoleErrMsg returns the description of a deleterole error.
func DeleteLiveRoleErrMsg(code int) string {
	if msg, ok := DeleteLiveRoleErrCodeMap[code]; ok {
		return msg
	}
	return DeleteLiveRoleDefaultErrMsg
}

// DeleteLiveRoleError turns the errcode from the reply into an error.
func DeleteLiveRoleError(r *DeleteLiveRoleResponse) error {
	return errors.New(DeleteLiveRoleErrMsg(r.ErrCode))
}

// DeleteLiveRole removes a role from an account.
//
// Removing the anchor role fails while the account still owns a room that has
// not gone live (400005): those rooms have to be deleted or handed over first.
func (a *Application) DeleteLiveRole(
	ctx context.Context,
	params DeleteLiveRoleParams,
) (*DeleteLiveRoleResponse, error) {
	if err := CheckDeleteLiveRoleParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, deleteLiveRolePath, nil, params)
	if err != nil {
		return nil, err
	}

	var result DeleteLiveRoleResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
