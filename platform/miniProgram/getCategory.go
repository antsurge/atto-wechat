package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getCategoryPath lists the categories the mini program belongs to.
const (
	getCategoryPath          = "/wxaapi/newtmpl/getcategory"
	GetCategoryDefaultErrMsg = "系统错误，请稍后再试"
)

// GetCategoryErrCodeMap holds the errcode descriptions WeChat documents for
// /wxaapi/newtmpl/getcategory, keyed by the errcode as it appears in the reply.
//
// WeChat does not publish a dedicated table for this endpoint; the map covers
// the codes the platform documents globally and omits the rest.
var GetCategoryErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	45011: "调用太频繁，请稍后再试",
}

// TemplateCategory is one category of the mini program.
type TemplateCategory struct {
	// ID is the category id, used as tid when browsing public templates.
	ID int64 `json:"id"`

	// Name is the human-readable category name.
	Name string `json:"name"`
}

// GetCategoryResponse is the reply of /wxaapi/newtmpl/getcategory.
type GetCategoryResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Data []TemplateCategory `json:"data"`
}

// GetCategoryErrMsg returns the description of a getCategory error.
func GetCategoryErrMsg(code int) string {
	if msg, ok := GetCategoryErrCodeMap[code]; ok {
		return msg
	}
	return GetCategoryDefaultErrMsg
}

// GetCategoryError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetCategoryErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetCategoryError(r *GetCategoryResponse) error {
	return errors.New(GetCategoryErrMsg(r.ErrCode))
}

// GetCategory lists the categories the mini program belongs to.
//
// A subscription message template can only be picked from the categories listed
// here, so this is the first call of the template setup: its ids are the tid
// values passed to GetPubTemplateTitles.
func (a *Application) GetCategory(
	ctx context.Context,
) (*GetCategoryResponse, error) {
	resp, err := a.api.Get(ctx, getCategoryPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetCategoryResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
