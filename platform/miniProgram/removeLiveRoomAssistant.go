package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// removeLiveRoomAssistantPath removes an assistant from a room.
const (
	removeLiveRoomAssistantPath          = "/wxaapi/broadcast/room/removeassistant"
	RemoveLiveRoomAssistantDefaultErrMsg = "系统错误，请稍后再试"
)

// RemoveLiveRoomAssistantErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/room/removeassistant, keyed by the errcode as
// it appears in the reply.
var RemoveLiveRoomAssistantErrCodeMap = liveErrCodes(
	300022,
	40001,
)

// RemoveLiveRoomAssistantParams removes one assistant from a room.
type RemoveLiveRoomAssistantParams struct {
	// RoomID is the room holding the assistant.
	RoomID int64 `json:"roomId"`

	// Username identifies the assistant to remove.
	Username string `json:"username"`
}

// CheckRemoveLiveRoomAssistantParams validates params before anything is sent to
// WeChat.
func CheckRemoveLiveRoomAssistantParams(params RemoveLiveRoomAssistantParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.Username == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: username is required")
	}

	return nil
}

// RemoveLiveRoomAssistantResponse is the reply of
// /wxaapi/broadcast/room/removeassistant.
//
// It carries no business payload: success is reported by errcode alone.
type RemoveLiveRoomAssistantResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat removed the assistant.
func (r *RemoveLiveRoomAssistantResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// RemoveLiveRoomAssistantErrMsg returns the description of a removeAssistant
// error.
func RemoveLiveRoomAssistantErrMsg(code int) string {
	if msg, ok := RemoveLiveRoomAssistantErrCodeMap[code]; ok {
		return msg
	}
	return RemoveLiveRoomAssistantDefaultErrMsg
}

// RemoveLiveRoomAssistantError turns the errcode from the reply into an error.
func RemoveLiveRoomAssistantError(r *RemoveLiveRoomAssistantResponse) error {
	return errors.New(RemoveLiveRoomAssistantErrMsg(r.ErrCode))
}

// RemoveLiveRoomAssistant removes an assistant from a room.
//
// The account loses its moderation rights in that room only.
func (a *Application) RemoveLiveRoomAssistant(
	ctx context.Context,
	params RemoveLiveRoomAssistantParams,
) (*RemoveLiveRoomAssistantResponse, error) {
	if err := CheckRemoveLiveRoomAssistantParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, removeLiveRoomAssistantPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result RemoveLiveRoomAssistantResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
