package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// updateLiveRoomReplayPath turns the replay of a room on or off.
const (
	updateLiveRoomReplayPath          = "/wxaapi/broadcast/room/updatereplay"
	UpdateLiveRoomReplayDefaultErrMsg = "系统错误，请稍后再试"
)

// UpdateLiveRoomReplayErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/room/updatereplay, keyed by the errcode as it appears in
// the reply.
var UpdateLiveRoomReplayErrCodeMap = liveErrCodes(
	300022,
	300023,
	40001,
)

// UpdateLiveRoomReplayParams changes the replay state of a room.
type UpdateLiveRoomReplayParams struct {
	// RoomID is the room whose replay changes.
	RoomID int64 `json:"roomId"`

	// CloseReplay is 1 to hide the replay and 0 to keep it.
	CloseReplay int `json:"closeReplay"`
}

// CheckUpdateLiveRoomReplayParams validates params before anything is sent to
// WeChat.
func CheckUpdateLiveRoomReplayParams(params UpdateLiveRoomReplayParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.CloseReplay != 0 && params.CloseReplay != 1:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: closeReplay must be 0 or 1",
		)
	}

	return nil
}

// UpdateLiveRoomReplayResponse is the reply of
// /wxaapi/broadcast/room/updatereplay.
//
// It carries no business payload: success is reported by errcode alone.
type UpdateLiveRoomReplayResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the change.
func (r *UpdateLiveRoomReplayResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdateLiveRoomReplayErrMsg returns the description of an updateReplay error.
func UpdateLiveRoomReplayErrMsg(code int) string {
	if msg, ok := UpdateLiveRoomReplayErrCodeMap[code]; ok {
		return msg
	}
	return UpdateLiveRoomReplayDefaultErrMsg
}

// UpdateLiveRoomReplayError turns the errcode from the reply into an error.
func UpdateLiveRoomReplayError(r *UpdateLiveRoomReplayResponse) error {
	return errors.New(UpdateLiveRoomReplayErrMsg(r.ErrCode))
}

// UpdateLiveRoomReplay turns the replay of a room on or off.
//
// The replay is generated after the stream ends, so the switch is what decides
// whether viewers can watch it later.
func (a *Application) UpdateLiveRoomReplay(
	ctx context.Context,
	params UpdateLiveRoomReplayParams,
) (*UpdateLiveRoomReplayResponse, error) {
	if err := CheckUpdateLiveRoomReplayParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updateLiveRoomReplayPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result UpdateLiveRoomReplayResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
