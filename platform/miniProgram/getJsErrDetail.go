package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getJsErrDetailPath lists the occurrences of one JS error.
const getJsErrDetailPath = "/wxaapi/log/jserr_detail"

// Values of desc in this endpoint.
//
// It reads "0" for ascending here and "2" in GetJsErrList: the two endpoints
// share the parameter name but not its values, so the constants are kept apart.
const (
	// JsErrDetailOrderAsc sorts ascending, "0".
	JsErrDetailOrderAsc = "0"
	// JsErrDetailOrderDesc sorts descending, "1".
	JsErrDetailOrderDesc = "1"
)

// Values of osName.
const (
	// JsErrOSAll is every system, "0".
	JsErrOSAll = "0"
	// JsErrOSAndroid is Android, "1".
	JsErrOSAndroid = "1"
	// JsErrOSIOS is iOS, "2".
	JsErrOSIOS = "2"
	// JsErrOSOther is anything else, "3".
	JsErrOSOther = "3"
)

// jsErrMD5Length is the length of the md5 hex digest both filters take.
const jsErrMD5Length = 32

// GetJsErrDetailParams selects the error and the occurrences to list.
type GetJsErrDetailParams struct {
	// StartTime is the first day, as "2006-01-02".
	StartTime string
	// EndTime is the last day, as "2006-01-02".
	EndTime string

	// ErrorMsgMD5 is the errorMsgMd5 of the error, from GetJsErrList.
	ErrorMsgMD5 string
	// ErrorStackMD5 is the errorStackMd5 of the error, from GetJsErrList.
	ErrorStackMD5 string

	// AppVersion is the mini program version, "0" for all.
	AppVersion string
	// SDKVersion is the base library version, "0" for all.
	SDKVersion string
	// ClientVersion is the WeChat client version, "0" for all.
	ClientVersion string
	// OSName is one of the JsErrOS* values, "0" for all.
	OSName string
	// OpenID narrows to one user.
	OpenID string

	// Offset is the index of the first entry to return.
	Offset int
	// Limit is how many entries to return; zero leaves the default to WeChat.
	Limit int
	// Desc is one of the JsErrDetailOrder* values; empty means descending.
	Desc string
}

// CheckGetJsErrDetailParams validates params before anything is sent.
func CheckGetJsErrDetailParams(params GetJsErrDetailParams) error {
	if err := CheckOperationDateRange("GetJsErrDetail", params.StartTime, params.EndTime); err != nil {
		return err
	}

	// A wrong md5 is answered with an empty list, which reads as "this error
	// never happened": the md5 pair is the only key into the detail, so it is
	// checked here where the message can say so.
	for _, md5 := range []struct {
		name  string
		value string
	}{
		{name: "errorMsgMd5", value: params.ErrorMsgMD5},
		{name: "errorStackMd5", value: params.ErrorStackMD5},
	} {
		if !isHexDigest(md5.value) {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: GetJsErrDetail "+md5.name+" must be a 32-character hex digest from GetJsErrList",
			)
		}
	}

	switch params.OSName {
	case "", JsErrOSAll, JsErrOSAndroid, JsErrOSIOS, JsErrOSOther:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrDetail osName must be 0-3",
		)
	}

	switch params.Desc {
	case "", JsErrDetailOrderAsc, JsErrDetailOrderDesc:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrDetail desc must be 0 (ascending) or 1 (descending)",
		)
	}

	if params.Offset < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrDetail offset must not be negative",
		)
	}

	if params.Limit < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrDetail limit must not be negative",
		)
	}

	return nil
}

// isHexDigest reports whether value is a 32-character hex string.
func isHexDigest(value string) bool {
	if len(value) != jsErrMD5Length {
		return false
	}

	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}

	return true
}

// body renders params as the JSON body the endpoint expects.
func (p GetJsErrDetailParams) body() *getJsErrDetailBody {
	return &getJsErrDetailBody{
		StartTime:     p.StartTime,
		EndTime:       p.EndTime,
		ErrorMsgMD5:   p.ErrorMsgMD5,
		ErrorStackMD5: p.ErrorStackMD5,
		AppVersion:    orZero(p.AppVersion),
		SDKVersion:    orZero(p.SDKVersion),
		ClientVersion: orZero(p.ClientVersion),
		OSName:        orZero(p.OSName),
		OpenID:        p.OpenID,
		Offset:        p.Offset,
		Limit:         p.Limit,
		Desc:          orDefault(p.Desc, JsErrDetailOrderDesc),
	}
}

