package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveRoomListPath lists live rooms, or the replay clips of one room.
const (
	getLiveRoomListPath          = "/wxa/business/getliveinfo"
	GetLiveRoomListDefaultErrMsg = "系统错误，请稍后再试"

	// LiveRoomActionGetReplay switches the endpoint from listing rooms to
	// listing the replay clips of a single room.
	LiveRoomActionGetReplay = "get_replay"
)

// GetLiveRoomListErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/business/getliveinfo, keyed by the errcode as it appears in the reply.
var GetLiveRoomListErrCodeMap = liveErrCodes(
	-1,
	40001,
)

// GetLiveRoomListParams selects the rooms, or the replay clips, to return.
type GetLiveRoomListParams struct {
	// Start is the offset of the first item, from 0.
	Start int `json:"start"`

	// Limit is the page size; WeChat recommends at most 100.
	Limit int `json:"limit"`

	// Action is LiveRoomActionGetReplay to fetch replay clips instead of rooms.
	Action string `json:"action,omitempty"`

	// RoomID is the room whose replays are wanted; required once Action is set.
	RoomID int64 `json:"room_id,omitempty"`
}

// CheckGetLiveRoomListParams validates params before anything is sent to
// WeChat.
func CheckGetLiveRoomListParams(params GetLiveRoomListParams) error {
	if params.Start < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: start must not be negative",
		)
	}

	if params.Limit <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: limit is required")
	}

	if params.Action != "" && params.Action != LiveRoomActionGetReplay {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: action must be LiveRoomActionGetReplay",
		)
	}

	if params.Action == LiveRoomActionGetReplay && params.RoomID <= 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: room_id is required when action is set",
		)
	}

	return nil
}

// LiveRoomGoods is one goods put on the shelf of a room.
type LiveRoomGoods struct {
	Name            string  `json:"name"`
	CoverImg        string  `json:"cover_img"`
	URL             string  `json:"url"`
	Price           float64 `json:"price"`
	Price2          float64 `json:"price2"`
	PriceType       int     `json:"price_type"`
	GoodsID         int64   `json:"goods_id"`
	ThirdPartyAppID string  `json:"third_party_appid"`
}

// LiveRoomInfo is one live room.
type LiveRoomInfo struct {
	Name          string          `json:"name"`
	CoverImg      string          `json:"cover_img"`
	StartTime     int64           `json:"start_time"`
	EndTime       int64           `json:"end_time"`
	AnchorName    string          `json:"anchor_name"`
	RoomID        int64           `json:"roomid"`
	Goods         []LiveRoomGoods `json:"goods"`
	LiveStatus    int             `json:"live_status"`
	ShareImg      string          `json:"share_img"`
	LiveType      int             `json:"live_type"`
	CloseLike     int             `json:"close_like"`
	CloseGoods    int             `json:"close_goods"`
	CloseComment  int             `json:"close_comment"`
	CloseKf       int             `json:"close_kf"`
	CloseReplay   int             `json:"close_replay"`
	IsFeedsPublic int             `json:"is_feeds_public"`
	CreaterOpenID string          `json:"creater_openid"`
	FeedsImg      string          `json:"feeds_img"`
}

// LiveReplayClip is one replay clip of a room.
type LiveReplayClip struct {
	CreateTime string `json:"create_time"`
	ExpireTime string `json:"expire_time"`
	MediaURL   string `json:"media_url"`
}

// GetLiveRoomListResponse is the reply of /wxa/business/getliveinfo.
type GetLiveRoomListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	RoomInfo   []LiveRoomInfo   `json:"room_info"`
	Total      int              `json:"total"`
	LiveReplay []LiveReplayClip `json:"live_replay"`
}

// OK reports whether WeChat answered with a usable list.
//
// An empty list is a valid answer, so only the errcode is checked.
func (r *GetLiveRoomListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveRoomListErrMsg returns the description of a getLiveInfo error.
func GetLiveRoomListErrMsg(code int) string {
	if msg, ok := GetLiveRoomListErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveRoomListDefaultErrMsg
}

// GetLiveRoomListError turns the errcode from the reply into an error.
func GetLiveRoomListError(r *GetLiveRoomListResponse) error {
	return errors.New(GetLiveRoomListErrMsg(r.ErrCode))
}

// GetLiveRoomList lists the rooms of the account, newest first.
//
// With Action set to LiveRoomActionGetReplay it returns the replay clips of
// RoomID instead of rooms; a clip URL expires at ExpireTime, and the source
// video is usually ready within ten minutes of the stream ending.
func (a *Application) GetLiveRoomList(
	ctx context.Context,
	params GetLiveRoomListParams,
) (*GetLiveRoomListResponse, error) {
	if err := CheckGetLiveRoomListParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getLiveRoomListPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result GetLiveRoomListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
