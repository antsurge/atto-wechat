package miniProgram

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// generateShortLinkPath turns a mini program page into a short link that opens
// the mini program from inside WeChat.
const (
	generateShortLinkPath          = "/wxa/genwxashortlink"
	GenerateShortLinkDefaultErrMsg = "系统错误，请稍后再试"

	// generateShortLinkPageURLMaxLen is the documented limit of page_url.
	generateShortLinkPageURLMaxLen = 1024
)

// GenerateShortLinkErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/genwxashortlink, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GenerateShortLinkErrCodeMap = map[int]string{
	40001: "access_token 无效",
	40066: "url 不存在，即已发布小程序没有对应 url",
	40225: "无效的页面标题",
	43104: "没有调用权限，目前只开放给电商类目：电商平台、商家自营、跨境电商",
	45009: "单天生成 Short Link 数量超过上限 100 万",
	85400: "长期有效的 Scheme 或 short link 达到生成上限 10 万",
}

// GenerateShortLinkParams describes one short link.
//
// Unlike GenerateSchemeParams the page is passed as a single page_url that may
// carry its own query, rather than as a path plus a separate query.
type GenerateShortLinkParams struct {
	// PageURL is the page the short link opens, such as
	// "pages/index/index?foo=bar". No leading slash is expected. The page
	// must exist in a published mini program.
	PageURL string

	// PageTitle is the title shown on the card. WeChat truncates anything
	// beyond 20 characters with an ellipsis, and answers 40225 when the title
	// itself is invalid.
	PageTitle string

	// IsPermanent makes the short link permanent instead of the default
	// 30-day lifetime. Permanent links are capped at 100,000 per mini program,
	// and WeChat answers 85400 once the cap is reached.
	IsPermanent bool
}

// CheckGenerateShortLinkParams validates params before anything is sent to
// WeChat.
//
// Only page_url is checked: it is the single required field, and WeChat
// documents a length limit for it. page_title is truncated rather than rejected,
// so its length is left to WeChat.
func CheckGenerateShortLinkParams(params GenerateShortLinkParams) error {
	if params.PageURL == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: page_url is required",
		)
	}

	if len([]rune(params.PageURL)) > generateShortLinkPageURLMaxLen {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: page_url must not exceed %d characters",
				generateShortLinkPageURLMaxLen,
			),
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GenerateShortLinkParams) body() *generateShortLinkBody {
	return &generateShortLinkBody{
		PageURL:     p.PageURL,
		PageTitle:   p.PageTitle,
		IsPermanent: p.IsPermanent,
	}
}

// generateShortLinkBody is the JSON body of /wxa/genwxashortlink.
type generateShortLinkBody struct {
	PageURL     string `json:"page_url"`
	PageTitle   string `json:"page_title,omitempty"`
	IsPermanent bool   `json:"is_permanent,omitempty"`
}

// GenerateShortLinkResponse is the reply of /wxa/genwxashortlink.
//
// Unlike the other generating endpoints, the business reply is a bare string
// rather than an object: link is the short link itself.
type GenerateShortLinkResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Link string `json:"link"`
}

// OK reports whether the reply carries a short link.
func (r *GenerateShortLinkResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Link != ""
}

// GenerateShortLinkErrMsg returns the description of a generateShortLink error.
func GenerateShortLinkErrMsg(code int) string {
	if msg, ok := GenerateShortLinkErrCodeMap[code]; ok {
		return msg
	}
	return GenerateShortLinkDefaultErrMsg
}

// GenerateShortLinkError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GenerateShortLinkErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func GenerateShortLinkError(r *GenerateShortLinkResponse) error {
	return errors.New(GenerateShortLinkErrMsg(r.ErrCode))
}

// GenerateShortLink returns a short link for params.
//
// A short link is the WeChat-internal counterpart of a URL Link: it opens the
// mini program from inside WeChat, not from a SMS or an external browser. The
// link is short enough to fit a scene, a card or a chat message.
//
// The endpoint is open to non-personal mini programs in the e-commerce
// categories only, and WeChat answers 43104 otherwise. It also answers 40066
// when page_url does not point at a page of the published mini program.
//
// Generating is capped at 1,000,000 links per day, permanent ones included, and
// at 100,000 permanent links in total.
func (a *Application) GenerateShortLink(
	ctx context.Context,
	params GenerateShortLinkParams,
) (*GenerateShortLinkResponse, error) {
	if err := CheckGenerateShortLinkParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, generateShortLinkPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GenerateShortLinkResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
