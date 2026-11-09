package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// updateLiveRoomKFPath turns the customer service entry of a room on or off.
const (
	updateLiveRoomKFPath          = "/wxaapi/broadcast/room/updatekf"
	UpdateLiveRoomKFDefaultErrMsg = "系统错误，请稍后再试"
)

// UpdateLiveRoomKFErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/updatekf, keyed by the errcode as it appears in the
// reply.
var UpdateLiveRoomKFErrCodeMap = liveErrCodes(
	300022,
	300023,
	300038,
	40001,
)

// UpdateLiveRoomKFParams changes the customer service entry of a room.
type UpdateLiveRoomKFParams struct {
	// RoomID is the room whose entry changes.
	RoomID int64 `json:"roomId"`

	// CloseKf is 1 to hide the customer service entry and 0 to show it.
	CloseKf int `json:"closeKf"`
}

// CheckUpdateLiveRoomKFParams validates params before anything is sent to
// WeChat.
func CheckUpdateLiveRoomKFParams(params UpdateLiveRoomKFParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.CloseKf != 0 && params.CloseKf != 1:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: closeKf must be 0 or 1",
		)
	}

	return nil
}

// UpdateLiveRoomKFResponse is the reply of /wxaapi/broadcast/room/updatekf.
//
// It carries no business payload: success is reported by errcode alone.
type UpdateLiveRoomKFResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the change.
func (r *UpdateLiveRoomKFResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdateLiveRoomKFErrMsg returns the description of an updateKf error.
func UpdateLiveRoomKFErrMsg(code int) string {
	if msg, ok := UpdateLiveRoomKFErrCodeMap[code]; ok {
		return msg
	}
	return UpdateLiveRoomKFDefaultErrMsg
}

// UpdateLiveRoomKFError turns the errcode from the reply into an error.
func UpdateLiveRoomKFError(r *UpdateLiveRoomKFResponse) error {
	return errors.New(UpdateLiveRoomKFErrMsg(r.ErrCode))
}

// UpdateLiveRoomKF turns the customer service entry of a room on or off.
//
// Turning it on requires customer service accounts configured in the mini
// program console, otherwise WeChat answers 300038.
func (a *Application) UpdateLiveRoomKF(
	ctx context.Context,
	params UpdateLiveRoomKFParams,
) (*UpdateLiveRoomKFResponse, error) {
	if err := CheckUpdateLiveRoomKFParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updateLiveRoomKFPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result UpdateLiveRoomKFResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
