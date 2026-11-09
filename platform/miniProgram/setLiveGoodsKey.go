package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setLiveGoodsKeyPath stores the vendor goods key of the live room.
const (
	setLiveGoodsKeyPath          = "/wxaapi/broadcast/goods/setkey"
	SetLiveGoodsKeyDefaultErrMsg = "系统错误，请稍后再试"
)

// SetLiveGoodsKeyErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/broadcast/goods/setkey, keyed by the errcode as it appears in the
// reply.
var SetLiveGoodsKeyErrCodeMap = liveErrCodes(
	200002,
	40001,
)

// SetLiveGoodsKeyParams stores the key the 直播挂件 uses to read the goods of
// the mini program.
type SetLiveGoodsKeyParams struct {
	// GoodsKey is the key, at most 32 characters. WeChat keeps it account wide,
	// so setting it again replaces the previous one.
	GoodsKey []string `json:"goodsKey"`
}

// CheckSetLiveGoodsKeyParams validates params before anything is sent to WeChat.
func CheckSetLiveGoodsKeyParams(params SetLiveGoodsKeyParams) error {
	if len(params.GoodsKey) == 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsKey is required")
	}

	for _, key := range params.GoodsKey {
		if key == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: goodsKey must not contain an empty key",
			)
		}
	}

	return nil
}

// SetLiveGoodsKeyResponse is the reply of /wxaapi/broadcast/goods/setkey.
//
// It carries no business payload: success is reported by errcode alone.
type SetLiveGoodsKeyResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat stored the key.
func (r *SetLiveGoodsKeyResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// SetLiveGoodsKeyErrMsg returns the description of a setkey error.
func SetLiveGoodsKeyErrMsg(code int) string {
	if msg, ok := SetLiveGoodsKeyErrCodeMap[code]; ok {
		return msg
	}
	return SetLiveGoodsKeyDefaultErrMsg
}

// SetLiveGoodsKeyError turns the errcode from the reply into an error.
func SetLiveGoodsKeyError(r *SetLiveGoodsKeyResponse) error {
	return errors.New(SetLiveGoodsKeyErrMsg(r.ErrCode))
}

// SetLiveGoodsKey stores the vendor goods key used by the 直播挂件.
//
// The plugin reads the goods list of the mini program from the server of the
// vendor, and this key is what lets that server tell the requests apart. It is
// account wide rather than per room, so one call covers every room.
func (a *Application) SetLiveGoodsKey(
	ctx context.Context,
	params SetLiveGoodsKeyParams,
) (*SetLiveGoodsKeyResponse, error) {
	if err := CheckSetLiveGoodsKeyParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, setLiveGoodsKeyPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result SetLiveGoodsKeyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
