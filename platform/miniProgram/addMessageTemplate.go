package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addMessageTemplatePath adds a public template to the mini program.
const (
	addMessageTemplatePath          = "/wxaapi/newtmpl/addtemplate"
	AddMessageTemplateDefaultErrMsg = "系统错误，请稍后再试"
)

// AddMessageTemplateErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/newtmpl/addtemplate, keyed by the errcode as it appears in the
// reply.
//
// WeChat does not publish a dedicated table for this endpoint; the map covers
// the codes the platform documents globally and omits the rest.
var AddMessageTemplateErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	45011: "调用太频繁，请稍后再试",
	47001: "数据格式错误",
}

// AddMessageTemplateParams selects the public template and its keywords.
type AddMessageTemplateParams struct {
	// TitleID is the public template id (tid) from GetPubTemplateTitles.
	TitleID string

	// KeywordIDs are the kids from GetPubTemplateKeywords to keep on the
	// template. They must belong to TitleID, and at least one is required.
	KeywordIDs []int64

	// SceneDescription explains where the message is sent, so the user can
	// tell the templates apart in the subscription dialog.
	SceneDescription string
}

// CheckAddMessageTemplateParams validates params before anything is sent to
// WeChat.
func CheckAddMessageTemplateParams(params AddMessageTemplateParams) error {
	if params.TitleID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: tid is required",
		)
	}

	if len(params.KeywordIDs) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: kidList is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p AddMessageTemplateParams) body() *addMessageTemplateBody {
	return &addMessageTemplateBody{
		TitleID:          p.TitleID,
		KeywordIDs:       p.KeywordIDs,
		SceneDescription: p.SceneDescription,
	}
}

// addMessageTemplateBody is the JSON body of /wxaapi/newtmpl/addtemplate.
type addMessageTemplateBody struct {
	TitleID          string  `json:"tid"`
	KeywordIDs       []int64 `json:"kidList"`
	SceneDescription string  `json:"sceneDesc,omitempty"`
}

// AddMessageTemplateResponse is the reply of /wxaapi/newtmpl/addtemplate.
type AddMessageTemplateResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	PriTemplateID string `json:"priTmplId"`
}

// OK reports whether the reply carries a template id.
func (r *AddMessageTemplateResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.PriTemplateID != ""
}

// AddMessageTemplateErrMsg returns the description of an addMessageTemplate
// error.
func AddMessageTemplateErrMsg(code int) string {
	if msg, ok := AddMessageTemplateErrCodeMap[code]; ok {
		return msg
	}
	return AddMessageTemplateDefaultErrMsg
}

// AddMessageTemplateError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// AddMessageTemplateErrCodeMap. An undocumented errcode has no agreed meaning,
// so it falls back to a generic message rather than exposing WeChat's raw
// errmsg.
func AddMessageTemplateError(r *AddMessageTemplateResponse) error {
	return errors.New(AddMessageTemplateErrMsg(r.ErrCode))
}

// AddMessageTemplate adds a public template to the mini program.
//
// The public templates of the account's categories are provided by WeChat, and
// this call freezes one of them into the mini program with the keywords the
// caller keeps. The returned priTmplId is what SendSubscribeMessage references,
// and it is the only id that survives a template being taken offline: the
// public tid may later point at a different sentence.
//
// A mini program may hold at most 25 templates, and each keyword keeps its own
// value type, which constrains the data of every message sent with it.
func (a *Application) AddMessageTemplate(
	ctx context.Context,
	params AddMessageTemplateParams,
) (*AddMessageTemplateResponse, error) {
	if err := CheckAddMessageTemplateParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addMessageTemplatePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result AddMessageTemplateResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
