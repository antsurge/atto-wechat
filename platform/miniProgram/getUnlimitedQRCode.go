package miniProgram

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getUnlimitedQRCodePath issues an unlimited mini program code: the codes are
// permanent, there is no total quota, and the endpoint accepts at most 5000
// calls per minute.
const (
	getUnlimitedQRCodePath          = "/wxa/getwxacodeunlimit"
	GetUnlimitedQRCodeDefaultErrMsg = "系统错误，请稍后再试"

	// unlimitedQRCodeSceneMaxLen is the documented limit: 32 visible
	// characters, not bytes.
	unlimitedQRCodeSceneMaxLen = 32

	// unlimitedQRCodeSceneChars are the punctuation marks scene accepts on top
	// of digits and letters. "%" is deliberately absent: WeChat does not
	// decode it, so callers must encode unsupported characters themselves.
	unlimitedQRCodeSceneChars = "!#$&'()*+,/:;=?@-._~"
)

// GetUnlimitedQRCodeErrCodeMap holds the errcode descriptions WeChat documents
// for /wxa/getwxacodeunlimit, keyed by the errcode as it appears in the reply.
var GetUnlimitedQRCodeErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40097: "参数错误",
	40129: "scene 不合法：最大 32 个可见字符，只支持数字、大小写英文及部分特殊字符",
	40169: "scene 长度不合法，或请求体不是 JSON 字符串",
	41030: "page 路径不正确",
	85096: "scancode_time 为系统保留参数，不允许配置",
}

// GetUnlimitedQRCodeParams describes one unlimited mini program code.
//
// Only Scene is required. Width, AutoColor, LineColor and IsHyaline leave
// their zero value to mean "use the WeChat default".
type GetUnlimitedQRCodeParams struct {
	// Scene is the value scanned into the mini program, passed as the query
	// parameter "scene". It is required: unlike GetQRCodeParams.Path, every
	// parameter of the code travels through this field.
	Scene string

	// Page is the page the code opens, for example "pages/index/index". It
	// defaults to the home page, must not start with "/" and must not carry
	// query parameters.
	Page string

	// CheckPath asks WeChat to verify that Page exists in the released mini
	// program. It defaults to true, so nil keeps that check on; passing false
	// skips it, which is useful for unreleased pages but is rate limited to
	// 60000 distinct paths.
	CheckPath *bool

	// EnvVersion selects the mini program version: release, trial or develop.
	// Empty means release.
	EnvVersion string

	// Width is the side length in pixels, 280 to 1280. Zero means 430.
	Width int

	// AutoColor asks WeChat to pick the line colour from the page. When it is
	// false, LineColor applies.
	AutoColor bool

	// LineColor is the line colour, black by default.
	LineColor *RGBColor

	// IsHyaline asks for a transparent background; the reply is then a PNG.
	IsHyaline bool
}

// CheckUnlimitedQRCodeParams validates params before anything is sent to
// WeChat.
//
// The checks mirror what WeChat rejects anyway, so a bad call fails locally
// with a clear reason instead of as an opaque 40129/40097.
func CheckUnlimitedQRCodeParams(params GetUnlimitedQRCodeParams) error {
	if params.Scene == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: scene must not be empty",
		)
	}

	if utf8.RuneCountInString(params.Scene) > unlimitedQRCodeSceneMaxLen {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: scene must not exceed 32 visible characters",
		)
	}

	if !validScene(params.Scene) {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: scene may only contain digits, letters and !#$&'()*+,/:;=?@-._~",
		)
	}

	if params.Page != "" {
		if strings.HasPrefix(params.Page, "/") {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: page must not start with /",
			)
		}

		if strings.Contains(params.Page, "?") || strings.Contains(params.Page, "scancode_time") {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: page must not carry parameters, put them in scene instead",
			)
		}
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

