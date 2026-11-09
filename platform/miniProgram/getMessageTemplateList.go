package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getMessageTemplateListPath lists the templates the mini program has added.
const (
	getMessageTemplateListPath          = "/wxaapi/newtmpl/gettemplate"
	GetMessageTemplateListDefaultErrMsg = "系统错误，请稍后再试"
)

// GetMessageTemplateListErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/newtmpl/gettemplate, keyed by the errcode as it appears
// in the reply.
//
// WeChat does not publish a dedicated table for this endpoint; the map covers
// the codes the platform documents globally and omits the rest.
var GetMessageTemplateListErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	45011: "调用太频繁，请稍后再试",
}

// MessageTemplate is one template the mini program has added.
type MessageTemplate struct {
	// PriTemplateID is the id used when sending, not the public template id.
	PriTemplateID string `json:"priTmplId"`

	// Title is the sentence of the template.
	Title string `json:"title"`

	// Content is the sentence with its keyword placeholders numbered.
	Content string `json:"content"`

	// Example is a filled-in sample of the sentence.
	Example string `json:"example"`

	// Type is the template type WeChat assigns: 2 for a one-off subscription.
	Type int `json:"type"`
}

// GetMessageTemplateListResponse is the reply of /wxaapi/newtmpl/gettemplate.
type GetMessageTemplateListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Data []MessageTemplate `json:"data"`
}

// GetMessageTemplateListErrMsg returns the description of a
// getMessageTemplateList error.
func GetMessageTemplateListErrMsg(code int) string {
	if msg, ok := GetMessageTemplateListErrCodeMap[code]; ok {
		return msg
	}
	return GetMessageTemplateListDefaultErrMsg
}

// GetMessageTemplateListError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetMessageTemplateListErrCodeMap. An undocumented errcode has no agreed
// meaning, so it falls back to a generic message rather than exposing WeChat's
// raw errmsg.
func GetMessageTemplateListError(r *GetMessageTemplateListResponse) error {
	return errors.New(GetMessageTemplateListErrMsg(r.ErrCode))
}

// GetMessageTemplateList lists the templates the mini program has added.
//
// The ids it returns are the priTmplId values SendSubscribeMessage needs: the
// public template id is only used while adding. A mini program may hold at most
// 25 templates.
func (a *Application) GetMessageTemplateList(
	ctx context.Context,
) (*GetMessageTemplateListResponse, error) {
	resp, err := a.api.Get(ctx, getMessageTemplateListPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetMessageTemplateListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
