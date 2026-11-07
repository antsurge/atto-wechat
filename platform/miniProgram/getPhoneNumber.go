package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPhoneNumberPath is the phone number endpoint: it exchanges a temporary
// code issued by the getPhoneNumber button callback for the user's phone.
const (
	getPhoneNumberPath          = "/wxa/business/getuserphonenumber"
	GetPhoneNumberDefaultErrMsg = "系统错误，请稍后再试"
)

// GetPhoneNumberErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/business/getuserphonenumber, keyed by the errcode as it appears in the
// reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GetPhoneNumberErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40029: "code 无效",
	40226: "用户需要短信验证",
	45011: "API 调用太频繁，请稍候再试",
}

// PhoneWatermark is the watermark attached to the phone number data, used to
// prove that the reply really came from WeChat for the expected appid.
type PhoneWatermark struct {
	// Timestamp is when WeChat produced the data, in seconds.
	Timestamp int64 `json:"timestamp"`
	// AppID is the mini program the data belongs to.
	AppID string `json:"appid"`
}

// GetPhoneNumberResponse is the reply of /wxa/business/getuserphonenumber.
//
// The structure keeps the two kinds of parameters apart: the embedded common
// header is what every WeChat API returns (errcode/errmsg), while the fields
// below are specific to this endpoint. WeChat nests them under phone_info, but
// UnmarshalJSON lifts them onto the response, so callers read r.PurePhoneNumber
// instead of r.PhoneInfo.PurePhoneNumber.
type GetPhoneNumberResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters, flattened from phone_info.
	//
	// PhoneNumber may carry the country prefix, e.g. "+8613800000000";
	// PurePhoneNumber is the number without it; CountryCode is "86" for
	// mainland China; Watermark proves the data came from WeChat for the
	// configured appid.
	PhoneNumber     string          `json:"phoneNumber"`
	PurePhoneNumber string          `json:"purePhoneNumber"`
	CountryCode     string          `json:"countryCode"`
	Watermark       *PhoneWatermark `json:"watermark,omitempty"`
}

// UnmarshalJSON decodes the reply, lifting the nested phone_info object onto
// the response fields.
func (r *GetPhoneNumberResponse) UnmarshalJSON(data []byte) error {
	var payload struct {
		response.WechatResponse
		PhoneInfo struct {
			PhoneNumber     string          `json:"phoneNumber"`
			PurePhoneNumber string          `json:"purePhoneNumber"`
			CountryCode     string          `json:"countryCode"`
			Watermark       *PhoneWatermark `json:"watermark"`
		} `json:"phone_info"`
	}

	if err := codec.Unmarshal(data, &payload); err != nil {
		return err
	}

	r.WechatResponse = payload.WechatResponse
	r.PhoneNumber = payload.PhoneInfo.PhoneNumber
	r.PurePhoneNumber = payload.PhoneInfo.PurePhoneNumber
	r.CountryCode = payload.PhoneInfo.CountryCode
	r.Watermark = payload.PhoneInfo.Watermark

	return nil
}

// OK reports whether the response contains a usable phone number.
func (r *GetPhoneNumberResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.PurePhoneNumber != ""
}

// GetPhoneNumberErrMsg returns the description of a getPhoneNumber error.
func GetPhoneNumberErrMsg(code int) string {
	if msg, ok := GetPhoneNumberErrCodeMap[code]; ok {
		return msg
	}
	return GetPhoneNumberDefaultErrMsg
}

// GetPhoneNumberError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetPhoneNumberErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetPhoneNumberError(r *GetPhoneNumberResponse) error {
	return errors.New(GetPhoneNumberErrMsg(r.ErrCode))
}

// GetPhoneNumber exchanges the code returned by the getPhoneNumber button
// callback for the user's phone number.
//
// The code is single use and expires after five minutes; it is bound to the
// appid that obtained it, so the same appid must call this endpoint. The phone
// number is personal data: do not log it or return it to the client unless the
// client asked for it.
//
// Compare Watermark.AppID with the configured appid before trusting the result,
// otherwise a reply from another mini program would be accepted.
func (a *Application) GetPhoneNumber(
	ctx context.Context,
	code string,
) (*GetPhoneNumberResponse, error) {
	if code == "" {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: code must not be empty")
	}

	resp, err := a.api.PostJSON(ctx, getPhoneNumberPath, nil, map[string]string{
		"code": code,
	})
	if err != nil {
		return nil, err
	}

	var result GetPhoneNumberResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
