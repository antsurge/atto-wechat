package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// updateLiveGoodsPath updates goods in the library.
const (
	updateLiveGoodsPath          = "/wxaapi/broadcast/goods/update"
	UpdateLiveGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// UpdateLiveGoodsErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/goods/update, keyed by the errcode as it appears in the
// reply.
var UpdateLiveGoodsErrCodeMap = liveErrCodes(
	200002,
	300002,
	300003,
	300004,
	300006,
	300007,
	300010,
	300011,
	300017,
	300018,
	300024,
	40001,
)

// UpdateLiveGoodsParams updates one goods of the library.
type UpdateLiveGoodsParams struct {
	// GoodsInfo describes the goods after the update and carries its GoodsID.
	GoodsInfo LiveGoodsInfo `json:"goodsInfo"`
}

// CheckUpdateLiveGoodsParams validates params before anything is sent to WeChat.
func CheckUpdateLiveGoodsParams(params UpdateLiveGoodsParams) error {
	info := params.GoodsInfo

	if info.GoodsID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	}

	switch {
	case info.CoverImgURL == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: coverImgUrl is required")
	case info.Name == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: name is required")
	case info.URL == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: url is required")
	}

	return nil
}

// UpdateLiveGoodsResponse is the reply of /wxaapi/broadcast/goods/update.
//
// It carries no business payload: success is reported by errcode alone.
type UpdateLiveGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat accepted the update.
func (r *UpdateLiveGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdateLiveGoodsErrMsg returns the description of an update goods error.
func UpdateLiveGoodsErrMsg(code int) string {
	if msg, ok := UpdateLiveGoodsErrCodeMap[code]; ok {
		return msg
	}
	return UpdateLiveGoodsDefaultErrMsg
}

// UpdateLiveGoodsError turns the errcode from the reply into an error.
func UpdateLiveGoodsError(r *UpdateLiveGoodsResponse) error {
	return errors.New(UpdateLiveGoodsErrMsg(r.ErrCode))
}

// UpdateLiveGoods updates goods of the library.
//
// What may change depends on the audit state: goods still under review cannot
// be touched (300010), and goods already approved only accept a new price type
// and new prices, so renaming them needs a rejected review first. Goods created
// in the live console cannot be updated at all (300011).
func (a *Application) UpdateLiveGoods(
	ctx context.Context,
	params UpdateLiveGoodsParams,
) (*UpdateLiveGoodsResponse, error) {
	if err := CheckUpdateLiveGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updateLiveGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result UpdateLiveGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
