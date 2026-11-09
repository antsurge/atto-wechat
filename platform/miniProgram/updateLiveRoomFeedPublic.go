package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// updateLiveRoomFeedPublicPath turns the official listing of a room on or off.
const (
	updateLiveRoomFeedPublicPath          = "/wxaapi/broadcast/room/updatefeedpublic"
	UpdateLiveRoomFeedPublicDefaultErrMsg = "系统错误，请稍后再试"
)

// UpdateLiveRoomFeedPublicErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/room/updatefeedpublic, keyed by the errcode as
// it appears in the reply.
var UpdateLiveRoomFeedPublicErrCodeMap = liveErrCodes(
	300022,
	300023,
	300037,
	40001,
)

// UpdateLiveRoomFeedPublicParams changes the official listing of a room.
type UpdateLiveRoomFeedPublicParams struct {
	// RoomID is the room whose listing changes.
	RoomID int64 `json:"roomId"`

	// IsFeedsPublic is 1 to let WeChat list the room in the shopping channel and
	// 0 to keep it out.
	IsFeedsPublic int `json:"isFeedsPublic"`
}

// CheckUpdateLiveRoomFeedPublicParams validates params before anything is sent
// to WeChat.
func CheckUpdateLiveRoomFeedPublicParams(params UpdateLiveRoomFeedPublicParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.IsFeedsPublic != 0 && params.IsFeedsPublic != 1:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: isFeedsPublic must be 0 or 1",
		)
	}

	return nil
}

// UpdateLiveRoomFeedPublicResponse is the reply of
// /wxaapi/broadcast/room/updatefeedpublic.
//
// It carries no business payload: success is reported by errcode alone.
type UpdateLiveRoomFeedPublicResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the change.
func (r *UpdateLiveRoomFeedPublicResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdateLiveRoomFeedPublicErrMsg returns the description of an
// updateFeedPublic error.
func UpdateLiveRoomFeedPublicErrMsg(code int) string {
	if msg, ok := UpdateLiveRoomFeedPublicErrCodeMap[code]; ok {
		return msg
	}
	return UpdateLiveRoomFeedPublicDefaultErrMsg
}

// UpdateLiveRoomFeedPublicError turns the errcode from the reply into an error.
func UpdateLiveRoomFeedPublicError(r *UpdateLiveRoomFeedPublicResponse) error {
	return errors.New(UpdateLiveRoomFeedPublicErrMsg(r.ErrCode))
}

// UpdateLiveRoomFeedPublic turns the official listing of a room on or off.
//
// A listed room appears in the 购物直播 channel and can attract viewers beyond
// the ones who were shared the room; the channel cover is the one given as
// FeedsImg at creation.
func (a *Application) UpdateLiveRoomFeedPublic(
	ctx context.Context,
	params UpdateLiveRoomFeedPublicParams,
) (*UpdateLiveRoomFeedPublicResponse, error) {
	if err := CheckUpdateLiveRoomFeedPublicParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updateLiveRoomFeedPublicPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result UpdateLiveRoomFeedPublicResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
