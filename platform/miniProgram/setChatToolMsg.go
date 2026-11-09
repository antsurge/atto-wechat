package miniProgram

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setChatToolMsgPath updates the card a chat tool shared into a group chat.
const (
	setChatToolMsgPath          = "/cgi-bin/message/wxopen/chattoolmsg/send"
	SetChatToolMsgDefaultErrMsg = "系统错误，请稍后再试"
)

// Target states of a chat tool card.
const (
	// ChatToolMsgStateRunning keeps the card on the "in progress" template.
	ChatToolMsgStateRunning = 1

	// ChatToolMsgStateFinished switches the card to the "finished" template.
	ChatToolMsgStateFinished = 3
)

// Mini program versions a card may open.
const (
	// ChatToolMsgVersionRelease opens the released version.
	ChatToolMsgVersionRelease = 0

	// ChatToolMsgVersionDevelop opens the development version.
	ChatToolMsgVersionDevelop = 1

	// ChatToolMsgVersionTrial opens the trial version.
	ChatToolMsgVersionTrial = 2
)

// Participant states shown on the card.
const (
	// ChatToolMsgParticipantJoined marks a member as having joined.
	ChatToolMsgParticipantJoined = 1
)

// SetChatToolMsgErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/message/wxopen/chattoolmsg/send, keyed by the errcode as it appears
// in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var SetChatToolMsgErrCodeMap = map[int]string{
	40001: "access_token 无效",
	40013: "AppID 无效",
	40097: "参数错误",
	42001: "access_token 已过期",
	44002: "POST 数据为空",
	47001: "数据格式错误",
	47501: "activity_id 错误",
	47502: "target_state 错误",
	47504: "activity_id 已过期",
}

// ChatToolMsgParticipator is one member shown on the card.
type ChatToolMsgParticipator struct {
	// GroupOpenID is the member's openid inside the group chat.
	GroupOpenID string `json:"group_openid"`

	// State is ChatToolMsgParticipantJoined once the member has joined.
	State int `json:"state"`
}

// SetChatToolMsgParams describes the card after the update.
type SetChatToolMsgParams struct {
	// ActivityID is the ID returned by CreateActivityID.
	ActivityID string

	// TargetState is ChatToolMsgStateRunning or ChatToolMsgStateFinished.
	TargetState int

	// TemplateID is the chat tool template the card was shared with. Unlike a
	// dynamic message, WeChat requires it on every update.
	TemplateID string

	// VersionType is the mini program version the card opens.
	VersionType int

	// ParticipatorList carries the members to show. It is optional: an update
	// that only closes the activity can leave it nil.
	ParticipatorList []ChatToolMsgParticipator
}

// CheckSetChatToolMsgParams validates params before anything is sent to WeChat.
//
// target_state and version_type are only checked for being set: WeChat owns
// their value sets, and a value added later must not be rejected here.
func CheckSetChatToolMsgParams(params SetChatToolMsgParams) error {
	if params.ActivityID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: activity_id is required",
		)
	}

	if params.TemplateID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: template_id is required",
		)
	}

	if params.TargetState == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: target_state is required",
		)
	}

	for i, participator := range params.ParticipatorList {
		if participator.GroupOpenID == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf(
					"miniprogram: participator_info_list[%d].group_openid is required",
					i,
				),
			)
		}

		if participator.State == 0 {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf(
					"miniprogram: participator_info_list[%d].state is required",
					i,
				),
			)
		}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SetChatToolMsgParams) body() *setChatToolMsgBody {
	return &setChatToolMsgBody{
		ActivityID:           p.ActivityID,
		TargetState:          p.TargetState,
		TemplateID:           p.TemplateID,
		VersionType:          p.VersionType,
		ParticipatorInfoList: p.ParticipatorList,
	}
}

// setChatToolMsgBody is the JSON body of
// /cgi-bin/message/wxopen/chattoolmsg/send.
type setChatToolMsgBody struct {
	ActivityID           string                    `json:"activity_id"`
	TargetState          int                       `json:"target_state"`
	TemplateID           string                    `json:"template_id"`
	VersionType          int                       `json:"version_type"`
	ParticipatorInfoList []ChatToolMsgParticipator `json:"participator_info_list,omitempty"`
}

// SetChatToolMsgResponse is the reply of
// /cgi-bin/message/wxopen/chattoolmsg/send.
//
// It carries no business payload: success is reported by errcode alone.
type SetChatToolMsgResponse struct {
	response.WechatResponse
}

// SetChatToolMsgErrMsg returns the description of a setChatToolMsg error.
func SetChatToolMsgErrMsg(code int) string {
	if msg, ok := SetChatToolMsgErrCodeMap[code]; ok {
		return msg
	}
	return SetChatToolMsgDefaultErrMsg
}

// SetChatToolMsgError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SetChatToolMsgErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func SetChatToolMsgError(r *SetChatToolMsgResponse) error {
	return errors.New(SetChatToolMsgErrMsg(r.ErrCode))
}

// SetChatToolMsg updates the card a chat tool shared into a group chat.
//
// It is the chat tool counterpart of SetUpdatableMsg: the card keeps reporting
// who has joined and whether the activity is still running. Because the card is
// a chat tool template rather than a fixed one, WeChat needs TemplateID on every
// update.
//
// The activity_id expires 24 hours after creation, and an expired one is
// answered with 47504.
func (a *Application) SetChatToolMsg(
	ctx context.Context,
	params SetChatToolMsgParams,
) (*SetChatToolMsgResponse, error) {
	if err := CheckSetChatToolMsgParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, setChatToolMsgPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SetChatToolMsgResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
