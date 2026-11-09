package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteLiveRoomGoodsPath removes goods from the shelf of a room.
const (
	deleteLiveRoomGoodsPath          = "/wxaapi/broadcast/goods/deleteInRoom"
	DeleteLiveRoomGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// DeleteLiveRoomGoodsErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/goods/deleteInRoom, keyed by the errcode as it appears
// in the reply.
var DeleteLiveRoomGoodsErrCodeMap = liveErrCodes(
	300022,
	300023,
	300024,
	300053,
)

// DeleteLiveRoomGoodsParams removes one goods from a room.
type DeleteLiveRoomGoodsParams struct {
	// RoomID is the room holding the goods.
	RoomID int64 `json:"roomId"`

	// GoodsID is the goods to remove from the room. The goods itself stays in
	// the goods library.
	GoodsID int64 `json:"goodsId"`
}

// CheckDeleteLiveRoomGoodsParams validates params before anything is sent to
// WeChat.
func CheckDeleteLiveRoomGoodsParams(params DeleteLiveRoomGoodsParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.GoodsID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	}

	return nil
}

// DeleteLiveRoomGoodsResponse is the reply of
// /wxaapi/broadcast/goods/deleteInRoom.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteLiveRoomGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat removed the goods from the room.
func (r *DeleteLiveRoomGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// DeleteLiveRoomGoodsErrMsg returns the description of a deleteInRoom error.
func DeleteLiveRoomGoodsErrMsg(code int) string {
	if msg, ok := DeleteLiveRoomGoodsErrCodeMap[code]; ok {
		return msg
	}
	return DeleteLiveRoomGoodsDefaultErrMsg
}

// DeleteLiveRoomGoodsError turns the errcode from the reply into an error.
func DeleteLiveRoomGoodsError(r *DeleteLiveRoomGoodsResponse) error {
	return errors.New(DeleteLiveRoomGoodsErrMsg(r.ErrCode))
}

// DeleteLiveRoomGoods removes goods from the shelf of a room.
//
// Only the room loses the goods; the goods library keeps it, so it can be
// imported into another room. A goods that is being pushed cannot be removed
// until the push finishes.
func (a *Application) DeleteLiveRoomGoods(
	ctx context.Context,
	params DeleteLiveRoomGoodsParams,
) (*DeleteLiveRoomGoodsResponse, error) {
	if err := CheckDeleteLiveRoomGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, deleteLiveRoomGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result DeleteLiveRoomGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
