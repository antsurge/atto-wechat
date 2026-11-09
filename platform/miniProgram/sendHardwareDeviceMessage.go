package miniProgram

import (
	"context"
	"strings"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// sendHardwareDeviceMessagePath pushes a subscribe message to the users of devices.
const sendHardwareDeviceMessagePath = "/cgi-bin/message/device/subscribe/send"

// Values of miniprogram_state: which build the message links into.
const (
	// HardwareDeviceMPStateDeveloper is the development build.
	HardwareDeviceMPStateDeveloper = "developer"
	// HardwareDeviceMPStateTrial is the trial build.
	HardwareDeviceMPStateTrial = "trial"
	// HardwareDeviceMPStateFormal is the release build, the default.
	HardwareDeviceMPStateFormal = "formal"
)

// Values of lang: the language of the message.
const (
	HardwareDeviceLangZHCN = "zh_CN"
	HardwareDeviceLangENUS = "en_US"
	HardwareDeviceLangZHHK = "zh_HK"
	HardwareDeviceLangZHTW = "zh_TW"
)

// HardwareDeviceMsgValue is one entry of a device message's data.
type HardwareDeviceMsgValue struct {
	// Value is the template parameter value, an enum value of the template.
	Value string `json:"value"`
}

// SendHardwareDeviceMessageParams is the message to push.
type SendHardwareDeviceMessageParams struct {
	// TemplateID is the subscribe template to fill.
	TemplateID string
	// ModelID is the device model id, obtained when registering the device.
	ModelID string
	// SN is the serial number of the device that triggered the message.
	SN string
	// ToOpenIDList are the users to notify.
	ToOpenIDList []string

	// Data fills the template: key is the template field, value its enum value.
	//
	// It is an object on the wire -- the documented shape is
	// {"key1":{"value":"xxx"}} -- unlike the employee-relation message, whose
	// data is a JSON-encoded string.
	Data map[string]HardwareDeviceMsgValue

	// Page is where the card leads, inside this mini program, parameters allowed
	// (e.g. "index?foo=bar"). Empty means the card does not navigate.
	Page string
	// MiniProgramState is one of the HardwareDeviceMPState* values; empty means
	// formal.
	MiniProgramState string
	// Lang is one of the HardwareDeviceLang* values; empty means zh_CN.
	Lang string
}

// CheckSendHardwareDeviceMessageParams validates params before anything is sent.
func CheckSendHardwareDeviceMessageParams(params SendHardwareDeviceMessageParams) error {
	if params.TemplateID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendHardwareDeviceMessage template_id is required",
		)
	}

	if err := CheckHardwareDevice(HardwareDevice{ModelID: params.ModelID, SN: params.SN}); err != nil {
		return err
	}

	if len(params.ToOpenIDList) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendHardwareDeviceMessage to_openid_list is required",
		)
	}

	// A message with an empty data object has nothing to fill the template with
	// and fails with 47003, one field at a time.
	if len(params.Data) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendHardwareDeviceMessage data is required",
		)
	}

	switch params.MiniProgramState {
	case "", HardwareDeviceMPStateDeveloper, HardwareDeviceMPStateTrial, HardwareDeviceMPStateFormal:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendHardwareDeviceMessage miniprogram_state must be developer, trial or formal",
		)
	}

	switch params.Lang {
	case "", HardwareDeviceLangZHCN, HardwareDeviceLangENUS, HardwareDeviceLangZHHK, HardwareDeviceLangZHTW:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendHardwareDeviceMessage lang must be zh_CN, en_US, zh_HK or zh_TW",
		)
	}

	// The page has to be a page of this mini program. A URL is the mistake worth
	// catching: it is accepted by nothing and comes back as 41030.
	if strings.Contains(params.Page, "://") {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: SendHardwareDeviceMessage page must be a page path, not a URL",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SendHardwareDeviceMessageParams) body() *sendHardwareDeviceMessageBody {
	return &sendHardwareDeviceMessageBody{
		TemplateID:       p.TemplateID,
		SN:               p.SN,
		Page:             p.Page,
		ToOpenIDList:     p.ToOpenIDList,
		MiniProgramState: p.MiniProgramState,
		ModelID:          p.ModelID,
		Data:             p.Data,
		Lang:             p.Lang,
	}
}

// sendHardwareDeviceMessageBody is the JSON body of a device message.
//
// modelId is camelCase here while every other field is snake_case: that is the
// documented wire name, and a struct tag is the only place it may show.
type sendHardwareDeviceMessageBody struct {
	TemplateID       string                            `json:"template_id"`
	SN               string                            `json:"sn"`
	Page             string                            `json:"page,omitempty"`
	ToOpenIDList     []string                          `json:"to_openid_list"`
	MiniProgramState string                            `json:"miniprogram_state,omitempty"`
	ModelID          string                            `json:"modelId"`
	Data             map[string]HardwareDeviceMsgValue `json:"data"`
	Lang             string                            `json:"lang,omitempty"`
}

// SendHardwareDeviceMessageResponse is the reply of the device message endpoint.
type SendHardwareDeviceMessageResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether the message was accepted for delivery.
//
// Accepted is all it says: subscription messages can still be dropped for a user
// who unsubscribed, which the per-user result is not reported here at all.
func (r *SendHardwareDeviceMessageResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// SendHardwareDeviceMessageError turns the errcode from the reply into an error.
func SendHardwareDeviceMessageError(r *SendHardwareDeviceMessageResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// SendHardwareDeviceMessage pushes a subscribe message for one device to the users
// who subscribed to it.
//
// This is the hardware path of the same mechanism as SendEmployeeRelationMsg, but
// the data here is an object, not a JSON-encoded string.
func (a *Application) SendHardwareDeviceMessage(
	ctx context.Context,
	params SendHardwareDeviceMessageParams,
) (*SendHardwareDeviceMessageResponse, error) {
	if err := CheckSendHardwareDeviceMessageParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, sendHardwareDeviceMessagePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SendHardwareDeviceMessageResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
