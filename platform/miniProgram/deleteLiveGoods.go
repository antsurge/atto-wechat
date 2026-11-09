package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteLiveGoodsPath removes goods from the library.
const (
	deleteLiveGoodsPath          = "/wxaapi/broadcast/goods/delete"
	DeleteLiveGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// DeleteLiveGoodsErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/goods/delete, keyed by the errcode as it appears in the
// reply.
var DeleteLiveGoodsErrCodeMap = liveErrCodes(
	200002,
	300011,
	300024,
	40001,
)

// DeleteLiveGoodsParams removes one goods from the library.
type DeleteLiveGoodsParams struct {
	// GoodsID is the goods to remove.
	GoodsID int64 `json:"goodsId"`
}

// CheckDeleteLiveGoodsParams validates params before anything is sent to WeChat.
func CheckDeleteLiveGoodsParams(params DeleteLiveGoodsParams) error {
	if params.GoodsID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	}

	return nil
}

// DeleteLiveGoodsResponse is the reply of /wxaapi/broadcast/goods/delete.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteLiveGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat removed the goods.
func (r *DeleteLiveGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// DeleteLiveGoodsErrMsg returns the description of a delete goods error.
func DeleteLiveGoodsErrMsg(code int) string {
	if msg, ok := DeleteLiveGoodsErrCodeMap[code]; ok {
		return msg
	}
	return DeleteLiveGoodsDefaultErrMsg
}

// DeleteLiveGoodsError turns the errcode from the reply into an error.
func DeleteLiveGoodsError(r *DeleteLiveGoodsResponse) error {
	return errors.New(DeleteLiveGoodsErrMsg(r.ErrCode))
}

// DeleteLiveGoods removes goods from the library.
//
// The goods also leaves every room that had it on the shelf. Goods created in
// the live console are refused (300011), and the removal cannot be undone.
func (a *Application) DeleteLiveGoods(
	ctx context.Context,
	params DeleteLiveGoodsParams,
) (*DeleteLiveGoodsResponse, error) {
	if err := CheckDeleteLiveGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, deleteLiveGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result DeleteLiveGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
