package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addLiveGoodsPath adds goods to the library and submits them for review.
const (
	addLiveGoodsPath          = "/wxaapi/broadcast/goods/add"
	AddLiveGoodsDefaultErrMsg = "系统错误，请稍后再试"
)

// AddLiveGoodsErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/goods/add, keyed by the errcode as it appears in the reply.
var AddLiveGoodsErrCodeMap = liveErrCodes(
	200002,
	300002,
	300003,
	300004,
	300006,
	300007,
	300018,
	300021,
	40001,
)

// AddLiveGoodsParams adds one goods and submits it for review.
type AddLiveGoodsParams struct {
	// GoodsInfo describes the goods.
	GoodsInfo LiveGoodsInfo `json:"goodsInfo"`
}

// CheckAddLiveGoodsParams validates params before anything is sent to WeChat.
func CheckAddLiveGoodsParams(params AddLiveGoodsParams) error {
	info := params.GoodsInfo

	switch {
	case info.CoverImgURL == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: coverImgUrl is required")
	case info.Name == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: name is required")
	case info.URL == "":
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: url is required")
	case info.Price <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: price is required")
	}

	switch info.PriceType {
	case LiveGoodsPriceTypeFixed:
	case LiveGoodsPriceTypeRange, LiveGoodsPriceTypeDiscount:
		if info.Price2 <= 0 {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: price2 is required for a range or a discount",
			)
		}
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: priceType must be LiveGoodsPriceTypeFixed, Range or Discount",
		)
	}

	return nil
}

// AddLiveGoodsResponse is the reply of /wxaapi/broadcast/goods/add.
type AddLiveGoodsResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	GoodsID int64 `json:"goodsId"`
	AuditID int64 `json:"auditId"`
}

// OK reports whether WeChat stored the goods and queued the review.
func (r *AddLiveGoodsResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.GoodsID != 0 && r.AuditID != 0
}

// AddLiveGoodsErrMsg returns the description of an add goods error.
func AddLiveGoodsErrMsg(code int) string {
	if msg, ok := AddLiveGoodsErrCodeMap[code]; ok {
		return msg
	}
	return AddLiveGoodsDefaultErrMsg
}

// AddLiveGoodsError turns the errcode from the reply into an error.
func AddLiveGoodsError(r *AddLiveGoodsResponse) error {
	return errors.New(AddLiveGoodsErrMsg(r.ErrCode))
}

// AddLiveGoods adds goods to the library and submits it for review.
//
// The goods only becomes importable into a room once the review passes, which
// GetLiveGoodsAuditInfo reports. Adding goods with errcode 300021 means WeChat
// stored them but rejected the review, so the returned GoodsID is still usable
// for an UpdateLiveGoods and a resubmission.
func (a *Application) AddLiveGoods(
	ctx context.Context,
	params AddLiveGoodsParams,
) (*AddLiveGoodsResponse, error) {
	if err := CheckAddLiveGoodsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addLiveGoodsPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result AddLiveGoodsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
