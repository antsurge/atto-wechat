package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteSubAnchorPath removes the deputy anchor of a room.
const (
	deleteSubAnchorPath          = "/wxaapi/broadcast/room/deletesubanchor"
	DeleteSubAnchorDefaultErrMsg = "系统错误，请稍后再试"
)

// DeleteSubAnchorErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/deletesubanchor, keyed by the errcode as it appears in
// the reply.
var DeleteSubAnchorErrCodeMap = liveErrCodes(-1)

// DeleteSubAnchorParams identifies the room whose deputy anchor is removed.
type DeleteSubAnchorParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64 `json:"roomId"`
}

// CheckDeleteSubAnchorParams validates params before anything is sent to
// WeChat.
func CheckDeleteSubAnchorParams(params DeleteSubAnchorParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	return nil
}

// DeleteSubAnchorResponse is the reply of
// /wxaapi/broadcast/room/deletesubanchor.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteSubAnchorResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat removed the deputy anchor.
func (r *DeleteSubAnchorResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// DeleteSubAnchorErrMsg returns the description of a deleteSubAnchor error.
func DeleteSubAnchorErrMsg(code int) string {
	if msg, ok := DeleteSubAnchorErrCodeMap[code]; ok {
		return msg
	}
	return DeleteSubAnchorDefaultErrMsg
}

// DeleteSubAnchorError turns the errcode from the reply into an error.
func DeleteSubAnchorError(r *DeleteSubAnchorResponse) error {
	return errors.New(DeleteSubAnchorErrMsg(r.ErrCode))
}

// DeleteSubAnchor removes the deputy anchor of a room.
//
// WeChat identifies the deputy anchor by the room, so the room id is the whole
// request: a room has at most one.
func (a *Application) DeleteSubAnchor(
	ctx context.Context,
	params DeleteSubAnchorParams,
) (*DeleteSubAnchorResponse, error) {
	if err := CheckDeleteSubAnchorParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, deleteSubAnchorPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result DeleteSubAnchorResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
