package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// generateURLLinkPath turns a page into an https://wxaurl.cn/... URL Link that
// can be opened from outside WeChat.
const (
	generateURLLinkPath          = "/wxa/generate_urllink"
	GenerateURLLinkDefaultErrMsg = "系统错误，请稍后再试"
)

// GenerateURLLinkErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/generate_urllink, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GenerateURLLinkErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40002: "暂无生成权限：个人主体小程序无权限，或 NFC 能力的小程序未申请权限",
	40013: "生成权限被封禁",
	40165: "path 填写错误",
	40212: "query 填写错误，query 格式遵循 URL 标准，即 k1=v1&k2=v2",
	44990: "调用频率过快，超过 100 次/秒",
	44993: "单天生成的加密 URL Scheme + URL Link 数量超过上限 50 万",
	85079: "小程序没有线上版本，即小程序尚未发布",
	85088: "未开通云开发",
	85401: "expire_time 填写错误，时间间隔需大于 1 分钟且小于 30 天",
	85402: "env_version 填写错误",
	85406: "URL Scheme（加密+明文）/加密 URL Link 单天累加访问次数超过上限",
}

// CloudBase configures the cloud hosting page that relays a URL Link, and is
// also the cloud_base object a query reply carries back.
//
// By default WeChat sends the user through its own intermediate H5 page. With
// CloudBase the mini program can serve that page itself, which only works when
// the cloud environment exists (otherwise WeChat answers 85088). Env is the only
// required field.
type CloudBase struct {
	// Env is the cloud environment ID. Required.
	Env string `json:"env"`

	// Domain is the custom domain of the static website. Empty falls back to
	// the default domain.
	Domain string `json:"domain,omitempty"`

	// Path is the H5 page inside the static website. It must not carry a
	// query.
	Path string `json:"path,omitempty"`

	// Query is the query that H5 page receives, limited like a page query:
	// 1024 characters drawn from digits, ASCII letters and
	// generateSchemeQueryChars.
	Query string `json:"query,omitempty"`

	// ResourceAppID is required when a third-party platform creates the
	// environment on a mini program's behalf.
	ResourceAppID string `json:"resource_appid,omitempty"`
}

// GenerateURLLinkParams describes one URL Link.
//
// Unlike GenerateSchemeParams the page fields are not wrapped in
// jump_wxa: generate_urllink takes path, query and env_version at the top
// level.
//
// Without IsExpire the link never expires. With it, exactly one of ExpireTime
// and ExpireInterval is required, chosen by ExpireType.
type GenerateURLLinkParams struct {
	// Path is the page the link opens. It must exist in a published mini
	// program. An empty path opens the home page.
	Path string

	// Query is the query the page receives.
	Query string

	// EnvVersion is release (default), trial or develop. It only takes
	// effect when the link is opened outside WeChat.
	EnvVersion string

	// IsExpire makes the link expire. False, the default, keeps it
	// permanent.
	//
	// WeChat's HTTPS parameter table no longer lists is_expire, but its cloud
	// call example still sends it and the server keeps honouring it: omitted
	// means a permanent link cap of 100,000, which fits the default below.
	IsExpire bool

	// ExpireType selects how expiry is expressed: 0 (the default) reads
	// ExpireTime, 1 reads ExpireInterval. It is only consulted when IsExpire
	// is true.
	ExpireType int

	// ExpireTime is the Unix timestamp the link dies at, required when
	// IsExpire is true and ExpireType is 0. WeChat limits it to at most 30
	// days from now, and answers 85401 otherwise.
	ExpireTime int64

	// ExpireInterval is the number of days the link lives for, required when
	// IsExpire is true and ExpireType is 1. WeChat caps it at 30 days.
	ExpireInterval int

	// CloudBase points the link at a cloud hosted relay page. Nil uses the
	// official H5 page.
	CloudBase *CloudBase
}

// CheckGenerateURLLinkParams validates params before anything is sent to WeChat.
func CheckGenerateURLLinkParams(params GenerateURLLinkParams) error {
	if err := checkPathQuery(params.Path, params.Query); err != nil {
		return err
	}

	if err := checkEnvVersion(params.EnvVersion); err != nil {
		return err
	}

	if err := checkExpire(params.IsExpire, params.ExpireType, params.ExpireTime, params.ExpireInterval); err != nil {
		return err
	}

	if params.CloudBase != nil {
		if params.CloudBase.Env == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: cloud_base.env is required",
			)
		}

		if err := checkPathQuery(params.CloudBase.Path, params.CloudBase.Query); err != nil {
			return err
		}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GenerateURLLinkParams) body() *generateURLLinkBody {
	return &generateURLLinkBody{
		Path:           p.Path,
		Query:          p.Query,
		EnvVersion:     p.EnvVersion,
		IsExpire:       p.IsExpire,
		ExpireType:     p.ExpireType,
		ExpireTime:     p.ExpireTime,
		ExpireInterval: p.ExpireInterval,
		CloudBase:      p.CloudBase,
	}
}

// generateURLLinkBody is the JSON body of /wxa/generate_urllink.
//
// The page fields sit at the top level here, not under jump_wxa.
type generateURLLinkBody struct {
	Path           string     `json:"path,omitempty"`
	Query          string     `json:"query,omitempty"`
	EnvVersion     string     `json:"env_version,omitempty"`
	IsExpire       bool       `json:"is_expire,omitempty"`
	ExpireType     int        `json:"expire_type,omitempty"`
	ExpireTime     int64      `json:"expire_time,omitempty"`
	ExpireInterval int        `json:"expire_interval,omitempty"`
	CloudBase      *CloudBase `json:"cloud_base,omitempty"`
}

// GenerateURLLinkResponse is the reply of /wxa/generate_urllink.
type GenerateURLLinkResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	UrlLink string `json:"url_link"`
}

// OK reports whether the reply carries a URL Link.
func (r *GenerateURLLinkResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.UrlLink != ""
}

// GenerateURLLinkErrMsg returns the description of a generateURLLink error.
func GenerateURLLinkErrMsg(code int) string {
	if msg, ok := GenerateURLLinkErrCodeMap[code]; ok {
		return msg
	}
	return GenerateURLLinkDefaultErrMsg
}

// GenerateURLLinkError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GenerateURLLinkErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func GenerateURLLinkError(r *GenerateURLLinkResponse) error {
	return errors.New(GenerateURLLinkErrMsg(r.ErrCode))
}

// GenerateURLLink returns an https://wxaurl.cn/... URL Link for params.
//
// A URL Link is the https:// counterpart of a scheme: it fits a SMS, an email
// or an external web page and opens the mini program when tapped. WeChat
// forwards the user through an intermediate H5 page first, which CloudBase can
// replace with one the mini program hosts.
//
// Only published mini programs can be targeted, and only mainland non-personal
// ones may generate links at all: WeChat answers 85079 and 40002 otherwise.
// Generating is capped at 500,000 links per day, shared with URL Scheme, and
// opening them at 3,000,000 times per day.
func (a *Application) GenerateURLLink(
	ctx context.Context,
	params GenerateURLLinkParams,
) (*GenerateURLLinkResponse, error) {
	if err := CheckGenerateURLLinkParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, generateURLLinkPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GenerateURLLinkResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
