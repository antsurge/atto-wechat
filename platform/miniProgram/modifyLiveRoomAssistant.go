package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// modifyLiveRoomAssistantPath renames an assistant of a room.
const (
	modifyLiveRoomAssistantPath          = "/wxaapi/broadcast/room/modifyassistant"
	ModifyLiveRoomAssistantDefaultErrMsg = "系统错误，请稍后再试"
)

// ModifyLiveRoomAssistantErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/room/modifyassistant, keyed by the errcode as
// it appears in the reply.
var ModifyLiveRoomAssistantErrCodeMap = liveErrCodes(
	300022,
	300023,
	40001,
)

// ModifyLiveRoomAssistantParams renames one assistant of a room.
type ModifyLiveRoomAssistantParams struct {
	// RoomID is the room holding the assistant.
	RoomID int64 `json:"roomId"`

	// Username identifies the assistant.
	Username string `json:"username"`

	// Nickname is the new display name inside the room.
	Nickname string `json:"nickname"`
}

// CheckModifyLiveRoomAssistantParams validates params before anything is sent to
// WeChat.
func CheckModifyLiveRoomAssistantParams(params ModifyLiveRoomAssistantParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.Username == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: username is required")
	case params.Nickname == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: nickname is required")
	}

	return nil
}

// ModifyLiveRoomAssistantResponse is the reply of
// /wxaapi/broadcast/room/modifyassistant.
//
// It carries no business payload: success is reported by errcode alone.
type ModifyLiveRoomAssistantResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the new nickname.
func (r *ModifyLiveRoomAssistantResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ModifyLiveRoomAssistantErrMsg returns the description of a modifyAssistant
// error.
func ModifyLiveRoomAssistantErrMsg(code int) string {
	if msg, ok := ModifyLiveRoomAssistantErrCodeMap[code]; ok {
		return msg
	}
	return ModifyLiveRoomAssistantDefaultErrMsg
}

// ModifyLiveRoomAssistantError turns the errcode from the reply into an error.
func ModifyLiveRoomAssistantError(r *ModifyLiveRoomAssistantResponse) error {
	return errors.New(ModifyLiveRoomAssistantErrMsg(r.ErrCode))
}

// ModifyLiveRoomAssistant renames an assistant of a room.
//
// Only the nickname shown in the room changes; the account stays the same.
func (a *Application) ModifyLiveRoomAssistant(
	ctx context.Context,
	params ModifyLiveRoomAssistantParams,
) (*ModifyLiveRoomAssistantResponse, error) {
	if err := CheckModifyLiveRoomAssistantParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, modifyLiveRoomAssistantPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ModifyLiveRoomAssistantResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
