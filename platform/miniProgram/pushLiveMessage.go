package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// pushLiveMessagePath reminds the long-term subscribers that a room is live.
const (
	pushLiveMessagePath          = "/wxa/business/push_message"
	PushLiveMessageDefaultErrMsg = "系统错误，请稍后再试"
)

// PushLiveMessageErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/business/push_message, keyed by the errcode as it appears in the reply.
var PushLiveMessageErrCodeMap = liveErrCodes(
	200002,
	9410005,
	9410006,
)

// PushLiveMessageParams sends the live start reminder of a room.
type PushLiveMessageParams struct {
	// RoomID is the room that just went live.
	RoomID int64 `json:"room_id"`

	// UserOpenIDs are the subscribers to remind, taken from GetLiveFollowers.
	UserOpenIDs []string `json:"user_openid"`
}

// CheckPushLiveMessageParams validates params before anything is sent to WeChat.
func CheckPushLiveMessageParams(params PushLiveMessageParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: room_id is required")
	}

	if len(params.UserOpenIDs) == 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: user_openid is required")
	}

	for _, openID := range params.UserOpenIDs {
		if openID == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: user_openid must not contain an empty openid",
			)
		}
	}

	return nil
}

// PushLiveMessageResponse is the reply of /wxa/business/push_message.
type PushLiveMessageResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	MessageID string `json:"message_id"`
}

// OK reports whether WeChat accepted the reminder.
func (r *PushLiveMessageResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// PushLiveMessageErrMsg returns the description of a push_message error.
func PushLiveMessageErrMsg(code int) string {
	if msg, ok := PushLiveMessageErrCodeMap[code]; ok {
		return msg
	}
	return PushLiveMessageDefaultErrMsg
}

// PushLiveMessageError turns the errcode from the reply into an error.
func PushLiveMessageError(r *PushLiveMessageResponse) error {
	return errors.New(PushLiveMessageErrMsg(r.ErrCode))
}

// PushLiveMessage reminds the long-term subscribers that a room is live.
//
// A user subscribes once and is then reminded for every stream, which is why
// the list comes from GetLiveFollowers instead of a per-message template id. Two
// limits apply: the reminder only works while the room is actually live
// (9410006), and only one room per day can push (9410005).
func (a *Application) PushLiveMessage(
	ctx context.Context,
	params PushLiveMessageParams,
) (*PushLiveMessageResponse, error) {
	if err := CheckPushLiveMessageParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, pushLiveMessagePath, nil, params)
	if err != nil {
		return nil, err
	}

	var result PushLiveMessageResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
