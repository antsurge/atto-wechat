package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// modifySubAnchorPath replaces the deputy anchor of a room.
const (
	modifySubAnchorPath          = "/wxaapi/broadcast/room/modifysubanchor"
	ModifySubAnchorDefaultErrMsg = "系统错误，请稍后再试"
)

// ModifySubAnchorErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/modifysubanchor, keyed by the errcode as it appears in
// the reply.
var ModifySubAnchorErrCodeMap = liveErrCodes(
	500002,
	500004,
)

// ModifySubAnchorParams replaces the deputy anchor of a room.
type ModifySubAnchorParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64 `json:"roomId"`

	// Username is the new deputy anchor's WeChat id.
	Username string `json:"username"`
}

// CheckModifySubAnchorParams validates params before anything is sent to
// WeChat.
func CheckModifySubAnchorParams(params ModifySubAnchorParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.Username == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: username is required")
	}

	return nil
}

// ModifySubAnchorResponse is the reply of
// /wxaapi/broadcast/room/modifysubanchor.
//
// It carries no business payload: success is reported by errcode alone.
type ModifySubAnchorResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the new deputy anchor.
func (r *ModifySubAnchorResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ModifySubAnchorErrMsg returns the description of a modifySubAnchor error.
func ModifySubAnchorErrMsg(code int) string {
	if msg, ok := ModifySubAnchorErrCodeMap[code]; ok {
		return msg
	}
	return ModifySubAnchorDefaultErrMsg
}

// ModifySubAnchorError turns the errcode from the reply into an error.
func ModifySubAnchorError(r *ModifySubAnchorResponse) error {
	return errors.New(ModifySubAnchorErrMsg(r.ErrCode))
}

// ModifySubAnchor replaces the deputy anchor of a room.
//
// It is the way to swap one deputy anchor for another: WeChat refuses a
// duplicate of the account already set, so removing it first is not needed.
func (a *Application) ModifySubAnchor(
	ctx context.Context,
	params ModifySubAnchorParams,
) (*ModifySubAnchorResponse, error) {
	if err := CheckModifySubAnchorParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, modifySubAnchorPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ModifySubAnchorResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
