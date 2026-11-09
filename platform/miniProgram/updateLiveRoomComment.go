package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// updateLiveRoomCommentPath mutes or unmutes the comments of a room.
const (
	updateLiveRoomCommentPath          = "/wxaapi/broadcast/room/updatecomment"
	UpdateLiveRoomCommentDefaultErrMsg = "系统错误，请稍后再试"
)

// UpdateLiveRoomCommentErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/room/updatecomment, keyed by the errcode as it
// appears in the reply.
var UpdateLiveRoomCommentErrCodeMap = liveErrCodes(
	300022,
	300023,
	40001,
)

// UpdateLiveRoomCommentParams changes the comment state of a room.
type UpdateLiveRoomCommentParams struct {
	// RoomID is the room whose comments change. WeChat names this field id.
	RoomID int64 `json:"id"`

	// BanComment is 1 to mute the comments and 0 to reopen them.
	BanComment int `json:"banComment"`
}

// CheckUpdateLiveRoomCommentParams validates params before anything is sent to
// WeChat.
func CheckUpdateLiveRoomCommentParams(params UpdateLiveRoomCommentParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: id is required")
	case params.BanComment != 0 && params.BanComment != 1:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: banComment must be 0 or 1",
		)
	}

	return nil
}

// UpdateLiveRoomCommentResponse is the reply of
// /wxaapi/broadcast/room/updatecomment.
//
// It carries no business payload: success is reported by errcode alone.
type UpdateLiveRoomCommentResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the change.
func (r *UpdateLiveRoomCommentResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdateLiveRoomCommentErrMsg returns the description of an updateComment error.
func UpdateLiveRoomCommentErrMsg(code int) string {
	if msg, ok := UpdateLiveRoomCommentErrCodeMap[code]; ok {
		return msg
	}
	return UpdateLiveRoomCommentDefaultErrMsg
}

// UpdateLiveRoomCommentError turns the errcode from the reply into an error.
func UpdateLiveRoomCommentError(r *UpdateLiveRoomCommentResponse) error {
	return errors.New(UpdateLiveRoomCommentErrMsg(r.ErrCode))
}

// UpdateLiveRoomComment mutes or unmutes the comments of a room.
//
// Muting hides the comment entry for every viewer; it does not delete what was
// already posted.
func (a *Application) UpdateLiveRoomComment(
	ctx context.Context,
	params UpdateLiveRoomCommentParams,
) (*UpdateLiveRoomCommentResponse, error) {
	if err := CheckUpdateLiveRoomCommentParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updateLiveRoomCommentPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result UpdateLiveRoomCommentResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
