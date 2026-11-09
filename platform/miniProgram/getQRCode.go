package miniProgram

import (
	"context"
	"errors"
	"strings"

	"github.com/antsurge/atto-wechat/core/client"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getQRCodePath issues the limited mini program code: the codes are permanent
// but share a total quota of 100,000 with /wxa/createwxaqrcode.
const (
	getQRCodePath          = "/wxa/getwxacode"
	GetQRCodeDefaultErrMsg = "系统错误，请稍后再试"
)

// GetQRCodeErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/getwxacode, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GetQRCodeErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40097: "参数错误",
	40159: "path 不能为空、长度超限，或请求体不是 JSON 字符串",
	41030: "path 对应的页面不存在，或小程序尚未发布",
	45009: "API 调用太频繁，请稍候再试",
	45029: "生成的小程序码数量已达上限",
	85096: "scancode_time 为系统保留参数，不允许配置",
}

// RGBColor is the line colour of a mini program code, with decimal components.
type RGBColor struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// GetQRCodeParams describes one mini program code.
//
// Only Path is required. Width, AutoColor, LineColor, IsHyaline and EnvVersion
// leave their zero value to mean "use the WeChat default", so an empty struct
// field never overrides the documented default.
type GetQRCodeParams struct {
	// Path is the page the code opens, for example "pages/index/index". It
	// must exist in the released mini program and must not start with "/".
	Path string

	// Width is the side length in pixels, 280 to 1280. Zero means 430.
	Width int

	// AutoColor asks WeChat to pick the line colour from the page. When it is
	// false, LineColor applies.
	AutoColor bool

	// LineColor is the line colour, black by default.
	LineColor *RGBColor

	// IsHyaline asks for a transparent background; the reply is then a PNG.
	IsHyaline bool

	// EnvVersion selects the mini program version: release, trial or develop.
	// Empty means release.
	EnvVersion string
}

// CheckQRCodeParams validates params before anything is sent to WeChat.
//
// The checks mirror what WeChat rejects anyway, so a bad call fails locally
// with a clear reason instead of as an opaque 40097. The path limit follows
// errcode 40159 (1024 bytes); the older 128-byte figure is left to WeChat,
// because rejecting a path that the endpoint now accepts would be wrong.
func CheckQRCodeParams(params GetQRCodeParams) error {
	if params.Path == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: path must not be empty",
		)
	}

	if len([]byte(params.Path)) > 1024 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: path must not exceed 1024 bytes",
		)
	}

	if params.Width != 0 && (params.Width < 280 || params.Width > 1280) {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: width must be between 280 and 1280",
		)
	}

	switch params.EnvVersion {
	case "", "release", "trial", "develop":
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: env_version must be release, trial or develop",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetQRCodeParams) body() *getQRCodeBody {
	return &getQRCodeBody{
		Path:       p.Path,
		Width:      p.Width,
		AutoColor:  p.AutoColor,
		LineColor:  p.LineColor,
		IsHyaline:  p.IsHyaline,
		EnvVersion: p.EnvVersion,
	}
}

// getQRCodeBody is the JSON body of /wxa/getwxacode.
type getQRCodeBody struct {
	Path       string    `json:"path"`
	Width      int       `json:"width,omitempty"`
	AutoColor  bool      `json:"auto_color,omitempty"`
	LineColor  *RGBColor `json:"line_color,omitempty"`
	IsHyaline  bool      `json:"is_hyaline,omitempty"`
	EnvVersion string    `json:"env_version,omitempty"`
}

// GetQRCodeResponse is the reply of /wxa/getwxacode.
//
// The endpoint answers with two different media types: a successful call
// returns the image bytes, a failed one returns JSON with errcode/errmsg. The
// structure models both, so callers can always inspect ErrCode before using
// Image.
type GetQRCodeResponse struct {
	// Common reply parameters. They are filled in only when WeChat returned
	// JSON instead of an image.
	response.WechatResponse

	// Business reply parameters. Image is the raw image bytes: JPEG normally,
	// PNG when IsHyaline was requested.
	Image       []byte
	ContentType string
}

// OK reports whether the response carries image bytes.
func (r *GetQRCodeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.Image) > 0
}

// GetQRCodeErrMsg returns the description of a getQRCode error.
func GetQRCodeErrMsg(code int) string {
	if msg, ok := GetQRCodeErrCodeMap[code]; ok {
		return msg
	}
	return GetQRCodeDefaultErrMsg
}

// GetQRCodeError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetQRCodeErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetQRCodeError(r *GetQRCodeResponse) error {
	return errors.New(GetQRCodeErrMsg(r.ErrCode))
}

// GetQRCode returns a permanent mini program code that opens params.Path.
//
// The endpoint is the limited variant: it fits a small number of codes and the
// total budget with /wxa/createwxaqrcode is 100,000. For codes created on the
// fly for many users, /wxa/getwxacodeunlimit is the right endpoint instead.
//
// WeChat answers a failure with JSON rather than an image, so the reply type is
// decided by Content-Type: an image/* reply is the code itself, anything else is
// parsed as an errcode. A body that is neither image nor JSON is still handed
// back as Image, because a successful call must not be reported as a failure.
//
// The path must belong to a released page; otherwise WeChat answers 41030.
func (a *Application) GetQRCode(
	ctx context.Context,
	params GetQRCodeParams,
) (*GetQRCodeResponse, error) {
	if err := CheckQRCodeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getQRCodePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	image, contentType, errResp := decodeImageReply(resp)
	if errResp.ErrCode != 0 {
		return &GetQRCodeResponse{WechatResponse: errResp, ContentType: contentType}, nil
	}

	return &GetQRCodeResponse{Image: image, ContentType: contentType}, nil
}

// decodeImageReply splits the two media types the image endpoints reply with:
// image bytes on success, JSON with errcode/errmsg on failure.
//
// It serves the QR code endpoints and the feedback attachments alike: both
// answer with a body that is either an image or an error document.
//
// A body that is neither image/* nor JSON is returned as image bytes, because
// WeChat does not guarantee a Content-Type and a successful call must not be
// reported as a failure. The errcode of a successful reply is zero, which is
// how callers tell the two outcomes apart.
func decodeImageReply(resp *client.Response) ([]byte, string, response.WechatResponse) {
	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "image/") {
		return resp.Body, contentType, response.WechatResponse{}
	}

	var errResp response.WechatResponse
	if err := resp.DecodeJSON(&errResp); err != nil {
		return resp.Body, contentType, response.WechatResponse{}
	}

	if errResp.ErrCode == 0 {
		return resp.Body, contentType, response.WechatResponse{}
	}

	return nil, contentType, errResp
}
