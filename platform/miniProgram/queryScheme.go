package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// querySchemePath looks up a scheme code, encrypted or plain, and reports the
// daily visit quota with query_type 1.
const (
	querySchemePath          = "/wxa/queryscheme"
	QuerySchemeDefaultErrMsg = "系统错误，请稍后再试"

	// querySchemeTypeInfo asks for the scheme details, the documented default.
	querySchemeTypeInfo = 0

	// querySchemeTypeQuota asks for the remaining visits of the day.
	querySchemeTypeQuota = 1
)

// QuerySchemeErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/queryscheme, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var QuerySchemeErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40002: "暂无生成权限：非个人主体小程序无权限，或未申请 NFC 能力",
	40097: "参数错误",
	40165: "path 填写错误",
	40212: "query 填写错误，query 格式遵循 URL 标准，即 k1=v1&k2=v2",
	85402: "env_version 填写错误",
	85403: "scheme 或 url link 不存在",
	85405: "小程序 appid 或 path 未开启明文 scheme",
}

// SchemeInfo is the scheme_info object of a query reply, or of a reply that
// generated a scheme.
//
// An encrypted scheme always carries CreateTime and ExpireTime; a plain scheme
// may leave both at zero because the fields are optional there.
type SchemeInfo struct {
	AppID      string `json:"appid"`
	Path       string `json:"path"`
	Query      string `json:"query"`
	CreateTime int64  `json:"create_time"`
	ExpireTime int64  `json:"expire_time"`
	EnvVersion string `json:"env_version"`
}

// QuotaInfo is the quota_info object of a query reply.
//
// The same quota covers URL Scheme links, both encrypted and plain, and URL
// Link links, so either query endpoint reports the same figure.
type QuotaInfo struct {
	// RemainVisitQuota is the remaining visits today of URL Scheme and URL
	// Link links combined.
	RemainVisitQuota int64 `json:"remain_visit_quota"`
}

// QuerySchemeParams describes one lookup.
//
// Scheme is required only for the default query type: asking for the quota does
// not need a scheme.
type QuerySchemeParams struct {
	// Scheme is the scheme to look up. Both encrypted
	// ("weixin://dl/business/?t=...") and plain
	// ("weixin://dl/business/?appid=...") schemes are accepted.
	Scheme string

	// QueryType selects what to look up: 0 asks for the scheme details, which
	// is the documented default, and 1 asks for the visits left today.
	QueryType int
}

// CheckQuerySchemeParams validates params before anything is sent to WeChat.
//
// Only the combination that WeChat rejects anyway is checked: an unknown
// query_type, and a missing scheme when details are requested. The documentation
// states no length limit for scheme, so none is enforced here.
func CheckQuerySchemeParams(params QuerySchemeParams) error {
	switch params.QueryType {
	case querySchemeTypeInfo:
		if params.Scheme == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: scheme must not be empty when query_type is 0",
			)
		}
	case querySchemeTypeQuota:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: query_type must be 0 or 1",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p QuerySchemeParams) body() *querySchemeBody {
	return &querySchemeBody{
		Scheme:    p.Scheme,
		QueryType: p.QueryType,
	}
}

// querySchemeBody is the JSON body of /wxa/queryscheme.
type querySchemeBody struct {
	Scheme    string `json:"scheme,omitempty"`
	QueryType int    `json:"query_type,omitempty"`
}

// QuerySchemeResponse is the reply of /wxa/queryscheme.
//
// Exactly one of the two business objects is filled in, depending on QueryType:
// SchemeInfo for a details lookup, QuotaInfo for a quota lookup. Both are
// pointers so callers can tell "not requested" from "requested but empty".
type QuerySchemeResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	SchemeInfo *SchemeInfo `json:"scheme_info,omitempty"`
	QuotaInfo  *QuotaInfo  `json:"quota_info,omitempty"`
}

// OK reports whether the reply carries the information that was asked for.
func (r *QuerySchemeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && (r.SchemeInfo != nil || r.QuotaInfo != nil)
}

// QuerySchemeErrMsg returns the description of a queryScheme error.
func QuerySchemeErrMsg(code int) string {
	if msg, ok := QuerySchemeErrCodeMap[code]; ok {
		return msg
	}
	return QuerySchemeDefaultErrMsg
}

// QuerySchemeError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// QuerySchemeErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func QuerySchemeError(r *QuerySchemeResponse) error {
	return errors.New(QuerySchemeErrMsg(r.ErrCode))
}

// QueryScheme looks up a scheme code that was generated earlier.
//
// With QueryType 0 it answers the appid, page, query and version the scheme
// opens, which is how a caller can map an opaque "?t=..." scheme back to its
// page. With QueryType 1 it answers how many visits are left today, and no
// scheme is needed.
//
// A scheme that WeChat does not know, including one that expired, answers 85403.
//
// The endpoint takes no fresh scheme here: generating is a different call, and
// this one only reads back what was generated before.
func (a *Application) QueryScheme(
	ctx context.Context,
	params QuerySchemeParams,
) (*QuerySchemeResponse, error) {
	if err := CheckQuerySchemeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, querySchemePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result QuerySchemeResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
