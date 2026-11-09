package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addLiveRoomAssistantPath adds assistants to a room.
const (
	addLiveRoomAssistantPath          = "/wxaapi/broadcast/room/addassistant"
	AddLiveRoomAssistantDefaultErrMsg = "系统错误，请稍后再试"
)

// AddLiveRoomAssistantErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/room/addassistant, keyed by the errcode as it appears in
// the reply.
var AddLiveRoomAssistantErrCodeMap = liveErrCodes(
	300022,
	300023,
	40001,
)

// LiveRoomAssistant is one assistant of a room.
type LiveRoomAssistant struct {
	// Username is the assistant's WeChat id.
	Username string `json:"username"`

	// Nickname is the display name inside the room.
	Nickname string `json:"nickname"`
}

// AddLiveRoomAssistantParams adds assistants to a room.
type AddLiveRoomAssistantParams struct {
	// RoomID is the room receiving the assistants.
	RoomID int64 `json:"roomId"`

	// Users are the assistants to add. An assistant can moderate the comments
	// and the goods of the room without being the anchor.
	Users []LiveRoomAssistant `json:"users"`
}

// CheckAddLiveRoomAssistantParams validates params before anything is sent to
// WeChat.
func CheckAddLiveRoomAssistantParams(params AddLiveRoomAssistantParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	if len(params.Users) == 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: users is required")
	}

	for _, user := range params.Users {
		if user.Username == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: users.username is required",
			)
		}
	}

	return nil
}

// AddLiveRoomAssistantResponse is the reply of
// /wxaapi/broadcast/room/addassistant.
//
// It carries no business payload: success is reported by errcode alone.
type AddLiveRoomAssistantResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat added the assistants.
func (r *AddLiveRoomAssistantResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// AddLiveRoomAssistantErrMsg returns the description of an addAssistant error.
func AddLiveRoomAssistantErrMsg(code int) string {
	if msg, ok := AddLiveRoomAssistantErrCodeMap[code]; ok {
		return msg
	}
	return AddLiveRoomAssistantDefaultErrMsg
}

// AddLiveRoomAssistantError turns the errcode from the reply into an error.
func AddLiveRoomAssistantError(r *AddLiveRoomAssistantResponse) error {
	return errors.New(AddLiveRoomAssistantErrMsg(r.ErrCode))
}

// AddLiveRoomAssistant adds assistants to a room.
//
// Assistants are per room, so the same account has to be added again in every
// room it moderates.
func (a *Application) AddLiveRoomAssistant(
	ctx context.Context,
	params AddLiveRoomAssistantParams,
) (*AddLiveRoomAssistantResponse, error) {
	if err := CheckAddLiveRoomAssistantParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addLiveRoomAssistantPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result AddLiveRoomAssistantResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
