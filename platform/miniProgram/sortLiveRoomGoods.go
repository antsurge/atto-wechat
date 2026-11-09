package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// sortLiveRoomGoodsPath reorders the shelf of a room.
const (
	sortLiveRoomGoodsPath          = "/wxaapi/broadcast/goods/sort"
	SortLiveRoomGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// SortLiveRoomGoodsErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/goods/sort, keyed by the errcode as it appears in the
// reply.
var SortLiveRoomGoodsErrCodeMap = liveErrCodes(
	300022,
	300023,
	300024,
	300053,
)

// LiveGoodsOrder names one goods in a new shelf order.
type LiveGoodsOrder struct {
	// GoodsID is the goods being placed.
	GoodsID int64 `json:"goodsId"`
}

// SortLiveRoomGoodsParams reorders the goods of a room.
type SortLiveRoomGoodsParams struct {
	// RoomID is the room whose shelf is reordered.
	RoomID int64 `json:"roomId"`

	// Goods is the new order, first item first. It has to list exactly the
	// goods on the shelf of the room.
	Goods []LiveGoodsOrder `json:"goods"`
}

// CheckSortLiveRoomGoodsParams validates params before anything is sent to
// WeChat.
func CheckSortLiveRoomGoodsParams(params SortLiveRoomGoodsParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case len(params.Goods) == 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goods is required")
	}

	for _, goods := range params.Goods {
		if goods.GoodsID <= 0 {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: goodsId is required",
			)
		}
	}

	return nil
}

// SortLiveRoomGoodsResponse is the reply of /wxaapi/broadcast/goods/sort.
//
// It carries no business payload: success is reported by errcode alone.
type SortLiveRoomGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the new order.
func (r *SortLiveRoomGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// SortLiveRoomGoodsErrMsg returns the description of a sort error.
func SortLiveRoomGoodsErrMsg(code int) string {
	if msg, ok := SortLiveRoomGoodsErrCodeMap[code]; ok {
		return msg
	}
	return SortLiveRoomGoodsDefaultErrMsg
}

// SortLiveRoomGoodsError turns the errcode from the reply into an error.
func SortLiveRoomGoodsError(r *SortLiveRoomGoodsResponse) error {
	return errors.New(SortLiveRoomGoodsErrMsg(r.ErrCode))
}

// SortLiveRoomGoods reorders the goods on the shelf of a room.
//
// The order is the one the viewers see, so this is how a goods is brought to
// the front of the shelf without pushing it.
func (a *Application) SortLiveRoomGoods(
	ctx context.Context,
	params SortLiveRoomGoodsParams,
) (*SortLiveRoomGoodsResponse, error) {
	if err := CheckSortLiveRoomGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, sortLiveRoomGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result SortLiveRoomGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
