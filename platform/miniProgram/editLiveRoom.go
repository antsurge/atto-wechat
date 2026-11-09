package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// editLiveRoomPath updates a live room.
const (
	editLiveRoomPath          = "/wxaapi/broadcast/room/editroom"
	EditLiveRoomDefaultErrMsg = "系统错误，请稍后再试"
)

// EditLiveRoomErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/editroom, keyed by the errcode as it appears in the
// reply.
var EditLiveRoomErrCodeMap = liveErrCodes(
	-1,
	200002,
	300002,
	300022,
	300023,
	300030,
	300031,
	300032,
	300037,
	40001,
)

// EditLiveRoomParams describes the room after the edit.
//
// WeChat replaces the room with what is sent here, so every required field has
// to carry its current value even when only one of them changes.
type EditLiveRoomParams struct {
	// RoomID identifies the room to edit.
	RoomID int64 `json:"id"`

	// Name is the room name, 3 to 17 Chinese characters.
	Name string `json:"name"`

	// CoverImg is a mediaID from UploadTempMedia, valid for three days.
	CoverImg string `json:"coverImg"`

	// StartTime is the planned start, in seconds.
	StartTime int64 `json:"startTime"`

	// EndTime is the planned end, in seconds.
	EndTime int64 `json:"endTime"`

	// AnchorName is the anchor nickname, 2 to 15 Chinese characters.
	AnchorName string `json:"anchorName"`

	// AnchorWechat is the anchor's WeChat id.
	AnchorWechat string `json:"anchorWechat"`

	// ShareImg is a mediaID for the share card.
	ShareImg string `json:"shareImg"`

	// FeedsImg is a mediaID for the shopping channel cover.
	FeedsImg string `json:"feedsImg"`

	// IsFeedsPublic turns the official listing on (1) or off (0); nil leaves
	// the current setting.
	IsFeedsPublic *int `json:"isFeedsPublic,omitempty"`

	// CloseLike, CloseGoods and CloseComment hide the like button, the goods
	// shelf and the comment entry. They cannot be reopened once the stream has
	// started.
	CloseLike    int `json:"closeLike"`
	CloseGoods   int `json:"closeGoods"`
	CloseComment int `json:"closeComment"`

	// CloseReplay, CloseShare and CloseKf hide the replay, share and customer
	// service entries; nil leaves the current setting.
	CloseReplay *int `json:"closeReplay,omitempty"`
	CloseShare  *int `json:"closeShare,omitempty"`
	CloseKf     *int `json:"closeKf,omitempty"`
}

// CheckEditLiveRoomParams validates params before anything is sent to WeChat.
func CheckEditLiveRoomParams(params EditLiveRoomParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: id is required")
	case params.Name == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: name is required")
	case params.CoverImg == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: coverImg is required")
	case params.StartTime <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: startTime is required")
	case params.EndTime <= params.StartTime:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: endTime must be later than startTime",
		)
	case params.AnchorName == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: anchorName is required")
	case params.AnchorWechat == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: anchorWechat is required")
	case params.ShareImg == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: shareImg is required")
	case params.FeedsImg == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: feedsImg is required")
	}

	return nil
}

// EditLiveRoomResponse is the reply of /wxaapi/broadcast/room/editroom.
//
// It carries no business payload: success is reported by errcode alone.
type EditLiveRoomResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the edit.
func (r *EditLiveRoomResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// EditLiveRoomErrMsg returns the description of an editRoom error.
func EditLiveRoomErrMsg(code int) string {
	if msg, ok := EditLiveRoomErrCodeMap[code]; ok {
		return msg
	}
	return EditLiveRoomDefaultErrMsg
}

// EditLiveRoomError turns the errcode from the reply into an error.
func EditLiveRoomError(r *EditLiveRoomResponse) error {
	return errors.New(EditLiveRoomErrMsg(r.ErrCode))
}

// EditLiveRoom edits a live room.
//
// Editing is refused once the room state no longer allows it, and the room
// options that were locked at creation (like the like button) stay locked.
func (a *Application) EditLiveRoom(
	ctx context.Context,
	params EditLiveRoomParams,
) (*EditLiveRoomResponse, error) {
	if err := CheckEditLiveRoomParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, editLiveRoomPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result EditLiveRoomResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
