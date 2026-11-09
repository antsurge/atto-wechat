package miniProgram

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// sendSubscribeMessagePath delivers a subscribed message to one user.
const (
	sendSubscribeMessagePath          = "/cgi-bin/message/subscribe/send"
	SendSubscribeMessageDefaultErrMsg = "系统错误，请稍后再试"
)

// Mini program versions a message may open.
const (
	// SubscribeMsgStateFormal opens the released version.
	SubscribeMsgStateFormal = "formal"

	// SubscribeMsgStateTrial opens the trial version.
	SubscribeMsgStateTrial = "trial"

	// SubscribeMsgStateDeveloper opens the development version.
	SubscribeMsgStateDeveloper = "developer"
)

// Languages a message may be rendered in.
const (
	SubscribeMsgLangZhCN = "zh_CN"
	SubscribeMsgLangEnUS = "en_US"
	SubscribeMsgLangZhHK = "zh_HK"
	SubscribeMsgLangZhTW = "zh_TW"
)

// SendSubscribeMessageErrCodeMap holds the errcode descriptions WeChat
// documents for /cgi-bin/message/subscribe/send, keyed by the errcode as it
// appears in the reply.
//
// 43101 is by far the most common one: the user never granted the subscription,
// or granted it once and has already used the grant.
var SendSubscribeMessageErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40003: "touser 不正确",
	40037: "template_id 不正确",
	41030: "page 路径不正确",
	42001: "access_token 已过期",
	43101: "用户拒绝接受消息，或已取消订阅",
	47003: "模板参数不准确，可能为空或不满足规则",
}

// SubscribeMessageValue is the value of one keyword.
type SubscribeMessageValue struct {
	// Value is the text rendered into the template. WeChat validates it
	// against the keyword's rule (thing, number, date, ...) and answers 47003
	// when it does not fit.
	Value string `json:"value"`
}

// SendSubscribeMessageParams describes one subscription message.
type SendSubscribeMessageParams struct {
	// ToUser is the openid of the recipient.
	ToUser string

	// TemplateID is the priTmplId from AddMessageTemplate, not the public tid.
	TemplateID string

	// Page is the page opened when the message is tapped. It must be a
	// published page; WeChat answers 41030 otherwise.
	Page string

	// Data carries one value per keyword kept on the template. The keys are
	// the keyword names, which include their rule and index, such as "thing1"
	// or "date2".
	Data map[string]SubscribeMessageValue

	// MiniProgramState is the version the message opens: SubscribeMsgState*.
	// Empty lets WeChat open the released version.
	MiniProgramState string

	// Lang is the language the message is rendered in: SubscribeMsgLang*.
	// Empty lets WeChat pick the user's language.
	Lang string
}

// CheckSendSubscribeMessageParams validates params before anything is sent to
// WeChat.
//
// What a keyword accepts depends on the template, so the values themselves are
// left to WeChat: it answers 47003 and names the offending field.
func CheckSendSubscribeMessageParams(params SendSubscribeMessageParams) error {
	if params.ToUser == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: touser is required",
		)
	}

	if params.TemplateID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: template_id is required",
		)
	}

	if len(params.Data) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: data is required",
		)
	}

	for key, value := range params.Data {
		if key == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: data key must not be empty",
			)
		}

		if value.Value == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf("miniprogram: data[%q].value is required", key),
			)
		}
	}

	switch params.MiniProgramState {
	case "", SubscribeMsgStateFormal, SubscribeMsgStateTrial, SubscribeMsgStateDeveloper:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: miniprogram_state must be %q, %q or %q",
				SubscribeMsgStateFormal,
				SubscribeMsgStateTrial,
				SubscribeMsgStateDeveloper,
			),
		)
	}

	switch params.Lang {
	case "", SubscribeMsgLangZhCN, SubscribeMsgLangEnUS, SubscribeMsgLangZhHK, SubscribeMsgLangZhTW:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: lang must be %q, %q, %q or %q",
				SubscribeMsgLangZhCN,
				SubscribeMsgLangEnUS,
				SubscribeMsgLangZhHK,
				SubscribeMsgLangZhTW,
			),
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SendSubscribeMessageParams) body() *sendSubscribeMessageBody {
	return &sendSubscribeMessageBody{
		ToUser:           p.ToUser,
		TemplateID:       p.TemplateID,
		Page:             p.Page,
		Data:             p.Data,
		MiniProgramState: p.MiniProgramState,
		Lang:             p.Lang,
	}
}

// sendSubscribeMessageBody is the JSON body of
// /cgi-bin/message/subscribe/send.
type sendSubscribeMessageBody struct {
	ToUser           string                           `json:"touser"`
	TemplateID       string                           `json:"template_id"`
	Page             string                           `json:"page,omitempty"`
	Data             map[string]SubscribeMessageValue `json:"data"`
	MiniProgramState string                           `json:"miniprogram_state,omitempty"`
	Lang             string                           `json:"lang,omitempty"`
}

// SendSubscribeMessageResponse is the reply of
// /cgi-bin/message/subscribe/send.
//
// It carries no business payload: success is reported by errcode alone.
type SendSubscribeMessageResponse struct {
	response.WechatResponse
}

// SendSubscribeMessageErrMsg returns the description of a
// sendSubscribeMessage error.
func SendSubscribeMessageErrMsg(code int) string {
	if msg, ok := SendSubscribeMessageErrCodeMap[code]; ok {
		return msg
	}
	return SendSubscribeMessageDefaultErrMsg
}

// SendSubscribeMessageError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SendSubscribeMessageErrCodeMap. An undocumented errcode has no agreed meaning,
// so it falls back to a generic message rather than exposing WeChat's raw
// errmsg.
func SendSubscribeMessageError(r *SendSubscribeMessageResponse) error {
	return errors.New(SendSubscribeMessageErrMsg(r.ErrCode))
}

// SendSubscribeMessage sends a subscription message to one user.
//
// The user must have granted the subscription first, either once through a
// tap or permanently through a request on the client. A one-off grant is
// consumed by a successful send, so the same message cannot be sent twice: the
// second attempt is answered with 43101.
//
// The template dictates the keys of Data and the rules of their values, which
// makes 47003 the error to surface verbatim to developers: its errmsg names the
// field WeChat rejected.
func (a *Application) SendSubscribeMessage(
	ctx context.Context,
	params SendSubscribeMessageParams,
) (*SendSubscribeMessageResponse, error) {
	if err := CheckSendSubscribeMessageParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, sendSubscribeMessagePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SendSubscribeMessageResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
