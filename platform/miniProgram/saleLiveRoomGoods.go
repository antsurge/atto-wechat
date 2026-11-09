package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// saleLiveRoomGoodsPath puts goods on sale, or takes them off sale.
const (
	saleLiveRoomGoodsPath          = "/wxaapi/broadcast/goods/onsale"
	SaleLiveRoomGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// SaleLiveRoomGoodsErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/goods/onsale, keyed by the errcode as it appears in the
// reply.
var SaleLiveRoomGoodsErrCodeMap = liveErrCodes(
	300022,
	300023,
	300024,
	300049,
	300053,
)

// SaleLiveRoomGoodsParams changes the sale state of one goods in a room.
type SaleLiveRoomGoodsParams struct {
	// RoomID is the room holding the goods.
	RoomID int64 `json:"roomId"`

	// GoodsID is the goods whose state changes.
	GoodsID int64 `json:"goodsId"`

	// OnSale is 1 to put the goods on sale and 0 to take it off.
	OnSale int `json:"onSale"`
}

// CheckSaleLiveRoomGoodsParams validates params before anything is sent to
// WeChat.
func CheckSaleLiveRoomGoodsParams(params SaleLiveRoomGoodsParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.GoodsID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	case params.OnSale != 0 && params.OnSale != 1:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: onSale must be 0 or 1",
		)
	}

	return nil
}

// SaleLiveRoomGoodsResponse is the reply of /wxaapi/broadcast/goods/onsale.
//
// It carries no business payload: success is reported by errcode alone.
type SaleLiveRoomGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the sale change.
func (r *SaleLiveRoomGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// SaleLiveRoomGoodsErrMsg returns the description of an onsale error.
func SaleLiveRoomGoodsErrMsg(code int) string {
	if msg, ok := SaleLiveRoomGoodsErrCodeMap[code]; ok {
		return msg
	}
	return SaleLiveRoomGoodsDefaultErrMsg
}

// SaleLiveRoomGoodsError turns the errcode from the reply into an error.
func SaleLiveRoomGoodsError(r *SaleLiveRoomGoodsResponse) error {
	return errors.New(SaleLiveRoomGoodsErrMsg(r.ErrCode))
}

// SaleLiveRoomGoods puts one goods on sale, or takes it off sale, in a room.
//
// The state is per room: taking the goods off sale in one room leaves it on
// sale elsewhere. A goods cannot change state while a push is in flight, and one
// already off sale cannot be pushed again.
func (a *Application) SaleLiveRoomGoods(
	ctx context.Context,
	params SaleLiveRoomGoodsParams,
) (*SaleLiveRoomGoodsResponse, error) {
	if err := CheckSaleLiveRoomGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, saleLiveRoomGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result SaleLiveRoomGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
