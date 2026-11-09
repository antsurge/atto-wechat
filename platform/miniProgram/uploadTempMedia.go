package miniProgram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/antsurge/atto-wechat/core/client"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/middleware"
	"github.com/antsurge/atto-wechat/core/response"
)

// uploadTempMediaPath uploads a media file used by a customer service message.
const (
	uploadTempMediaPath          = "/cgi-bin/media/upload"
	UploadTempMediaDefaultErrMsg = "系统错误，请稍后再试"
)

// Media types the endpoint accepts.
const (
	// TempMediaTypeImage is a JPG or PNG image, up to 10 MB.
	TempMediaTypeImage = "image"

	// TempMediaTypeVoice is a voice clip, up to 2 MB and 60 seconds.
	TempMediaTypeVoice = "voice"

	// TempMediaTypeVideo is a video clip, up to 10 MB.
	TempMediaTypeVideo = "video"

	// TempMediaTypeThumb is a thumbnail, up to 64 KB.
	TempMediaTypeThumb = "thumb"
)

// UploadTempMediaErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/media/upload, keyed by the errcode as it appears in the reply.
var UploadTempMediaErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40004: "媒体文件类型不合法",
	40005: "媒体文件大小超出限制",
	40006: "媒体文件格式不合法",
	40007: "媒体文件不存在",
	40009: "图片尺寸超出限制",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
}

// UploadTempMediaParams describes the file to upload.
type UploadTempMediaParams struct {
	// MediaType is one of the TempMediaType* constants.
	MediaType string

	// Filename is the name the file is sent under. WeChat reads the extension
	// from it, so it must match the content, such as "cover.png".
	Filename string

	// Data is the file content.
	Data []byte
}

// CheckUploadTempMediaParams validates params before anything is sent to WeChat.
func CheckUploadTempMediaParams(params UploadTempMediaParams) error {
	switch params.MediaType {
	case TempMediaTypeImage, TempMediaTypeVoice, TempMediaTypeVideo, TempMediaTypeThumb:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: type must be %q, %q, %q or %q",
				TempMediaTypeImage,
				TempMediaTypeVoice,
				TempMediaTypeVideo,
				TempMediaTypeThumb,
			),
		)
	}

	if params.Filename == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: filename is required",
		)
	}

	if len(params.Data) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: data is required",
		)
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p UploadTempMediaParams) query() url.Values {
	return url.Values{"type": {p.MediaType}}
}

// encodeMultipart builds the multipart body of the upload.
//
// The file travels in a part named "media", which is the only field the endpoint
// reads; the file name inside it is what WeChat uses to recognise the format.
func (p UploadTempMediaParams) encodeMultipart() (body []byte, contentType string, err error) {
	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("media", p.Filename)
	if err != nil {
		return nil, "", cerrors.Wrap(
			cerrors.CodeEncodeFailed,
			"miniprogram: encode multipart media",
			err,
		)
	}

	if _, err := part.Write(p.Data); err != nil {
		return nil, "", cerrors.Wrap(
			cerrors.CodeEncodeFailed,
			"miniprogram: write multipart media",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		return nil, "", cerrors.Wrap(
			cerrors.CodeEncodeFailed,
			"miniprogram: close multipart writer",
			err,
		)
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

// UploadTempMediaResponse is the reply of /cgi-bin/media/upload.
type UploadTempMediaResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt int64  `json:"created_at"`
}

// OK reports whether the reply carries a media id.
func (r *UploadTempMediaResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.MediaID != ""
}

// UploadTempMediaErrMsg returns the description of an uploadTempMedia error.
func UploadTempMediaErrMsg(code int) string {
	if msg, ok := UploadTempMediaErrCodeMap[code]; ok {
		return msg
	}
	return UploadTempMediaDefaultErrMsg
}

// UploadTempMediaError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// UploadTempMediaErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func UploadTempMediaError(r *UploadTempMediaResponse) error {
	return errors.New(UploadTempMediaErrMsg(r.ErrCode))
}

// UploadTempMedia uploads a media file for a customer service message.
//
// A temporary media id lives three days, which is longer than the 48-hour reply
// window it is used in but too short to store: keep the bytes and upload again
// when a later conversation needs the same picture.
//
// The endpoint is multipart/form-data, so it bypasses PostJSON and encodes the
// body itself with the file in a part named "media".
func (a *Application) UploadTempMedia(
	ctx context.Context,
	params UploadTempMediaParams,
) (*UploadTempMediaResponse, error) {
	if err := CheckUploadTempMediaParams(params); err != nil {
		return nil, err
	}

	body, contentType, err := params.encodeMultipart()
	if err != nil {
		return nil, err
	}

	resp, err := a.api.Do(ctx, &client.Request{
		Method:   http.MethodPost,
		Path:     uploadTempMediaPath,
		Query:    params.query(),
		Body:     body,
		BodyType: middleware.BodyMultipart,
		Header:   http.Header{"Content-Type": {contentType}},
	})
	if err != nil {
		return nil, err
	}

	var result UploadTempMediaResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
