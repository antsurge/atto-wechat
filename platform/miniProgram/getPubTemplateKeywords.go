package miniProgram

import (
	"context"
	"errors"
	"net/url"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPubTemplateKeywordsPath lists the keywords of a public template.
const (
	getPubTemplateKeywordsPath          = "/wxaapi/newtmpl/getpubtemplatekeywords"
	GetPubTemplateKeywordsDefaultErrMsg = "系统错误，请稍后再试"
)

// GetPubTemplateKeywordsErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/newtmpl/getpubtemplatekeywords, keyed by the errcode as
// it appears in the reply.
//
// WeChat does not publish a dedicated table for this endpoint; the map covers
// the codes the platform documents globally and omits the rest.
var GetPubTemplateKeywordsErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	45011: "调用太频繁，请稍后再试",
}

// GetPubTemplateKeywordsParams identifies the public template to inspect.
type GetPubTemplateKeywordsParams struct {
	// TitleID is the public template id (tid) from GetPubTemplateTitles.
	TitleID string
}

// CheckGetPubTemplateKeywordsParams validates params before anything is sent to
// WeChat.
func CheckGetPubTemplateKeywordsParams(params GetPubTemplateKeywordsParams) error {
	if params.TitleID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: tid is required",
		)
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetPubTemplateKeywordsParams) query() url.Values {
	return url.Values{"tid": {p.TitleID}}
}

// PubTemplateKeyword is one keyword of a public template.
type PubTemplateKeyword struct {
	// Kid is the keyword id, used in AddMessageTemplate's kidList.
	Kid int64 `json:"kid"`

	// Name is the keyword name shown on the card, such as "物品名称".
	Name string `json:"name"`

	// Example is a sample value for the keyword.
	Example string `json:"example"`

	// Rule is the value type WeChat validates against, such as "thing",
	// "number", "date" or "phrase".
	Rule string `json:"rule"`
}

// GetPubTemplateKeywordsResponse is the reply of
// /wxaapi/newtmpl/getpubtemplatekeywords.
type GetPubTemplateKeywordsResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Data []PubTemplateKeyword `json:"data"`
}

// GetPubTemplateKeywordsErrMsg returns the description of a
// getPubTemplateKeywords error.
func GetPubTemplateKeywordsErrMsg(code int) string {
	if msg, ok := GetPubTemplateKeywordsErrCodeMap[code]; ok {
		return msg
	}
	return GetPubTemplateKeywordsDefaultErrMsg
}

// GetPubTemplateKeywordsError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetPubTemplateKeywordsErrCodeMap. An undocumented errcode has no agreed
// meaning, so it falls back to a generic message rather than exposing WeChat's
// raw errmsg.
func GetPubTemplateKeywordsError(r *GetPubTemplateKeywordsResponse) error {
	return errors.New(GetPubTemplateKeywordsErrMsg(r.ErrCode))
}

// GetPubTemplateKeywords lists the keywords of a public template.
//
// A public template is a fixed sentence with holes in it, and every hole is a
// keyword with its own value type. The kid values returned here are what
// AddMessageTemplate picks, and the rules tell the caller what the data of a
// message may contain.
func (a *Application) GetPubTemplateKeywords(
	ctx context.Context,
	params GetPubTemplateKeywordsParams,
) (*GetPubTemplateKeywordsResponse, error) {
	if err := CheckGetPubTemplateKeywordsParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getPubTemplateKeywordsPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetPubTemplateKeywordsResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
