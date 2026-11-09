package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// downloadLiveGoodsVideoPath returns the explanation video of a pushed goods.
const (
	downloadLiveGoodsVideoPath          = "/wxaapi/broadcast/goods/getVideo"
	DownloadLiveGoodsVideoDefaultErrMsg = "系统错误，请稍后再试"
)

// DownloadLiveGoodsVideoErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/goods/getVideo, keyed by the errcode as it
// appears in the reply.
var DownloadLiveGoodsVideoErrCodeMap = liveErrCodes(
	-1,
	300022,
	300024,
	300044,
	300045,
)

// DownloadLiveGoodsVideoParams identifies the explanation video to download.
type DownloadLiveGoodsVideoParams struct {
	// RoomID is the room the goods was pushed in.
	RoomID int64 `json:"roomId"`

	// GoodsID is the goods whose explanation video is wanted.
	GoodsID int64 `json:"goodsId"`
}

// CheckDownloadLiveGoodsVideoParams validates params before anything is sent to
// WeChat.
func CheckDownloadLiveGoodsVideoParams(params DownloadLiveGoodsVideoParams) error {
	switch {
	case params.RoomID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	case params.GoodsID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	}

	return nil
}

// DownloadLiveGoodsVideoResponse is the reply of
// /wxaapi/broadcast/goods/getVideo.
type DownloadLiveGoodsVideoResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	URL string `json:"url"`
}

// OK reports whether WeChat returned a video address.
func (r *DownloadLiveGoodsVideoResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.URL != ""
}

// DownloadLiveGoodsVideoErrMsg returns the description of a getVideo error.
func DownloadLiveGoodsVideoErrMsg(code int) string {
	if msg, ok := DownloadLiveGoodsVideoErrCodeMap[code]; ok {
		return msg
	}
	return DownloadLiveGoodsVideoDefaultErrMsg
}

// DownloadLiveGoodsVideoError turns the errcode from the reply into an error.
func DownloadLiveGoodsVideoError(r *DownloadLiveGoodsVideoResponse) error {
	return errors.New(DownloadLiveGoodsVideoErrMsg(r.ErrCode))
}

// DownloadLiveGoodsVideo returns the explanation video of a goods pushed in a
// room.
//
// WeChat records the video from the moment the goods is pushed, and it takes a
// while to produce, so a goods just pushed answers 300045 until the video is
// ready. A goods that was never pushed has none at all (300044).
func (a *Application) DownloadLiveGoodsVideo(
	ctx context.Context,
	params DownloadLiveGoodsVideoParams,
) (*DownloadLiveGoodsVideoResponse, error) {
	if err := CheckDownloadLiveGoodsVideoParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, downloadLiveGoodsVideoPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result DownloadLiveGoodsVideoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
