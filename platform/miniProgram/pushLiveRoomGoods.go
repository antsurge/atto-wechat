package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// pushLiveRoomGoodsPath pushes goods to the viewers of a room.
const (
	pushLiveRoomGoodsPath          = "/wxaapi/broadcast/goods/push"
	PushLiveRoomGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// PushLiveRoomGoodsErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/goods/push, keyed by the errcode as it appears in the
// reply.
var PushLiveRoomGoodsErrCodeMap = liveErrCodes(
	300022,
	300023,
	300024,
	300047,
	300052,
	300053,
	9410006,
)

// PushLiveRoomGoodsParams pushes one goods to the viewers of a room.
type PushLiveRoomGoodsParams struct {
	// RoomID is the room to push in.
	RoomID int64 `json:"roomId"`

	// GoodsID is the goods to push. It has to be on the shelf of the room and
	// on sale.
	GoodsID int64 `json:"goodsId"`
}

// CheckPushLiveRoomGoodsParams validates params before anything is sent to
// WeChat.
func CheckPushLiveRoomGoodsParams(params PushLiveRoomGoodsParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.GoodsID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	}

	return nil
}

// PushLiveRoomGoodsResponse is the reply of /wxaapi/broadcast/goods/push.
//
// It carries no business payload: success is reported by errcode alone.
type PushLiveRoomGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat pushed the goods.
func (r *PushLiveRoomGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// PushLiveRoomGoodsErrMsg returns the description of a push error.
func PushLiveRoomGoodsErrMsg(code int) string {
	if msg, ok := PushLiveRoomGoodsErrCodeMap[code]; ok {
		return msg
	}
	return PushLiveRoomGoodsDefaultErrMsg
}

// PushLiveRoomGoodsError turns the errcode from the reply into an error.
func PushLiveRoomGoodsError(r *PushLiveRoomGoodsResponse) error {
	return errors.New(PushLiveRoomGoodsErrMsg(r.ErrCode))
}

// PushLiveRoomGoods pushes goods to the viewers of a room.
//
// The push is the "讲解" moment: the goods card pops up in the room and the
// viewers can open it. Only one push can be in flight per room, and the room has
// to be live, so a second call returns 300047 until the first one settles.
func (a *Application) PushLiveRoomGoods(
	ctx context.Context,
	params PushLiveRoomGoodsParams,
) (*PushLiveRoomGoodsResponse, error) {
	if err := CheckPushLiveRoomGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, pushLiveRoomGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result PushLiveRoomGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
