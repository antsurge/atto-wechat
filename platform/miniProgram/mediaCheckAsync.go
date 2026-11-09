package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// mediaCheckAsyncPath submits an image or audio for asynchronous checking.
const mediaCheckAsyncPath = "/wxa/media_check_async"

// MediaCheckVersion is the only version WeChat accepts; 2.0 is the sole spelling
// of the v2 interface.
const MediaCheckVersion = 2

// Values of media_type.
const (
	// MediaCheckTypeAudio is 1.
	MediaCheckTypeAudio = 1
	// MediaCheckTypeImage is 2.
	MediaCheckTypeImage = 2
)

// MediaCheckTypeMap holds the descriptions of the media_type values.
var MediaCheckTypeMap = map[int]string{
	MediaCheckTypeAudio: "音频",
	MediaCheckTypeImage: "图片",
}

// MediaCheckTypeMsg returns the description of a media type, or "" when unknown.
func MediaCheckTypeMsg(mediaType int) string {
	return MediaCheckTypeMap[mediaType]
}

// MediaCheckAsyncErrCodeMap holds the errcode descriptions WeChat documents for
// this endpoint, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var MediaCheckAsyncErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效或不是最新的",
	40003: "openid 无效",
	40004: "media_type 错误（支持 1 音频、2 图片）",
	43104: "appid 与 openid 不匹配",
	44991: "超出接口每分钟调用限制",
	45009: "超出接口每日调用限额",
	61010: "用户访问记录超时（用户需在近两小时访问过小程序）",
	87020: "重复请求超出接口调用限制",
}

// MediaCheckAsyncDefaultErrMsg is returned for an errcode this endpoint does not
// document.
const MediaCheckAsyncDefaultErrMsg = "系统错误，请稍后再试"

// MediaCheckAsyncErrMsg returns the description of a mediaCheckAsync error, or
// the default when the errcode is undocumented.
func MediaCheckAsyncErrMsg(code int) string {
	if msg, ok := MediaCheckAsyncErrCodeMap[code]; ok {
		return msg
	}

	return MediaCheckAsyncDefaultErrMsg
}

// MediaCheckAsyncParams describes the media to check.
//
// The verdict does not come back here: the reply only carries the trace_id of the
// submission, and the result arrives later as a wxa_media_check push (see
// OnMediaCheckResult).
type MediaCheckAsyncParams struct {
	// MediaURL is the media to check, http(s), downloadable by WeChat's checking
	// servers.
	//
	// Images: jpg, jpeg, png, bmp, gif (first frame). Audio: mp3, aac, ac3, wma,
	// flac, vorbis, opus, wav.
	MediaURL string

	// MediaType is one of the MediaCheckType* values, and has to match MediaURL:
	// a mismatch is not detected by WeChat, the download simply fails later and the
	// push reports MediaCheckDownloadFailed.
	MediaType int

	// Scene is one of the MsgSecCheckScene* values.
	Scene int

	// OpenID is the user who submitted the media.
	//
	// WeChat requires the user to have visited the mini program within the last
	// two hours, and answers 61010 otherwise.
	OpenID string
}

// CheckMediaCheckAsyncParams validates params before anything is sent to WeChat.
func CheckMediaCheckAsyncParams(params MediaCheckAsyncParams) error {
	if params.MediaURL == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MediaCheckAsync media_url is required",
		)
	}

	// A local check here is worth more than usual: WeChat accepts the submission and
	// only fails at download time, minutes later, in a push nobody is watching.
	parsed, err := url.Parse(params.MediaURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MediaCheckAsync media_url must be an absolute http(s) URL",
		)
	}

	if _, ok := MediaCheckTypeMap[params.MediaType]; !ok {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MediaCheckAsync media_type must be 1 (audio) or 2 (image)",
		)
	}

	if _, ok := MsgSecCheckSceneMap[params.Scene]; !ok {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MediaCheckAsync scene must be 1-4",
		)
	}

	if params.OpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MediaCheckAsync openid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p MediaCheckAsyncParams) body() *mediaCheckAsyncBody {
	return &mediaCheckAsyncBody{
		MediaURL:  p.MediaURL,
		MediaType: p.MediaType,
		Version:   MediaCheckVersion,
		Scene:     p.Scene,
		OpenID:    p.OpenID,
	}
}

// mediaCheckAsyncBody is the JSON body of /wxa/media_check_async.
type mediaCheckAsyncBody struct {
	MediaURL  string `json:"media_url"`
	MediaType int    `json:"media_type"`
	Version   int    `json:"version"`
	Scene     int    `json:"scene"`
	OpenID    string `json:"openid"`
}

// MediaCheckAsyncResponse is the reply of /wxa/media_check_async.
type MediaCheckAsyncResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// TraceID identifies the submission and is what matches the later push.
	TraceID string `json:"trace_id"`
}

// OK reports whether the media was accepted for checking.
//
// It says nothing about the content: the verdict comes with the push.
func (r *MediaCheckAsyncResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.TraceID != ""
}

// MediaCheckAsyncError turns the errcode from the reply into an error.
func MediaCheckAsyncError(r *MediaCheckAsyncResponse) error {
	return errors.New(MediaCheckAsyncErrMsg(r.ErrCode))
}

// MediaCheckAsync submits an image or audio for asynchronous content checking.
//
// The answer arrives within 30 minutes as a wxa_media_check push; keep the
// TraceID from the reply to match it. Nothing in this call tells whether the media
// is acceptable, so the content has to stay unpublished until the push arrives.
func (a *Application) MediaCheckAsync(
	ctx context.Context,
	params MediaCheckAsyncParams,
) (*MediaCheckAsyncResponse, error) {
	if err := CheckMediaCheckAsyncParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, mediaCheckAsyncPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result MediaCheckAsyncResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
