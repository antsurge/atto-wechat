package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addSubAnchorPath adds the deputy anchor of a room.
const (
	addSubAnchorPath          = "/wxaapi/broadcast/room/addsubanchor"
	AddSubAnchorDefaultErrMsg = "系统错误，请稍后再试"
)

// AddSubAnchorErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/room/addsubanchor, keyed by the errcode as it appears in
// the reply.
var AddSubAnchorErrCodeMap = liveErrCodes(
	500004,
	500005,
)

// AddSubAnchorParams sets the deputy anchor of a room.
type AddSubAnchorParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64 `json:"roomId"`

	// Username is the deputy anchor's WeChat id. It cannot repeat the deputy
	// anchor already set, nor the anchor of the room.
	Username string `json:"username"`
}

// CheckAddSubAnchorParams validates params before anything is sent to WeChat.
func CheckAddSubAnchorParams(params AddSubAnchorParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.Username == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: username is required")
	}

	return nil
}

// AddSubAnchorResponse is the reply of /wxaapi/broadcast/room/addsubanchor.
//
// It carries no business payload: success is reported by errcode alone.
type AddSubAnchorResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the deputy anchor.
func (r *AddSubAnchorResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// AddSubAnchorErrMsg returns the description of an addSubAnchor error.
func AddSubAnchorErrMsg(code int) string {
	if msg, ok := AddSubAnchorErrCodeMap[code]; ok {
		return msg
	}
	return AddSubAnchorDefaultErrMsg
}

// AddSubAnchorError turns the errcode from the reply into an error.
func AddSubAnchorError(r *AddSubAnchorResponse) error {
	return errors.New(AddSubAnchorErrMsg(r.ErrCode))
}

// AddSubAnchor sets the deputy anchor of a room.
//
// The account has to have completed real-name verification in the 小程序直播
// mini program, and cannot be the anchor of the same room.
func (a *Application) AddSubAnchor(
	ctx context.Context,
	params AddSubAnchorParams,
) (*AddSubAnchorResponse, error) {
	if err := CheckAddSubAnchorParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addSubAnchorPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result AddSubAnchorResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
