package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// createLiveRoomPath creates a live room.
const (
	createLiveRoomPath          = "/wxaapi/broadcast/room/create"
	CreateLiveRoomDefaultErrMsg = "系统错误，请稍后再试"
)

// CreateLiveRoomErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/create, keyed by the errcode as it appears in the
// reply.
var CreateLiveRoomErrCodeMap = liveErrCodes(
	-1,
	1007,
	200002,
	300002,
	300028,
	300031,
	300032,
	300034,
	300036,
	300037,
	300038,
	300039,
	300040,
	300041,
	40001,
)

// CreateLiveRoomParams describes the room to create.
type CreateLiveRoomParams struct {
	// Name is the room name, 3 to 17 Chinese characters.
	Name string `json:"name"`

	// CoverImg is a mediaID from UploadTempMedia, valid for three days. The
	// image should be 1080x1920 and at most 2M.
	CoverImg string `json:"coverImg"`

	// StartTime is the planned start, in seconds. It must be at least ten
	// minutes from now and within six months.
	StartTime int64 `json:"startTime"`

	// EndTime is the planned end, in seconds. The room must last between 30
	// minutes and 24 hours.
	EndTime int64 `json:"endTime"`

	// AnchorName is the anchor nickname, 2 to 15 Chinese characters.
	AnchorName string `json:"anchorName"`

	// AnchorWechat is the anchor's WeChat id. An anchor who has not completed
	// real-name verification has to do it in the 小程序直播 mini program first.
	AnchorWechat string `json:"anchorWechat"`

	// SubAnchorWechat is the WeChat id of the deputy anchor.
	SubAnchorWechat string `json:"subAnchorWechat,omitempty"`

	// CreaterWechat restricts the room to its creator, the admins and the
	// anchor. Leaving it empty makes the room visible to every member.
	CreaterWechat string `json:"createrWechat,omitempty"`

	// ShareImg is a mediaID for the share card, 800x640 and at most 1M.
	ShareImg string `json:"shareImg"`

	// FeedsImg is a mediaID for the shopping channel cover, 800x800 and at most
	// 100KB.
	FeedsImg string `json:"feedsImg"`

	// IsFeedsPublic turns the official listing on (1) or off (0); nil leaves
	// WeChat's default, which is on.
	IsFeedsPublic *int `json:"isFeedsPublic,omitempty"`

	// LiveType is LiveRoomTypePush or LiveRoomTypePhone.
	LiveType int `json:"type"`

	// CloseLike, CloseGoods and CloseComment hide the like button, the goods
	// shelf and the comment entry. They cannot be reopened once the stream has
	// started.
	CloseLike    int `json:"closeLike"`
	CloseGoods   int `json:"closeGoods"`
	CloseComment int `json:"closeComment"`

	// CloseReplay, CloseShare and CloseKf hide the replay, share and customer
	// service entries; nil leaves WeChat's default.
	CloseReplay *int `json:"closeReplay,omitempty"`
	CloseShare  *int `json:"closeShare,omitempty"`
	CloseKf     *int `json:"closeKf,omitempty"`
}

// CheckCreateLiveRoomParams validates params before anything is sent to WeChat.
func CheckCreateLiveRoomParams(params CreateLiveRoomParams) error {
	switch {
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
	case params.LiveType != LiveRoomTypePhone && params.LiveType != LiveRoomTypePush:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: type must be LiveRoomTypePhone or LiveRoomTypePush",
		)
	}

	for name, flag := range map[string]int{
		"closeLike":    params.CloseLike,
		"closeGoods":   params.CloseGoods,
		"closeComment": params.CloseComment,
	} {
		if flag != 0 && flag != 1 {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: "+name+" must be 0 or 1",
			)
		}
	}

	return nil
}

// CreateLiveRoomResponse is the reply of /wxaapi/broadcast/room/create.
type CreateLiveRoomResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	RoomID    int64  `json:"roomId"`
	QRCodeURL string `json:"qrcode_url"`
}

// OK reports whether WeChat created the room.
func (r *CreateLiveRoomResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.RoomID != 0
}

// CreateLiveRoomErrMsg returns the description of a createRoom error.
func CreateLiveRoomErrMsg(code int) string {
	if msg, ok := CreateLiveRoomErrCodeMap[code]; ok {
		return msg
	}
	return CreateLiveRoomDefaultErrMsg
}

// CreateLiveRoomError turns the errcode from the reply into an error.
func CreateLiveRoomError(r *CreateLiveRoomResponse) error {
	return errors.New(CreateLiveRoomErrMsg(r.ErrCode))
}

// CreateLiveRoom creates a live room.
//
// The room shows up in the live room list right away, but it can only go live
// inside the window given by StartTime and EndTime; the mediaIDs passed as
// images expire after three days, which is why a room planned further out has
// to be created later or edited before it starts.
func (a *Application) CreateLiveRoom(
	ctx context.Context,
	params CreateLiveRoomParams,
) (*CreateLiveRoomResponse, error) {
	if err := CheckCreateLiveRoomParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, createLiveRoomPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result CreateLiveRoomResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
