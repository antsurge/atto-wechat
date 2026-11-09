package miniProgram

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setTypingPath shows or hides the "typing" hint in a conversation.
const (
	setTypingPath          = "/cgi-bin/message/custom/business/typing"
	SetTypingDefaultErrMsg = "系统错误，请稍后再试"
)

// Commands the typing hint accepts.
const (
	// TypingCommandTyping shows the hint for 30 seconds.
	TypingCommandTyping = "Typing"

	// TypingCommandCancelTyping hides it early.
	TypingCommandCancelTyping = "CancelTyping"
)

// SetTypingErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/message/custom/business/typing, keyed by the errcode as it appears
// in the reply.
var SetTypingErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40003: "touser 不正确",
	42001: "access_token 已过期",
	45015: "回复时间超过限制，用户已超过 48 小时未与客服交互",
}

// SetTypingParams describes the hint to show.
type SetTypingParams struct {
	// ToUser is the openid of the user in the conversation.
	ToUser string

	// Command is TypingCommandTyping or TypingCommandCancelTyping.
	Command string
}

// CheckSetTypingParams validates params before anything is sent to WeChat.
func CheckSetTypingParams(params SetTypingParams) error {
	if params.ToUser == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: touser is required",
		)
	}

	switch params.Command {
	case TypingCommandTyping, TypingCommandCancelTyping:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: command must be %q or %q",
				TypingCommandTyping,
				TypingCommandCancelTyping,
			),
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SetTypingParams) body() *setTypingBody {
	return &setTypingBody{ToUser: p.ToUser, Command: p.Command}
}

// setTypingBody is the JSON body of /cgi-bin/message/custom/business/typing.
type setTypingBody struct {
	ToUser  string `json:"touser"`
	Command string `json:"command"`
}

// SetTypingResponse is the reply of /cgi-bin/message/custom/business/typing.
//
// It carries no business payload: success is reported by errcode alone.
type SetTypingResponse struct {
	response.WechatResponse
}

// SetTypingErrMsg returns the description of a setTyping error.
func SetTypingErrMsg(code int) string {
	if msg, ok := SetTypingErrCodeMap[code]; ok {
		return msg
	}
	return SetTypingDefaultErrMsg
}

// SetTypingError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SetTypingErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func SetTypingError(r *SetTypingResponse) error {
	return errors.New(SetTypingErrMsg(r.ErrCode))
}

// SetTyping shows or hides the typing hint in a conversation.
//
// The hint tells the user that a human is answering, and lasts 30 seconds by
// itself: a slow agent has to refresh it, and a finished one may cancel it
// early. It shares the 48-hour window of SendCustomMessage.
func (a *Application) SetTyping(
	ctx context.Context,
	params SetTypingParams,
) (*SetTypingResponse, error) {
	if err := CheckSetTypingParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, setTypingPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SetTypingResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
