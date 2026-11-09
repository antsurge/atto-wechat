package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteMessageTemplatePath removes a template from the mini program.
const (
	deleteMessageTemplatePath          = "/wxaapi/newtmpl/deltemplate"
	DeleteMessageTemplateDefaultErrMsg = "系统错误，请稍后再试"
)

// DeleteMessageTemplateErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/newtmpl/deltemplate, keyed by the errcode as it appears
// in the reply.
//
// WeChat does not publish a dedicated table for this endpoint; the map covers
// the codes the platform documents globally and omits the rest.
var DeleteMessageTemplateErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	45011: "调用太频繁，请稍后再试",
	47001: "数据格式错误",
}

// DeleteMessageTemplateParams identifies the template to remove.
type DeleteMessageTemplateParams struct {
	// PriTemplateID is the template id returned by AddMessageTemplate, which is
	// the id used when sending, not the public template id.
	PriTemplateID string
}

// CheckDeleteMessageTemplateParams validates params before anything is sent to
// WeChat.
func CheckDeleteMessageTemplateParams(params DeleteMessageTemplateParams) error {
	if params.PriTemplateID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: priTmplId is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p DeleteMessageTemplateParams) body() *deleteMessageTemplateBody {
	return &deleteMessageTemplateBody{PriTemplateID: p.PriTemplateID}
}

// deleteMessageTemplateBody is the JSON body of /wxaapi/newtmpl/deltemplate.
type deleteMessageTemplateBody struct {
	PriTemplateID string `json:"priTmplId"`
}

// DeleteMessageTemplateResponse is the reply of /wxaapi/newtmpl/deltemplate.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteMessageTemplateResponse struct {
	response.WechatResponse
}

// DeleteMessageTemplateErrMsg returns the description of a
// deleteMessageTemplate error.
func DeleteMessageTemplateErrMsg(code int) string {
	if msg, ok := DeleteMessageTemplateErrCodeMap[code]; ok {
		return msg
	}
	return DeleteMessageTemplateDefaultErrMsg
}

// DeleteMessageTemplateError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// DeleteMessageTemplateErrCodeMap. An undocumented errcode has no agreed
// meaning, so it falls back to a generic message rather than exposing WeChat's
// raw errmsg.
func DeleteMessageTemplateError(r *DeleteMessageTemplateResponse) error {
	return errors.New(DeleteMessageTemplateErrMsg(r.ErrCode))
}

// DeleteMessageTemplate removes a subscription message template.
//
// The template is chosen from the public templates of the account's categories,
// so removing one frees a slot in the 25-template limit but also stops every
// message that still references it: WeChat answers 43101 at send time once the
// user has subscribed to a template that no longer exists.
func (a *Application) DeleteMessageTemplate(
	ctx context.Context,
	params DeleteMessageTemplateParams,
) (*DeleteMessageTemplateResponse, error) {
	if err := CheckDeleteMessageTemplateParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, deleteMessageTemplatePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result DeleteMessageTemplateResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
