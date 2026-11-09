package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPubTemplateTitlesPath lists the public templates of some categories.
const (
	getPubTemplateTitlesPath          = "/wxaapi/newtmpl/getpubtemplatetitles"
	GetPubTemplateTitlesDefaultErrMsg = "系统错误，请稍后再试"
)

// GetPubTemplateTitlesErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/newtmpl/getpubtemplatetitles, keyed by the errcode as it appears
// in the reply.
//
// WeChat does not publish a dedicated table for this endpoint; the map covers
// the codes the platform documents globally and omits the rest.
var GetPubTemplateTitlesErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	45011: "调用太频繁，请稍后再试",
}

// GetPubTemplateTitlesParams selects the public templates to browse.
type GetPubTemplateTitlesParams struct {
	// CategoryIDs are the category ids from GetCategory. At least one is
	// required; WeChat takes them joined by commas.
	CategoryIDs []int64

	// Start is the offset of the first template to return, from 0.
	Start int

	// Limit is the page size. WeChat defaults to 30 when it is zero.
	Limit int
}

// CheckGetPubTemplateTitlesParams validates params before anything is sent to
// WeChat.
func CheckGetPubTemplateTitlesParams(params GetPubTemplateTitlesParams) error {
	if len(params.CategoryIDs) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ids is required",
		)
	}

	if params.Start < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: start must not be negative",
		)
	}

	if params.Limit < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: limit must not be negative",
		)
	}

	return nil
}

// query renders params as the query string of the endpoint.
//
// A zero limit is dropped so WeChat applies its default rather than receiving a
// page size of zero.
func (p GetPubTemplateTitlesParams) query() url.Values {
	ids := make([]string, 0, len(p.CategoryIDs))
	for _, id := range p.CategoryIDs {
		ids = append(ids, strconv.FormatInt(id, 10))
	}

	q := url.Values{
		"ids":   {strings.Join(ids, ",")},
		"start": {strconv.Itoa(p.Start)},
	}

	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}

	return q
}

// PubTemplateTitle is one public template.
type PubTemplateTitle struct {
	// TitleID is the public template id (tid), used by AddMessageTemplate.
	TitleID string `json:"tid"`

	// Title is the sentence of the template, with its keywords in braces.
	Title string `json:"title"`

	// Type is the template type WeChat assigns: 2 for a one-off subscription.
	Type int `json:"type"`

	// Scenes are the category ids the template belongs to.
	Scenes []string `json:"scenes"`
}

// GetPubTemplateTitlesResponse is the reply of
// /wxaapi/newtmpl/getpubtemplatetitles.
type GetPubTemplateTitlesResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Count int                `json:"count"`
	Data  []PubTemplateTitle `json:"data"`
}

// GetPubTemplateTitlesErrMsg returns the description of a
// getPubTemplateTitles error.
func GetPubTemplateTitlesErrMsg(code int) string {
	if msg, ok := GetPubTemplateTitlesErrCodeMap[code]; ok {
		return msg
	}
	return GetPubTemplateTitlesDefaultErrMsg
}

// GetPubTemplateTitlesError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetPubTemplateTitlesErrCodeMap. An undocumented errcode has no agreed meaning,
// so it falls back to a generic message rather than exposing WeChat's raw
// errmsg.
func GetPubTemplateTitlesError(r *GetPubTemplateTitlesResponse) error {
	return errors.New(GetPubTemplateTitlesErrMsg(r.ErrCode))
}

// GetPubTemplateTitles lists the public templates of the given categories.
//
// Public templates are provided by WeChat, and a mini program may add at most 25
// of them. Browsing needs a category id from GetCategory, and the count in the
// reply is the total rather than the size of this page.
func (a *Application) GetPubTemplateTitles(
	ctx context.Context,
	params GetPubTemplateTitlesParams,
) (*GetPubTemplateTitlesResponse, error) {
	if err := CheckGetPubTemplateTitlesParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getPubTemplateTitlesPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetPubTemplateTitlesResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
