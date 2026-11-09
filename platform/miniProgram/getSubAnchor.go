package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getSubAnchorPath returns the deputy anchor of a room.
const (
	getSubAnchorPath          = "/wxaapi/broadcast/room/getsubanchor"
	GetSubAnchorDefaultErrMsg = "系统错误，请稍后再试"
)

// GetSubAnchorErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/getsubanchor, keyed by the errcode as it appears in
// the reply.
var GetSubAnchorErrCodeMap = liveErrCodes(-1)

// GetSubAnchorParams identifies the room whose deputy anchor is wanted.
type GetSubAnchorParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64
}

// CheckGetSubAnchorParams validates params before anything is sent to WeChat.
func CheckGetSubAnchorParams(params GetSubAnchorParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetSubAnchorParams) query() url.Values {
	return url.Values{"roomId": {strconv.FormatInt(p.RoomID, 10)}}
}

// GetSubAnchorResponse is the reply of /wxaapi/broadcast/room/getsubanchor.
type GetSubAnchorResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Username string `json:"username"`
}

// OK reports whether WeChat returned a deputy anchor.
func (r *GetSubAnchorResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Username != ""
}

// GetSubAnchorErrMsg returns the description of a getSubAnchor error.
func GetSubAnchorErrMsg(code int) string {
	if msg, ok := GetSubAnchorErrCodeMap[code]; ok {
		return msg
	}
	return GetSubAnchorDefaultErrMsg
}

// GetSubAnchorError turns the errcode from the reply into an error.
func GetSubAnchorError(r *GetSubAnchorResponse) error {
	return errors.New(GetSubAnchorErrMsg(r.ErrCode))
}

// GetSubAnchor returns the deputy anchor of a room.
//
// A room has at most one deputy anchor, and an empty username with errcode 0
// means none was set, which OK reports as a failure so the caller does not read
// an empty string as a real account.
func (a *Application) GetSubAnchor(
	ctx context.Context,
	params GetSubAnchorParams,
) (*GetSubAnchorResponse, error) {
	if err := CheckGetSubAnchorParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getSubAnchorPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetSubAnchorResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
