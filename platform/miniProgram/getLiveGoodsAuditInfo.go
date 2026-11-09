package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveGoodsAuditInfoPath reports the audit state of goods.
const (
	getLiveGoodsAuditInfoPath          = "/wxa/business/getgoodswarehouse"
	GetLiveGoodsAuditInfoDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveGoodsAuditInfoErrCodeMap holds the errcode descriptions WeChat
// documents for /wxa/business/getgoodswarehouse, keyed by the errcode as it
// appears in the reply.
var GetLiveGoodsAuditInfoErrCodeMap = liveErrCodes(
	200002,
	40001,
)

// GetLiveGoodsAuditInfoParams names the goods whose audit state is wanted.
type GetLiveGoodsAuditInfoParams struct {
	// GoodsIDs are the ids returned by AddLiveGoods, at most 100 per call.
	GoodsIDs []int64 `json:"goods_ids"`
}

// CheckGetLiveGoodsAuditInfoParams validates params before anything is sent to
// WeChat.
func CheckGetLiveGoodsAuditInfoParams(params GetLiveGoodsAuditInfoParams) error {
	if len(params.GoodsIDs) == 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goods_ids is required")
	}

	for _, id := range params.GoodsIDs {
		if id <= 0 {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: goods_ids must not contain a non-positive id",
			)
		}
	}

	return nil
}

// LiveGoodsAuditItem is one goods with its audit result.
type LiveGoodsAuditItem struct {
	GoodsID         int64   `json:"goodsId"`
	CoverImgURL     string  `json:"coverImgUrl"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	PriceType       int     `json:"priceType"`
	Price2          float64 `json:"price2"`
	URL             string  `json:"url"`
	ThirdPartyAppID string  `json:"thirdPartyAppid"`

	// AuditStatus is LiveGoodsAuditPending, Reviewing, Approved or Rejected.
	AuditStatus int `json:"auditStatus"`

	// AuditMsg explains a rejection.
	AuditMsg string `json:"auditMsg"`
}

// GetLiveGoodsAuditInfoResponse is the reply of
// /wxa/business/getgoodswarehouse.
type GetLiveGoodsAuditInfoResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Goods []LiveGoodsAuditItem `json:"goods"`
	Total int                  `json:"total"`
}

// OK reports whether WeChat answered with a usable list.
func (r *GetLiveGoodsAuditInfoResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveGoodsAuditInfoErrMsg returns the description of a getgoodsWarehouse
// error.
func GetLiveGoodsAuditInfoErrMsg(code int) string {
	if msg, ok := GetLiveGoodsAuditInfoErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveGoodsAuditInfoDefaultErrMsg
}

// GetLiveGoodsAuditInfoError turns the errcode from the reply into an error.
func GetLiveGoodsAuditInfoError(r *GetLiveGoodsAuditInfoResponse) error {
	return errors.New(GetLiveGoodsAuditInfoErrMsg(r.ErrCode))
}

// GetLiveGoodsAuditInfo reports the audit state of goods.
//
// This is how a caller learns whether goods may be imported into a room: only
// approved goods may, and a rejected one carries the reason in AuditMsg.
func (a *Application) GetLiveGoodsAuditInfo(
	ctx context.Context,
	params GetLiveGoodsAuditInfoParams,
) (*GetLiveGoodsAuditInfoResponse, error) {
	if err := CheckGetLiveGoodsAuditInfoParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getLiveGoodsAuditInfoPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result GetLiveGoodsAuditInfoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