// getJsErrDetailBody is the JSON body of an error detail request.
type getJsErrDetailBody struct {
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	ErrorMsgMD5   string `json:"errorMsgMd5"`
	ErrorStackMD5 string `json:"errorStackMd5"`
	AppVersion    string `json:"appVersion"`
	SDKVersion    string `json:"sdkVersion"`
	ClientVersion string `json:"clientVersion"`
	OSName        string `json:"osName"`
	OpenID        string `json:"openid"`
	Offset        int    `json:"offset"`
	Limit         int    `json:"limit"`
	Desc          string `json:"desc"`
}

// JsErrDetailItem is one occurrence of an error.
//
// The field names on the wire are inconsistent -- Count and TimeStamp and
// ClientVersion are capitalised, the rest are not -- and openId differs from the
// openid of every other endpoint. The tags keep them verbatim, because a wrong
// case is a silently empty field.
type JsErrDetailItem struct {
	// Count is how many times this exact occurrence happened, as a string.
	Count string `json:"Count"`
	// SDKVersion is the base library version.
	SDKVersion string `json:"sdkVersion"`
	// ClientVersion is the WeChat client version.
	ClientVersion string `json:"ClientVersion"`
	// ErrorStackMD5 is the md5 of the stack.
	ErrorStackMD5 string `json:"errorStackMd5"`
	// TimeStamp is when it happened, "2021-01-25 16:36:39", as a string.
	TimeStamp string `json:"TimeStamp"`
	// AppVersion is the mini program version.
	AppVersion string `json:"appVersion"`
	// ErrorMsgMD5 is the md5 of the message.
	ErrorMsgMD5 string `json:"errorMsgMd5"`
	// ErrorMsg is the error message.
	ErrorMsg string `json:"errorMsg"`
	// ErrorStack is the stack trace.
	ErrorStack string `json:"errorStack"`
	// Ds is the date, "2021-01-25".
	Ds string `json:"Ds"`
	// OSName is the system, "1" for Android and "2" for iOS.
	OSName string `json:"OsName"`
	// OpenID is the user who hit the error. The tag is openId, not openid.
	OpenID string `json:"openId"`
	// PluginVersion is the plugin version, "0" when no plugin was involved.
	PluginVersion string `json:"pluginversion"`
	// AppID is the mini program appid.
	AppID string `json:"appId"`
	// DeviceModel is the device model, e.g. "Redmi Note 5Aarm64-v8a".
	DeviceModel string `json:"DeviceModel"`
	// Source is undocumented and still empty in every example.
	Source string `json:"source"`
	// Route is undocumented and still empty in every example.
	Route string `json:"route"`
	// Uin is undocumented and still empty in every example.
	Uin string `json:"Uin"`
	// Nickname is the user's nickname, empty unless the user shared it.
	Nickname string `json:"nickname"`
}

// GetJsErrDetailResponse is the reply of /wxaapi/log/jserr_detail.
type GetJsErrDetailResponse struct {
	// errcode and errmsg on failure.
	response.WechatResponse

	// Success is the field this endpoint sets to report a completed search.
	Success bool `json:"success"`
	// OpenID echoes the openid filter.
	OpenID string `json:"openid"`
	// Data is the page of occurrences.
	Data []JsErrDetailItem `json:"data"`
	// TotalCount is how many occurrences match, not how many are in Data.
	TotalCount int `json:"totalCount"`
}

// OK reports whether the search ran.
func (r *GetJsErrDetailResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Success
}

// GetJsErrDetailError turns the errcode from the reply into an error.
func GetJsErrDetailError(r *GetJsErrDetailResponse) error {
	return operationError(r.ErrCode)
}

// GetJsErrDetail lists the occurrences of one JS error.
//
// The md5 pair comes from GetJsErrList. Note that limit has no documented cap
// here while its sibling caps at 30, and that the two endpoints disagree on the
// value of desc for ascending.
func (a *Application) GetJsErrDetail(
	ctx context.Context,
	params GetJsErrDetailParams,
) (*GetJsErrDetailResponse, error) {
	if err := CheckGetJsErrDetailParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getJsErrDetailPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetJsErrDetailResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
