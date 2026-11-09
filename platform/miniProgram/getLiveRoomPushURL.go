package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveRoomPushURLPath returns the push address of a room.
const (
	getLiveRoomPushURLPath          = "/wxaapi/broadcast/room/getpushurl"
	GetLiveRoomPushURLDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveRoomPushURLErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/room/getpushurl, keyed by the errcode as it appears in
// the reply.
var GetLiveRoomPushURLErrCodeMap = liveErrCodes(
	300022,
	300023,
	40001,
	43001,
)

// GetLiveRoomPushURLParams identifies the room whose push address is wanted.
type GetLiveRoomPushURLParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64
}

// CheckGetLiveRoomPushURLParams validates params before anything is sent to
// WeChat.
func CheckGetLiveRoomPushURLParams(params GetLiveRoomPushURLParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetLiveRoomPushURLParams) query() url.Values {
	return url.Values{"roomId": {strconv.FormatInt(p.RoomID, 10)}}
}

// GetLiveRoomPushURLResponse is the reply of
// /wxaapi/broadcast/room/getpushurl.
type GetLiveRoomPushURLResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	PushAddr string `json:"pushAddr"`
}

// OK reports whether WeChat returned a push address.
func (r *GetLiveRoomPushURLResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.PushAddr != ""
}

// GetLiveRoomPushURLErrMsg returns the description of a getPushURL error.
func GetLiveRoomPushURLErrMsg(code int) string {
	if msg, ok := GetLiveRoomPushURLErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveRoomPushURLDefaultErrMsg
}

// GetLiveRoomPushURLError turns the errcode from the reply into an error.
func GetLiveRoomPushURLError(r *GetLiveRoomPushURLResponse) error {
	return errors.New(GetLiveRoomPushURLErrMsg(r.ErrCode))
}

// GetLiveRoomPushURL returns the address a push-type room streams to.
//
// The address carries the stream key, so it must stay on the server: anyone who
// gets it can broadcast in the room.
func (a *Application) GetLiveRoomPushURL(
	ctx context.Context,
	params GetLiveRoomPushURLParams,
) (*GetLiveRoomPushURLResponse, error) {
	if err := CheckGetLiveRoomPushURLParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getLiveRoomPushURLPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetLiveRoomPushURLResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
