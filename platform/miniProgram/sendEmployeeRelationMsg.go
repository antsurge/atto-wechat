package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// sendEmployeeRelationMsgPath pushes a template message over 用工关系.
const sendEmployeeRelationMsgPath = "/cgi-bin/message/wxopen/employeerelationmsg/send"

// SendEmployeeRelationMsgErrCodeMap holds the errcode descriptions WeChat
// documents for this endpoint, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var SendEmployeeRelationMsgErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40097: "参数错误",
	40256: "该用户没有绑定用工关系",
	40257: "解析接收用户 openid 失败",
	43104: "app 无权限推送模版消息",
	50001: "用户未订阅模版",
}

// SendEmployeeRelationMsgDefaultErrMsg is returned for an errcode this endpoint
// does not document.
const SendEmployeeRelationMsgDefaultErrMsg = "系统错误，请稍后再试"

// SendEmployeeRelationMsgErrMsg returns the description of a
// sendEmployeeRelationMsg error, or the default when the errcode is undocumented.
func SendEmployeeRelationMsgErrMsg(code int) string {
	if msg, ok := SendEmployeeRelationMsgErrCodeMap[code]; ok {
		return msg
	}

	return SendEmployeeRelationMsgDefaultErrMsg
}

// SendEmployeeRelationMsgParams is the message to push.
type SendEmployeeRelationMsgParams struct {
	// TemplateID is the template to fill.
	TemplateID string

	// Page is the mini program page the message opens.
	Page string

	// ToUser is the openid of the user, who must have a 用工关系 with the mini
	// program -- otherwise WeChat answers 40256.
	ToUser string

	// Data is the template content, as a JSON *string*: WeChat expects
	// "{\"data\":{...}}" here, not a nested object. Use
	// BuildEmployeeRelationMsgData to produce it instead of writing the escaping by
	// hand.
	Data string
}

// CheckSendEmployeeRelationMsgParams validates params before anything is sent to
// WeChat.
func CheckSendEmployeeRelationMsgParams(params SendEmployeeRelationMsgParams) error {
	switch {
	case params.TemplateID == "":
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendEmployeeRelationMsg template_id is required",
		)
	case params.Page == "":
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendEmployeeRelationMsg page is required",
		)
	case params.ToUser == "":
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendEmployeeRelationMsg touser is required",
		)
	case params.Data == "":
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendEmployeeRelationMsg data is required",
		)
	}

	// data is a string that has to hold a JSON object. Checking it here catches the
	// two mistakes this field invites -- sending the object instead of the string,
	// or escaping it twice -- which WeChat reports only as a generic 参数错误.
	var payload map[string]any
	if err := codec.Unmarshal([]byte(params.Data), &payload); err != nil || payload == nil {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendEmployeeRelationMsg data must be a JSON object encoded as a string",
		)
	}

	return nil
}

// BuildEmployeeRelationMsgData renders template values as the data string the
// endpoint expects.
//
// The result is "{\"data\":{\"key\":{\"value\":\"...\"}}}", which is what
// SendEmployeeRelationMsgParams.Data carries. Map keys are sorted by the encoder,
// so the same input always produces the same string.
func BuildEmployeeRelationMsgData(values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: employee relation message data must not be empty",
		)
	}

	payload := make(map[string]map[string]string, len(values))

	for key, value := range values {
		if key == "" {
			return "", cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: employee relation message keyword must not be empty",
			)
		}

		payload[key] = map[string]string{"value": value}
	}

	body, err := codec.Marshal(map[string]any{"data": payload})
	if err != nil {
		return "", cerrors.Wrap(cerrors.CodeEncodeFailed, "miniprogram: encode employee relation message data", err)
	}

	return string(body), nil
}

// body renders params as the JSON body the endpoint expects.
func (p SendEmployeeRelationMsgParams) body() *sendEmployeeRelationMsgBody {
	return &sendEmployeeRelationMsgBody{
		TemplateID: p.TemplateID,
		Page:       p.Page,
		ToUser:     p.ToUser,
		Data:       p.Data,
	}
}

// sendEmployeeRelationMsgBody is the JSON body of the push endpoint.
type sendEmployeeRelationMsgBody struct {
	TemplateID string `json:"template_id"`
	Page       string `json:"page"`
	ToUser     string `json:"touser"`
	Data       string `json:"data"`
}

// SendEmployeeRelationMsgResponse is the reply of the push endpoint.
//
// It carries no message id: success is reported by errcode alone.
type SendEmployeeRelationMsgResponse struct {
	response.WechatResponse
}

// OK reports whether the message was accepted.
func (r *SendEmployeeRelationMsgResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// SendEmployeeRelationMsgError turns the errcode from the reply into an error.
func SendEmployeeRelationMsgError(r *SendEmployeeRelationMsgResponse) error {
	return errors.New(SendEmployeeRelationMsgErrMsg(r.ErrCode))
}

// SendEmployeeRelationMsg pushes a template message to a user who has a 用工关系
// with the mini program, through 小程序商家助手.
//
// The 用工关系 itself is established outside this API -- the user binds it in the
// mini program's 商家助手 flow -- so the openid has to come from a binding that
// already exists.
func (a *Application) SendEmployeeRelationMsg(
	ctx context.Context,
	params SendEmployeeRelationMsgParams,
) (*SendEmployeeRelationMsgResponse, error) {
	if err := CheckSendEmployeeRelationMsgParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, sendEmployeeRelationMsgPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SendEmployeeRelationMsgResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
