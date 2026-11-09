package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveGoodsListPath lists the approved goods of the library.
const (
	getLiveGoodsListPath          = "/wxaapi/broadcast/goods/getapproved"
	GetLiveGoodsListDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveGoodsListErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/goods/getapproved, keyed by the errcode as it appears in the
// reply.
var GetLiveGoodsListErrCodeMap = liveErrCodes(
	-1,
	200002,
	40001,
)

// GetLiveGoodsListParams selects the goods to return.
type GetLiveGoodsListParams struct {
	// Offset is the number of goods to skip, from 0.
	Offset int

	// Limit is the page size, at most 100.
	Limit int

	// Status filters by audit state: LiveGoodsAuditPending, Reviewing, Approved
	// or Rejected. It is required.
	Status int
}

// CheckGetLiveGoodsListParams validates params before anything is sent to
// WeChat.
func CheckGetLiveGoodsListParams(params GetLiveGoodsListParams) error {
	switch {
	case params.Offset < 0:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: offset must not be negative",
		)
	case params.Limit <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: limit is required")
	}

	switch params.Status {
	case LiveGoodsAuditPending,
		LiveGoodsAuditReviewing,
		LiveGoodsAuditApproved,
		LiveGoodsAuditRejected:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: status must be a LiveGoodsAudit constant",
		)
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetLiveGoodsListParams) query() url.Values {
	return url.Values{
		"offset": {strconv.Itoa(p.Offset)},
		"limit":  {strconv.Itoa(p.Limit)},
		"status": {strconv.Itoa(p.Status)},
	}
}

// LiveApprovedGoods is one goods of the library.
type LiveApprovedGoods struct {
	GoodsID         int64   `json:"goodsId"`
	CoverImgURL     string  `json:"coverImgUrl"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	PriceType       int     `json:"priceType"`
	Price2          float64 `json:"price2"`
	URL             string  `json:"url"`
	ThirdPartyAppID string  `json:"thirdPartyAppid"`
}

// GetLiveGoodsListResponse is the reply of
// /wxaapi/broadcast/goods/getapproved.
type GetLiveGoodsListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Goods []LiveApprovedGoods `json:"goods"`
	Total int                 `json:"total"`
}

// OK reports whether WeChat answered with a usable list.
func (r *GetLiveGoodsListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveGoodsListErrMsg returns the description of a getapproved error.
func GetLiveGoodsListErrMsg(code int) string {
	if msg, ok := GetLiveGoodsListErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveGoodsListDefaultErrMsg
}

// GetLiveGoodsListError turns the errcode from the reply into an error.
func GetLiveGoodsListError(r *GetLiveGoodsListResponse) error {
	return errors.New(GetLiveGoodsListErrMsg(r.ErrCode))
}

// GetLiveGoodsList lists the goods of the library, filtered by audit state.
//
// This is the goods library, not the shelf of a room: what a room holds is
// reported by GetLiveRoomList.
func (a *Application) GetLiveGoodsList(
	ctx context.Context,
	params GetLiveGoodsListParams,
) (*GetLiveGoodsListResponse, error) {
	if err := CheckGetLiveGoodsListParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getLiveGoodsListPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetLiveGoodsListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
