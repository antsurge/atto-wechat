package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// createQRCodePath issues the plain mini program QR code. It is the oldest of
// the three code endpoints and the most limited one: the codes are permanent
// but share a total quota of 100,000 with /wxa/getwxacode.
//
// Note the path differs from the other two endpoints: it lives under
// /cgi-bin/wxaapp instead of /wxa.
const (
	createQRCodePath          = "/cgi-bin/wxaapp/createwxaqrcode"
	CreateQRCodeDefaultErrMsg = "系统错误，请稍后再试"
)

// CreateQRCodeErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/wxaapp/createwxaqrcode, keyed by the errcode as it appears in the
// reply.
//
// 40159 and 85096 are not listed on this endpoint's own page, but the path
// checks behind them are shared with /wxa/getwxacode, so the same errcodes can
// come back here and are described accordingly.
var CreateQRCodeErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40097: "参数错误",
	40159: "path 不能为空、长度超限，或请求体不是 JSON 字符串",
	41030: "path 对应的页面不存在，或小程序尚未发布",
	45009: "API 调用太频繁，请稍候再试",
	45029: "生成的小程序码数量已达上限",
	85096: "scancode_time 为系统保留参数，不允许配置",
}

// CreateQRCodeParams describes one mini program QR code.
//
// Only Path is required. Width leaves its zero value to mean the documented
// default of 430 px.
type CreateQRCodeParams struct {
	// Path is the page the code opens, for example "pages/index/index". It
	// must not be empty and is limited to 128 bytes, which is stricter than
	// the 1024-byte limit of /wxa/getwxacode.
	//
	// For mini games the query part alone may be passed, for example
	// "?foo=bar", and it then shows up in wx.getLaunchOptionsSync().query.
	Path string

	// Width is the side length in pixels, 280 to 1280. Zero means 430.
	Width int
}

// CheckCreateQRCodeParams validates params before anything is sent to WeChat.
//
// The checks mirror what WeChat rejects anyway, so a bad call fails locally
// with a clear reason instead of as an opaque errcode. The 128-byte limit is
// the one this endpoint documents itself, unlike the 1024-byte limit of
// /wxa/getwxacode.
func CheckCreateQRCodeParams(params CreateQRCodeParams) error {
	if params.Path == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: path must not be empty",
		)
	}

	if len([]byte(params.Path)) > 128 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: path must not exceed 128 bytes",
		)
	}

	if params.Width != 0 && (params.Width < 280 || params.Width > 1280) {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: width must be between 280 and 1280",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p CreateQRCodeParams) body() *createQRCodeBody {
	return &createQRCodeBody{
		Path:  p.Path,
		Width: p.Width,
	}
}

// createQRCodeBody is the JSON body of /cgi-bin/wxaapp/createwxaqrcode.
//
// The endpoint takes no styling options: unlike /wxa/getwxacode it always
// renders a black QR code on a white background.
type createQRCodeBody struct {
	Path  string `json:"path"`
	Width int    `json:"width,omitempty"`
}

// CreateQRCodeResponse is the reply of /cgi-bin/wxaapp/createwxaqrcode.
//
// The endpoint answers with two different media types: a successful call
// returns the image bytes, a failed one returns JSON with errcode/errmsg. The
// structure models both, so callers can always inspect ErrCode before using
// Image.
type CreateQRCodeResponse struct {
	// Common reply parameters. They are filled in only when WeChat returned
	// JSON instead of an image.
	response.WechatResponse

	// Business reply parameters. Image is the raw image bytes; this endpoint
	// returns JPEG.
	Image       []byte
	ContentType string
}

// OK reports whether the response carries image bytes.
func (r *CreateQRCodeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.Image) > 0
}

// CreateQRCodeErrMsg returns the description of a createQRCode error.
func CreateQRCodeErrMsg(code int) string {
	if msg, ok := CreateQRCodeErrCodeMap[code]; ok {
		return msg
	}
	return CreateQRCodeDefaultErrMsg
}

// CreateQRCodeError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// CreateQRCodeErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func CreateQRCodeError(r *CreateQRCodeResponse) error {
	return errors.New(CreateQRCodeErrMsg(r.ErrCode))
}

// CreateQRCode returns a mini program QR code that opens params.Path.
//
// Use this endpoint only when a plain QR code is enough and the number of codes
// is small: the quota is shared with /wxa/getwxacode and cannot be reset. When
// codes are created on the fly for many users, use GetUnlimitedQRCode; when the
// look has to be customised, use GetQRCode.
//
// The mini program must be released: an unreleased page makes WeChat answer
// 41030.
//
// WeChat answers a failure with JSON rather than an image, so the reply type is
// decided by Content-Type, exactly like the other code endpoints.
func (a *Application) CreateQRCode(
	ctx context.Context,
	params CreateQRCodeParams,
) (*CreateQRCodeResponse, error) {
	if err := CheckCreateQRCodeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, createQRCodePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	image, contentType, errResp := decodeQRCodeImageReply(resp)
	if errResp.ErrCode != 0 {
		return &CreateQRCodeResponse{WechatResponse: errResp, ContentType: contentType}, nil
	}

	return &CreateQRCodeResponse{Image: image, ContentType: contentType}, nil
}
