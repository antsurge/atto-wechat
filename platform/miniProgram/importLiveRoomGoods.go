package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// importLiveRoomGoodsPath puts goods into the shelf of a room.
const (
	importLiveRoomGoodsPath          = "/wxaapi/broadcast/room/addgoods"
	ImportLiveRoomGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// ImportLiveRoomGoodsErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/room/addgoods, keyed by the errcode as it appears in
// the reply.
var ImportLiveRoomGoodsErrCodeMap = liveErrCodes(
	200002,
	300022,
	300023,
	300024,
	300025,
	300027,
	300033,
	40001,
)

// ImportLiveRoomGoodsParams imports approved goods into a room.
type ImportLiveRoomGoodsParams struct {
	// GoodsIDs are the ids returned by AddLiveGoods. Goods added from the live
	// console cannot be imported this way.
	GoodsIDs []int64 `json:"ids"`

	// RoomID is the room receiving the goods.
	RoomID int64 `json:"roomId"`
}

// CheckImportLiveRoomGoodsParams validates params before anything is sent to
// WeChat.
func CheckImportLiveRoomGoodsParams(params ImportLiveRoomGoodsParams) error {
	switch {
	case len(params.GoodsIDs) == 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: ids is required")
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	return nil
}

// ImportLiveRoomGoodsResponse is the reply of
// /wxaapi/broadcast/room/addgoods.
//
// It carries no business payload: success is reported by errcode alone.
type ImportLiveRoomGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat imported the goods.
func (r *ImportLiveRoomGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ImportLiveRoomGoodsErrMsg returns the description of an addGoods error.
func ImportLiveRoomGoodsErrMsg(code int) string {
	if msg, ok := ImportLiveRoomGoodsErrCodeMap[code]; ok {
		return msg
	}
	return ImportLiveRoomGoodsDefaultErrMsg
}

// ImportLiveRoomGoodsError turns the errcode from the reply into an error.
func ImportLiveRoomGoodsError(r *ImportLiveRoomGoodsResponse) error {
	return errors.New(ImportLiveRoomGoodsErrMsg(r.ErrCode))
}

// ImportLiveRoomGoods imports approved goods into a room.
//
// Every id has to be an approved goods created through the API; a goods still
// under review, rejected, or added in the live console is refused, and the room
// caps how many it can hold.
func (a *Application) ImportLiveRoomGoods(
	ctx context.Context,
	params ImportLiveRoomGoodsParams,
) (*ImportLiveRoomGoodsResponse, error) {
	if err := CheckImportLiveRoomGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, importLiveRoomGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ImportLiveRoomGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
