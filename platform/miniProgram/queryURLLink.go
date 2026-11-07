package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// queryURLLinkPath looks up a generated URL Link and reports the daily visit
// quota with query_type 1.
const (
	queryURLLinkPath          = "/wxa/query_urllink"
	QueryURLLinkDefaultErrMsg = "系统错误，请稍后再试"

	// queryURLLinkTypeInfo asks for the URL Link details, the documented
	// default.
	queryURLLinkTypeInfo = 0

	// queryURLLinkTypeQuota asks for the remaining visits of the day.
	queryURLLinkTypeQuota = 1
)

// QueryURLLinkErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/query_urllink, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var QueryURLLinkErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40097: "参数错误",
	85403: "url_link 不存在",
}

// URLLinkInfo is the url_link_info object of a query reply.
//
// ExpireTime is zero for a permanent link. CloudBase mirrors what the link was
// generated with, and is absent when the official H5 page relays it.
type URLLinkInfo struct {
	AppID      string     `json:"appid"`
	Path       string     `json:"path"`
	Query      string     `json:"query"`
	CreateTime int64      `json:"create_time"`
	ExpireTime int64      `json:"expire_time"`
	EnvVersion string     `json:"env_version"`
	CloudBase  *CloudBase `json:"cloud_base,omitempty"`
}

// QueryURLLinkParams describes one lookup.
//
// URLLink is required only for the default query type: asking for the quota does
// not need a link.
type QueryURLLinkParams struct {
	// URLLink is the encrypted URL Link to look up ("https://wxaurl.cn/...").
	URLLink string

	// QueryType selects what to look up: 0 asks for the link details, which
	// is the documented default, and 1 asks for the visits left today.
	QueryType int
}

// CheckQueryURLLinkParams validates params before anything is sent to WeChat.
//
// Only the combinations WeChat rejects anyway are checked: an unknown
// query_type, and a missing URL Link when details are requested. The
// documentation states no length limit for url_link, so none is enforced here.
func CheckQueryURLLinkParams(params QueryURLLinkParams) error {
	switch params.QueryType {
	case queryURLLinkTypeInfo:
		if params.URLLink == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: url_link must not be empty when query_type is 0",
			)
		}
	case queryURLLinkTypeQuota:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: query_type must be 0 or 1",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p QueryURLLinkParams) body() *queryURLLinkBody {
	return &queryURLLinkBody{
		URLLink:   p.URLLink,
		QueryType: p.QueryType,
	}
}

// queryURLLinkBody is the JSON body of /wxa/query_urllink.
type queryURLLinkBody struct {
	URLLink   string `json:"url_link,omitempty"`
	QueryType int    `json:"query_type,omitempty"`
}

// QueryURLLinkResponse is the reply of /wxa/query_urllink.
//
// Exactly one of the two business objects is filled in, depending on QueryType:
// URLLinkInfo for a details lookup, QuotaInfo for a quota lookup. Both are
// pointers so callers can tell "not requested" from "requested but empty".
type QueryURLLinkResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	URLLinkInfo *URLLinkInfo `json:"url_link_info,omitempty"`
	QuotaInfo   *QuotaInfo   `json:"quota_info,omitempty"`
}

// OK reports whether the reply carries the information that was asked for.
func (r *QueryURLLinkResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && (r.URLLinkInfo != nil || r.QuotaInfo != nil)
}

// QueryURLLinkErrMsg returns the description of a queryUrlLink error.
func QueryURLLinkErrMsg(code int) string {
	if msg, ok := QueryURLLinkErrCodeMap[code]; ok {
		return msg
	}
	return QueryURLLinkDefaultErrMsg
}

// QueryURLLinkError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// QueryURLLinkErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func QueryURLLinkError(r *QueryURLLinkResponse) error {
	return errors.New(QueryURLLinkErrMsg(r.ErrCode))
}

// QueryURLLink looks up a URL Link that was generated earlier.
//
// With QueryType 0 it answers the appid, page, query and version the link opens,
// which is how a caller can map an opaque "https://wxaurl.cn/..." link back to
// its page. With QueryType 1 it answers how many visits are left today, and no
// link is needed.
//
// A link that WeChat does not know, including one that expired, answers 85403.
//
// The endpoint takes no fresh link here: generating is a different call, and
// this one only reads back what was generated before.
func (a *Application) QueryURLLink(
	ctx context.Context,
	params QueryURLLinkParams,
) (*QueryURLLinkResponse, error) {
	if err := CheckQueryURLLinkParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, queryURLLinkPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result QueryURLLinkResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
