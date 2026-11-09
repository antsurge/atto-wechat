package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteLiveRoomPath removes a live room.
const (
	deleteLiveRoomPath          = "/wxaapi/broadcast/room/deleteroom"
	DeleteLiveRoomDefaultErrMsg = "系统错误，请稍后再试"
)

// DeleteLiveRoomErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/deleteroom, keyed by the errcode as it appears in the
// reply.
var DeleteLiveRoomErrCodeMap = liveErrCodes(
	200002,
	300022,
	300023,
	40001,
)

// DeleteLiveRoomParams identifies the room to remove.
type DeleteLiveRoomParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64 `json:"id"`
}

// CheckDeleteLiveRoomParams validates params before anything is sent to WeChat.
func CheckDeleteLiveRoomParams(params DeleteLiveRoomParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: id is required")
	}

	return nil
}

// DeleteLiveRoomResponse is the reply of /wxaapi/broadcast/room/deleteroom.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteLiveRoomResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat removed the room.
func (r *DeleteLiveRoomResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// DeleteLiveRoomErrMsg returns the description of a deleteRoom error.
func DeleteLiveRoomErrMsg(code int) string {
	if msg, ok := DeleteLiveRoomErrCodeMap[code]; ok {
		return msg
	}
	return DeleteLiveRoomDefaultErrMsg
}

// DeleteLiveRoomError turns the errcode from the reply into an error.
func DeleteLiveRoomError(r *DeleteLiveRoomResponse) error {
	return errors.New(DeleteLiveRoomErrMsg(r.ErrCode))
}

// DeleteLiveRoom removes a live room.
//
// Deletion is refused while the room is in a state that does not allow it, and
// it cannot be undone: the room disappears from the list together with its
// goods and replays.
func (a *Application) DeleteLiveRoom(
	ctx context.Context,
	params DeleteLiveRoomParams,
) (*DeleteLiveRoomResponse, error) {
	if err := CheckDeleteLiveRoomParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, deleteLiveRoomPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result DeleteLiveRoomResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
