package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveRoomSharedCodePath returns the share code of a room.
const (
	getLiveRoomSharedCodePath          = "/wxaapi/broadcast/room/getsharedcode"
	GetLiveRoomSharedCodeDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveRoomSharedCodeErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/room/getsharedcode, keyed by the errcode as
// it appears in the reply.
var GetLiveRoomSharedCodeErrCodeMap = liveErrCodes(
	1001,
	300022,
	300023,
	40001,
)

// GetLiveRoomSharedCodeParams identifies the room whose share code is wanted.
type GetLiveRoomSharedCodeParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64

	// Params is the custom parameter carried into the room, at most 600 bytes
	// after encoding. It is what maps a visitor back to a share source.
	Params string
}

// CheckGetLiveRoomSharedCodeParams validates params before anything is sent to
// WeChat.
func CheckGetLiveRoomSharedCodeParams(params GetLiveRoomSharedCodeParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetLiveRoomSharedCodeParams) query() url.Values {
	q := url.Values{"roomId": {strconv.FormatInt(p.RoomID, 10)}}
	if p.Params != "" {
		q.Set("params", p.Params)
	}

	return q
}

// GetLiveRoomSharedCodeResponse is the reply of
// /wxaapi/broadcast/room/getsharedcode.
type GetLiveRoomSharedCodeResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	CDNURL    string `json:"cdnUrl"`
	PagePath  string `json:"pagePath"`
	PosterURL string `json:"posterUrl"`
}

// OK reports whether WeChat returned a share code.
func (r *GetLiveRoomSharedCodeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && (r.CDNURL != "" || r.PagePath != "")
}

// GetLiveRoomSharedCodeErrMsg returns the description of a getSharedCode error.
func GetLiveRoomSharedCodeErrMsg(code int) string {
	if msg, ok := GetLiveRoomSharedCodeErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveRoomSharedCodeDefaultErrMsg
}

// GetLiveRoomSharedCodeError turns the errcode from the reply into an error.
func GetLiveRoomSharedCodeError(r *GetLiveRoomSharedCodeResponse) error {
	return errors.New(GetLiveRoomSharedCodeErrMsg(r.ErrCode))
}

// GetLiveRoomSharedCode returns the share code, path and poster of a room.
//
// PagePath already points at the live player plugin with the room id filled in,
// so it can be handed to a mini program code generator or turned into a link
// without further string building.
func (a *Application) GetLiveRoomSharedCode(
	ctx context.Context,
	params GetLiveRoomSharedCodeParams,
) (*GetLiveRoomSharedCodeResponse, error) {
	if err := CheckGetLiveRoomSharedCodeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getLiveRoomSharedCodePath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetLiveRoomSharedCodeResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
