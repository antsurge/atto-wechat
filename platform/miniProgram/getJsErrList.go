package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getJsErrListPath lists the JS errors grouped by message and stack.
const getJsErrListPath = "/wxaapi/log/jserr_list"

// Values of errType: where the error came from.
const (
	// JsErrTypeAll is every error, "0".
	JsErrTypeAll = "0"
	// JsErrTypeBusiness is an error in the business code, "1".
	JsErrTypeBusiness = "1"
	// JsErrTypePlugin is an error in a plugin, "2".
	JsErrTypePlugin = "2"
	// JsErrTypeFramework is an error in the framework, "3".
	JsErrTypeFramework = "3"
)

// Values of orderby.
const (
	// JsErrListOrderByUV sorts by the number of users affected, "uv".
	JsErrListOrderByUV = "uv"
	// JsErrListOrderByPV sorts by the number of occurrences, "pv".
	JsErrListOrderByPV = "pv"
)

// Values of desc in this endpoint.
//
// The two JS error endpoints do not agree on this parameter: here ascending is
// "2", in GetJsErrDetail it is "0". Sharing one constant set would silently sort
// half the queries the wrong way.
const (
	// JsErrListOrderDesc sorts descending, "1".
	JsErrListOrderDesc = "1"
	// JsErrListOrderAsc sorts ascending, "2".
	JsErrListOrderAsc = "2"
)

// JsErrListMaxLimit is the documented maximum of limit.
const JsErrListMaxLimit = 30

// JsErrListDefaultLimit is the limit used when none is given.
const JsErrListDefaultLimit = 10

// GetJsErrListParams filters the error list.
//
// Every field is documented as required, and "0" is how "no filter" is spelled
// for the version fields, so empty values are filled with those defaults rather
// than dropped.
type GetJsErrListParams struct {
	// StartTime is the first day, as "2006-01-02".
	StartTime string
	// EndTime is the last day, as "2006-01-02".
	EndTime string

	// AppVersion is the mini program version, "0" for all.
	AppVersion string
	// ErrType is one of the JsErrType* values, "0" for all.
	ErrType string
	// Keyword filters by a term in the error message.
	Keyword string
	// OpenID narrows to one user.
	OpenID string

	// OrderBy is one of the JsErrListOrderBy* values; empty means "uv".
	OrderBy string
	// Desc is one of the JsErrListOrder* values; empty means descending.
	Desc string

	// Offset is the index of the first entry to return.
	Offset int
	// Limit is how many entries to return, at most JsErrListMaxLimit; zero means
	// JsErrListDefaultLimit.
	Limit int
}

// CheckGetJsErrListParams validates params before anything is sent.
func CheckGetJsErrListParams(params GetJsErrListParams) error {
	if err := CheckOperationDateRange("GetJsErrList", params.StartTime, params.EndTime); err != nil {
		return err
	}

	switch params.ErrType {
	case "", JsErrTypeAll, JsErrTypeBusiness, JsErrTypePlugin, JsErrTypeFramework:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrList errType must be 0-3",
		)
	}

	switch params.OrderBy {
	case "", JsErrListOrderByUV, JsErrListOrderByPV:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrList orderby must be uv or pv",
		)
	}

	switch params.Desc {
	case "", JsErrListOrderAsc, JsErrListOrderDesc:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrList desc must be 1 (descending) or 2 (ascending)",
		)
	}

	if params.Offset < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrList offset must not be negative",
		)
	}

	// WeChat refuses a limit above 30 with a parameter error, so the cap is
	// enforced here where the message can say what the limit is.
	if params.Limit < 0 || params.Limit > JsErrListMaxLimit {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetJsErrList limit must be between 0 and 30",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetJsErrListParams) body() *getJsErrListBody {
	limit := p.Limit
	if limit == 0 {
		limit = JsErrListDefaultLimit
	}

	return &getJsErrListBody{
		AppVersion: orZero(p.AppVersion),
		ErrType:    orZero(p.ErrType),
		StartTime:  p.StartTime,
		EndTime:    p.EndTime,
		Keyword:    p.Keyword,
		OpenID:     p.OpenID,
		OrderBy:    orDefault(p.OrderBy, JsErrListOrderByUV),
		Desc:       orDefault(p.Desc, JsErrListOrderDesc),
		Offset:     p.Offset,
		Limit:      limit,
	}
}

// getJsErrListBody is the JSON body of an error list request.
//
// The field names are camelCase, unlike the snake_case of most endpoints: they
// are what this endpoint documents, and a struct tag is the only place they show.
type getJsErrListBody struct {
	AppVersion string `json:"appVersion"`
	ErrType    string `json:"errType"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	Keyword    string `json:"keyword"`
	OpenID     string `json:"openid"`
	OrderBy    string `json:"orderby"`
	Desc       string `json:"desc"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

// orZero returns "0" for an empty filter, the value WeChat uses for "all".
func orZero(value string) string {
	if value == "" {
		return "0"
	}

	return value
}

// orDefault returns fallback for an empty value.
func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

// JsErrItem is one error, with the count of the users and events it hit.
type JsErrItem struct {
	// ErrorMsgMD5 is the md5 of the message, e.g.
	// "53b4825ec4a41d966f88c298c718de80". It is what identifies the error here
	// and what GetJsErrDetail takes as a filter.
	ErrorMsgMD5 string `json:"errorMsgMd5"`
	// ErrorMsg is the error message.
	ErrorMsg string `json:"errorMsg"`
	// UV is how many users hit the error.
	UV int `json:"uv"`
	// PV is how many times it happened.
	PV int `json:"pv"`
	// ErrorStackMD5 is the md5 of the stack.
	ErrorStackMD5 string `json:"errorStackMd5"`
	// ErrorStack is the stack trace.
	ErrorStack string `json:"errorStack"`
	// PVPercent is the share of all error events, as a string.
	PVPercent string `json:"pvPercent"`
	// UVPercent is the share of all affected users, as a string.
	UVPercent string `json:"uvPercent"`
}

// GetJsErrListResponse is the reply of /wxaapi/log/jserr_list.
type GetJsErrListResponse struct {
	// errcode and errmsg on failure.
	response.WechatResponse

	// Success is the field this endpoint sets to report a completed search.
	Success bool `json:"success"`
	// OpenID echoes the openid filter.
	OpenID string `json:"openid"`
	// Data is the page of errors.
	Data []JsErrItem `json:"data"`
	// TotalCount is how many errors match, not how many are in Data.
	TotalCount int `json:"totalCount"`
}

// OK reports whether the search ran.
//
// Both signals are required: this endpoint reports in success rather than only
// in errcode, and a reply can carry success=false.
func (r *GetJsErrListResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Success
}

// GetJsErrListError turns the errcode from the reply into an error.
func GetJsErrListError(r *GetJsErrListResponse) error {
	return operationError(r.ErrCode)
}

// GetJsErrList lists the JS errors of the mini program.
//
// The identity of an error is its md5 pair, not its index: pass both to
// GetJsErrDetail to see who hit it, on which devices.
func (a *Application) GetJsErrList(
	ctx context.Context,
	params GetJsErrListParams,
) (*GetJsErrListResponse, error) {
	if err := CheckGetJsErrListParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getJsErrListPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetJsErrListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// JsErrTypeMsg returns the Chinese name of an errType value.
func JsErrTypeMsg(errType string) string {
	switch errType {
	case JsErrTypeBusiness:
		return "业务代码错误"
	case JsErrTypePlugin:
		return "插件错误"
	case JsErrTypeFramework:
		return "系统框架错误"
	default:
		return "全部"
	}
}
