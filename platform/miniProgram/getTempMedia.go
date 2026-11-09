package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getTempMediaPath downloads a media file used by a customer service message.
const (
	getTempMediaPath          = "/cgi-bin/media/get"
	GetTempMediaDefaultErrMsg = "系统错误，请稍后再试"
)

// GetTempMediaErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/media/get, keyed by the errcode as it appears in the reply.
//
// The endpoint answers with the file itself on success, so an errcode only
// appears when the media cannot be served at all.
var GetTempMediaErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40007: "media_id 不存在或已过期",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
}

// GetTempMediaParams identifies the media to download.
type GetTempMediaParams struct {
	// MediaID is the id returned by UploadTempMedia.
	MediaID string
}

// CheckGetTempMediaParams validates params before anything is sent to WeChat.
func CheckGetTempMediaParams(params GetTempMediaParams) error {
	if params.MediaID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: media_id is required",
		)
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetTempMediaParams) query() url.Values {
	return url.Values{"media_id": {p.MediaID}}
}

// GetTempMediaResponse is the reply of /cgi-bin/media/get.
//
// The endpoint answers with the file itself rather than with JSON, so the body
// is kept as bytes and the errcode is only filled when WeChat refuses the
// request instead.
type GetTempMediaResponse struct {
	// Common reply parameters, set only when the call failed.
	response.WechatResponse

	// Business reply parameters.
	ContentType string
	Data        []byte
}

// OK reports whether the reply carries a file.
func (r *GetTempMediaResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.Data) > 0
}

// GetTempMediaErrMsg returns the description of a getTempMedia error.
func GetTempMediaErrMsg(code int) string {
	if msg, ok := GetTempMediaErrCodeMap[code]; ok {
		return msg
	}
	return GetTempMediaDefaultErrMsg
}

// GetTempMediaError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetTempMediaErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetTempMediaError(r *GetTempMediaResponse) error {
	return errors.New(GetTempMediaErrMsg(r.ErrCode))
}

// GetTempMedia downloads a media file from a customer service message.
//
// It reads the opposite direction of UploadTempMedia: the file a user sent to
// the customer service, or one that is about to be forwarded. There is no
// Expires header to rely on: a temporary media id lives three days, after which
// WeChat answers 40007 and the file is gone.
//
// A successful reply is the raw file, so the body is only decoded as JSON to
// detect the error case; ContentType is what WeChat reported for the file.
func (a *Application) GetTempMedia(
	ctx context.Context,
	params GetTempMediaParams,
) (*GetTempMediaResponse, error) {
	if err := CheckGetTempMediaParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getTempMediaPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetTempMediaResponse
	if err := resp.DecodeJSON(&result.WechatResponse); err == nil && result.ErrCode != 0 {
		return &result, nil
	}

	result.ContentType = resp.Header.Get("Content-Type")
	result.Data = resp.Body

	return &result, nil
}