// validScene reports whether scene only uses characters WeChat accepts.
func validScene(scene string) bool {
	for _, r := range scene {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case strings.ContainsRune(unlimitedQRCodeSceneChars, r):
		default:
			return false
		}
	}

	return true
}

// body renders params as the JSON body the endpoint expects.
func (p GetUnlimitedQRCodeParams) body() *getUnlimitedQRCodeBody {
	return &getUnlimitedQRCodeBody{
		Scene:      p.Scene,
		Page:       p.Page,
		CheckPath:  p.CheckPath,
		EnvVersion: p.EnvVersion,
		Width:      p.Width,
		AutoColor:  p.AutoColor,
		LineColor:  p.LineColor,
		IsHyaline:  p.IsHyaline,
	}
}

// getUnlimitedQRCodeBody is the JSON body of /wxa/getwxacodeunlimit.
type getUnlimitedQRCodeBody struct {
	Scene      string    `json:"scene"`
	Page       string    `json:"page,omitempty"`
	CheckPath  *bool     `json:"check_path,omitempty"`
	EnvVersion string    `json:"env_version,omitempty"`
	Width      int       `json:"width,omitempty"`
	AutoColor  bool      `json:"auto_color,omitempty"`
	LineColor  *RGBColor `json:"line_color,omitempty"`
	IsHyaline  bool      `json:"is_hyaline,omitempty"`
}

// GetUnlimitedQRCodeResponse is the reply of /wxa/getwxacodeunlimit.
//
// Like /wxa/getwxacode it answers with two media types: image bytes on success,
// JSON with errcode/errmsg on failure. The structure models both, so callers
// can inspect ErrCode before using Image.
type GetUnlimitedQRCodeResponse struct {
	// Common reply parameters. They are filled in only when WeChat returned
	// JSON instead of an image.
	response.WechatResponse

	// Business reply parameters. Image is the raw image bytes: JPEG normally,
	// PNG when IsHyaline was requested.
	Image       []byte
	ContentType string
}

// OK reports whether the response carries image bytes.
func (r *GetUnlimitedQRCodeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.Image) > 0
}

// GetUnlimitedQRCodeErrMsg returns the description of a getUnlimitedQRCode
// error.
func GetUnlimitedQRCodeErrMsg(code int) string {
	if msg, ok := GetUnlimitedQRCodeErrCodeMap[code]; ok {
		return msg
	}
	return GetUnlimitedQRCodeDefaultErrMsg
}

// GetUnlimitedQRCodeError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight
// from GetUnlimitedQRCodeErrCodeMap. An undocumented errcode has no agreed
// meaning, so it falls back to a generic message rather than exposing WeChat's
// raw errmsg.
func GetUnlimitedQRCodeError(r *GetUnlimitedQRCodeResponse) error {
	return errors.New(GetUnlimitedQRCodeErrMsg(r.ErrCode))
}

// GetUnlimitedQRCode returns a permanent mini program code driven by scene.
//
// This is the unlimited variant: there is no total quota, which makes it the
// right endpoint for codes created on the fly for many users. The trade-off is
// that the page cannot carry parameters itself; everything the page needs
// travels in Scene, which the mini program reads from the "scene" query
// parameter after decoding it.
//
// WeChat answers a failure with JSON rather than an image, so the reply type is
// decided by Content-Type, exactly like /wxa/getwxacode.
func (a *Application) GetUnlimitedQRCode(
	ctx context.Context,
	params GetUnlimitedQRCodeParams,
) (*GetUnlimitedQRCodeResponse, error) {
	if err := CheckUnlimitedQRCodeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getUnlimitedQRCodePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	image, contentType, errResp := decodeQRCodeImageReply(resp)
	if errResp.ErrCode != 0 {
		return &GetUnlimitedQRCodeResponse{WechatResponse: errResp, ContentType: contentType}, nil
	}

	return &GetUnlimitedQRCodeResponse{Image: image, ContentType: contentType}, nil
}
