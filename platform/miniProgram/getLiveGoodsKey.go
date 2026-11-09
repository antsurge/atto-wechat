package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveGoodsKeyPath reads the vendor goods key of the live room.
const (
	getLiveGoodsKeyPath          = "/wxaapi/broadcast/goods/getkey"
	GetLiveGoodsKeyDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveGoodsKeyErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/goods/getkey, keyed by the errcode as it appears in the
// reply.
var GetLiveGoodsKeyErrCodeMap = liveErrCodes(
	200002,
	40001,
)

// GetLiveGoodsKeyResponse is the reply of /wxaapi/broadcast/goods/getkey.
type GetLiveGoodsKeyResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	VendorGoodsKey []string `json:"vendorGoodsKey"`
}

// OK reports whether WeChat answered with the key.
func (r *GetLiveGoodsKeyResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveGoodsKeyErrMsg returns the description of a getkey error.
func GetLiveGoodsKeyErrMsg(code int) string {
	if msg, ok := GetLiveGoodsKeyErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveGoodsKeyDefaultErrMsg
}

// GetLiveGoodsKeyError turns the errcode from the reply into an error.
func GetLiveGoodsKeyError(r *GetLiveGoodsKeyResponse) error {
	return errors.New(GetLiveGoodsKeyErrMsg(r.ErrCode))
}

// GetLiveGoodsKey reads the vendor goods key of the 直播挂件.
//
// An empty list with errcode 0 means no key was set, which is why OK still
// reports success: the caller can compare against an empty slice to tell "not
// configured" from "configured".
func (a *Application) GetLiveGoodsKey(
	ctx context.Context,
) (*GetLiveGoodsKeyResponse, error) {
	resp, err := a.api.Get(ctx, getLiveGoodsKeyPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetLiveGoodsKeyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
