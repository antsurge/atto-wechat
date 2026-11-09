package miniProgram

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setUpdatableMsgPath updates a dynamic message that has already been shared.
const (
	setUpdatableMsgPath          = "/cgi-bin/message/wxopen/updatablemsg/send"
	SetUpdatableMsgDefaultErrMsg = "系统错误，请稍后再试"
)

// Target states of a dynamic message.
//
// A message only moves forward: 0 may become 1, and 1 is final.
const (
	// UpdatableMsgStateNotStarted renders "成员正在加入，当前 {member_count}/{room_limit} 人".
	UpdatableMsgStateNotStarted = 0

	// UpdatableMsgStateStarted renders "已开始".
	UpdatableMsgStateStarted = 1
)

// Parameter names a dynamic message accepts.
//
// The first two are required while the message has not started, the last two
// once it has: WeChat renders a different template for each state.
const (
	// UpdatableMsgParamMemberCount is the number of members that have joined.
	UpdatableMsgParamMemberCount = "member_count"

	// UpdatableMsgParamRoomLimit is the capacity of the room.
	UpdatableMsgParamRoomLimit = "room_limit"

	// UpdatableMsgParamPath is the page opened by the "进入" button.
	UpdatableMsgParamPath = "path"

	// UpdatableMsgParamVersionType is the mini program version opened by the
	// "进入" button: "develop", "trial" or "release".
	UpdatableMsgParamVersionType = "version_type"
)

// SetUpdatableMsgErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/message/wxopen/updatablemsg/send, keyed by the errcode as it appears
// in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var SetUpdatableMsgErrCodeMap = map[int]string{
	40001: "access_token 无效",
	42001: "access_token 已过期",
	44002: "POST 数据为空",
	47001: "数据格式错误",
	47501: "activity_id 错误",
	47502: "target_state 错误",
	47503: "version_type 错误",
	47504: "activity_id 已过期",
}

// UpdatableMsgParameter is one field of the card to rewrite.
type UpdatableMsgParameter struct {
	// Name is one of the UpdatableMsgParam* constants.
	Name string `json:"name"`

	// Value is the new text of that field. It is always a string, even for a
	// count: the card renders it verbatim.
	Value string `json:"value"`
}

// SetUpdatableMsgParams describes the card after the update.
type SetUpdatableMsgParams struct {
	// ActivityID is the ID returned by CreateActivityID.
	ActivityID string

	// TargetState is UpdatableMsgStateNotStarted or UpdatableMsgStateStarted.
	TargetState int

	// ParameterList carries the fields to rewrite. Only the listed ones change,
	// so a caller may send just member_count when the room fills up.
	ParameterList []UpdatableMsgParameter
}

// CheckSetUpdatableMsgParams validates params before anything is sent to WeChat.
//
// The checks mirror what WeChat documents as mandatory: an unknown parameter
// name is dropped by WeChat rather than reported, and a state without its
// parameters renders an empty card, so both are worth catching here.
func CheckSetUpdatableMsgParams(params SetUpdatableMsgParams) error {
	if params.ActivityID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: activity_id is required",
		)
	}

	if params.TargetState != UpdatableMsgStateNotStarted &&
		params.TargetState != UpdatableMsgStateStarted {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: target_state must be %d or %d",
				UpdatableMsgStateNotStarted,
				UpdatableMsgStateStarted,
			),
		)
	}

	if len(params.ParameterList) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: parameter_list is required",
		)
	}

	seen := make(map[string]bool, len(params.ParameterList))

	for _, parameter := range params.ParameterList {
		switch parameter.Name {
		case UpdatableMsgParamMemberCount,
			UpdatableMsgParamRoomLimit,
			UpdatableMsgParamPath,
			UpdatableMsgParamVersionType:
		default:
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf("miniprogram: unknown parameter name %q", parameter.Name),
			)
		}

		if parameter.Value == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf("miniprogram: value of %q is required", parameter.Name),
			)
		}

		seen[parameter.Name] = true
	}

	// WeChat renders one template per state, and each reads a different pair of
	// parameters. Sending the other pair is harmless but useless, whereas
	// omitting the expected one leaves a blank field on the card.
	required := []string{
		UpdatableMsgParamMemberCount,
		UpdatableMsgParamRoomLimit,
	}
	if params.TargetState == UpdatableMsgStateStarted {
		required = []string{
			UpdatableMsgParamPath,
			UpdatableMsgParamVersionType,
		}
	}

	for _, name := range required {
		if !seen[name] {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf(
					"miniprogram: %q is required when target_state is %d",
					name,
					params.TargetState,
				),
			)
		}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SetUpdatableMsgParams) body() *setUpdatableMsgBody {
	return &setUpdatableMsgBody{
		ActivityID:  p.ActivityID,
		TargetState: p.TargetState,
		TemplateInfo: setUpdatableMsgTemplateInfo{
			ParameterList: p.ParameterList,
		},
	}
}

// setUpdatableMsgBody is the JSON body of
// /cgi-bin/message/wxopen/updatablemsg/send.
type setUpdatableMsgBody struct {
	ActivityID   string                      `json:"activity_id"`
	TargetState  int                         `json:"target_state"`
	TemplateInfo setUpdatableMsgTemplateInfo `json:"template_info"`
}

// setUpdatableMsgTemplateInfo wraps the fields of the card, so the parameters
// stay nested under template_info as WeChat expects.
type setUpdatableMsgTemplateInfo struct {
	ParameterList []UpdatableMsgParameter `json:"parameter_list"`
}

// SetUpdatableMsgResponse is the reply of
// /cgi-bin/message/wxopen/updatablemsg/send.
//
// It carries no business payload: success is reported by errcode alone.
type SetUpdatableMsgResponse struct {
	response.WechatResponse
}

// SetUpdatableMsgErrMsg returns the description of a setUpdatableMsg error.
func SetUpdatableMsgErrMsg(code int) string {
	if msg, ok := SetUpdatableMsgErrCodeMap[code]; ok {
		return msg
	}
	return SetUpdatableMsgDefaultErrMsg
}

// SetUpdatableMsgError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SetUpdatableMsgErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func SetUpdatableMsgError(r *SetUpdatableMsgResponse) error {
	return errors.New(SetUpdatableMsgErrMsg(r.ErrCode))
}

// SetUpdatableMsg rewrites a dynamic message that has already been shared.
//
// The message is identified by ActivityID and lives at most 24 hours. Its
// appearance is not arbitrary: WeChat renders one of two fixed templates, so
// TargetState chooses which fields are writable, and ParameterList only changes
// the values inside that template.
//
// A message that has started cannot go back, and an expired activity_id is
// answered with 47504; create a new one and share it again instead.
func (a *Application) SetUpdatableMsg(
	ctx context.Context,
	params SetUpdatableMsgParams,
) (*SetUpdatableMsgResponse, error) {
	if err := CheckSetUpdatableMsgParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, setUpdatableMsgPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SetUpdatableMsgResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
