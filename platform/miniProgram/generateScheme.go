package miniProgram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// generateSchemePath turns a page into a weixin:// scheme link that can be
// opened from outside WeChat.
const (
	generateSchemePath          = "/wxa/generatescheme"
	GenerateSchemeDefaultErrMsg = "系统错误，请稍后再试"

	// generateSchemeQueryMaxLen is the documented limit of the page query.
	generateSchemeQueryMaxLen = 1024

	// generateSchemeQueryChars are the non-alphanumeric characters a page
	// query may contain.
	generateSchemeQueryChars = "!#$&'()*+,/:;=?@-.~%"

	// generateSchemeExpireMaxDays caps both expire_time and expire_interval:
	// a generated scheme may live 30 days at most.
	generateSchemeExpireMaxDays = 30

	// generateSchemeExpireTypeTime expires at an absolute moment.
	generateSchemeExpireTypeTime = 0

	// generateSchemeExpireTypeInterval expires after a number of days.
	generateSchemeExpireTypeInterval = 1
)

// GenerateSchemeErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/generatescheme, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GenerateSchemeErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40002: "暂无生成权限：个人主体小程序无权限，或 NFC 能力的小程序未申请权限",
	40013: "生成权限被封禁",
	40165: "path 填写错误",
	40212: "query 填写错误，query 格式遵循 URL 标准，即 k1=v1&k2=v2",
	44990: "调用频率过快，超过 100 次/秒",
	44993: "单天生成的加密 URL Scheme + URL Link 数量超过上限 50 万",
	85079: "小程序没有线上版本，即小程序尚未发布",
	85401: "expire_time 填写错误，时间间隔需大于 1 分钟且小于 30 天",
	85402: "env_version 填写错误",
	85406: "URL Scheme（加密+明文）/加密 URL Link 单天累加访问次数超过上限",
}

// JumpWXA is the jump_wxa object shared by the scheme generating endpoints.
//
// Path must not carry a query: every business parameter belongs in Query. The
// documented examples write the path with a leading slash
// ("/pages/index/index"), unlike the QR code endpoints which take it without
// one, so it is passed through as given.
type JumpWXA struct {
	// Path is the page the scheme opens. It must exist in a published
	// mini program. An empty path opens the home page.
	Path string

	// Query is the query the page receives, limited to 1024 characters and
	// to digits, ASCII letters and the characters in
	// generateSchemeQueryChars.
	Query string

	// EnvVersion is release (default), trial or develop. It only takes
	// effect when the link is opened outside WeChat.
	EnvVersion string
}

// GenerateSchemeParams describes one scheme link.
//
// Without IsExpire the link never expires. With it, exactly one of ExpireTime
// and ExpireInterval is required, chosen by ExpireType.
type GenerateSchemeParams struct {
	// JumpWXA is the target mini program page. Nil opens the home page.
	JumpWXA *JumpWXA

	// IsExpire makes the link expire. False, the default, keeps it
	// permanent.
	IsExpire bool

	// ExpireType selects how expiry is expressed: 0 (the default) reads
	// ExpireTime, 1 reads ExpireInterval. It is only consulted when IsExpire
	// is true.
	ExpireType int

	// ExpireTime is the Unix timestamp the link dies at, required when
	// IsExpire is true and ExpireType is 0. WeChat limits it to between one
	// minute and 30 days from now, and answers 85401 outside that range.
	ExpireTime int64

	// ExpireInterval is the number of days the link lives for, required when
	// IsExpire is true and ExpireType is 1. WeChat caps it at 30 days.
	ExpireInterval int
}

// CheckGenerateSchemeParams validates params before anything is sent to WeChat.
//
// The expire fields are checked only when IsExpire asks for them, because WeChat
// ignores them otherwise. ExpireTime is only required to be positive here: the
// "between one minute and 30 days from now" window depends on the clock, so it
// is left to WeChat, which answers 85401 when it is violated.
func CheckGenerateSchemeParams(params GenerateSchemeParams) error {
	if err := checkJumpWXA(params.JumpWXA); err != nil {
		return err
	}

	return checkExpire(params.IsExpire, params.ExpireType, params.ExpireTime, params.ExpireInterval)
}

// checkJumpWXA validates the jump_wxa object shared by the scheme endpoints.
//
// The expire fields are checked by each endpoint, because only generateScheme
// has them.
func checkJumpWXA(jump *JumpWXA) error {
	if jump == nil {
		return nil
	}

	if err := checkPathQuery(jump.Path, jump.Query); err != nil {
		return err
	}

	return checkEnvVersion(jump.EnvVersion)
}

// checkPathQuery validates a mini program page path and its query, the rules
// shared by every endpoint that opens a page.
//
// A path may not carry a query of its own: business parameters belong in query,
// which is limited to generateSchemeQueryMaxLen characters drawn from digits,
// ASCII letters and generateSchemeQueryChars.
func checkPathQuery(path, query string) error {
	if strings.Contains(path, "?") {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: path must not carry a query, pass it through query",
		)
	}

	if len([]rune(query)) > generateSchemeQueryMaxLen {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf("miniprogram: query must not exceed %d characters", generateSchemeQueryMaxLen),
		)
	}

	if !isValidGenerateSchemeQuery(query) {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: query contains unsupported characters",
		)
	}

	return nil
}

// checkEnvVersion validates the env_version of a generated link.
func checkEnvVersion(version string) error {
	switch version {
	case "", "release", "trial", "develop":
		return nil
	}

	return cerrors.New(
		cerrors.CodeInvalidArgument,
		"miniprogram: env_version must be release, trial or develop",
	)
}

// checkExpire validates the is_expire/expire_* combination shared by the
// generating endpoints.
//
// The expire fields only matter when isExpire asks for them. ExpireTime must be
// positive; the "between one minute and 30 days from now" window depends on the
// clock, so it is left to WeChat, which answers 85401 when it is violated.
func checkExpire(isExpire bool, expireType int, expireTime int64, expireInterval int) error {
	if !isExpire {
		return nil
	}

	switch expireType {
	case generateSchemeExpireTypeTime:
		if expireTime <= 0 {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: expire_time is required when is_expire is true and expire_type is 0",
			)
		}
	case generateSchemeExpireTypeInterval:
		if expireInterval <= 0 || expireInterval > generateSchemeExpireMaxDays {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				fmt.Sprintf(
					"miniprogram: expire_interval must be between 1 and %d days",
					generateSchemeExpireMaxDays,
				),
			)
		}
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: expire_type must be 0 or 1",
		)
	}

	return nil
}

// isValidGenerateSchemeQuery reports whether query is made of digits, ASCII
// letters and the characters generateSchemeQueryChars allows.
func isValidGenerateSchemeQuery(query string) bool {
	for _, r := range query {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if strings.ContainsRune(generateSchemeQueryChars, r) {
			continue
		}
		return false
	}
	return true
}

// body renders params as the JSON body the endpoint expects.
func (p GenerateSchemeParams) body() *generateSchemeBody {
	var jump *jumpWXABody
	if p.JumpWXA != nil {
		jump = &jumpWXABody{
			Path:       p.JumpWXA.Path,
			Query:      p.JumpWXA.Query,
			EnvVersion: p.JumpWXA.EnvVersion,
		}
	}

	return &generateSchemeBody{
		JumpWXA:        jump,
		IsExpire:       p.IsExpire,
		ExpireType:     p.ExpireType,
		ExpireTime:     p.ExpireTime,
		ExpireInterval: p.ExpireInterval,
	}
}

// generateSchemeBody is the JSON body of /wxa/generatescheme.
type generateSchemeBody struct {
	JumpWXA        *jumpWXABody `json:"jump_wxa,omitempty"`
	IsExpire       bool         `json:"is_expire,omitempty"`
	ExpireType     int          `json:"expire_type,omitempty"`
	ExpireTime     int64        `json:"expire_time,omitempty"`
	ExpireInterval int          `json:"expire_interval,omitempty"`
}

// jumpWXABody is the jump_wxa object of the generating endpoints.
type jumpWXABody struct {
	Path       string `json:"path,omitempty"`
	Query      string `json:"query,omitempty"`
	EnvVersion string `json:"env_version,omitempty"`
}

// GenerateSchemeResponse is the reply of /wxa/generatescheme.
type GenerateSchemeResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OpenLink string `json:"openlink"`
}

// OK reports whether the reply carries a scheme link.
func (r *GenerateSchemeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.OpenLink != ""
}

// GenerateSchemeErrMsg returns the description of a generateScheme error.
func GenerateSchemeErrMsg(code int) string {
	if msg, ok := GenerateSchemeErrCodeMap[code]; ok {
		return msg
	}
	return GenerateSchemeDefaultErrMsg
}

// GenerateSchemeError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GenerateSchemeErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GenerateSchemeError(r *GenerateSchemeResponse) error {
	return errors.New(GenerateSchemeErrMsg(r.ErrCode))
}

// GenerateScheme returns an encrypted weixin:// scheme for params.JumpWXA.
//
// This is the way to open a mini program from a SMS, an email or an external
// web page: the page builds the link once on the server and lets the user open
// it later. QueryScheme can turn a generated link back into its page, and
// GenerateNfcScheme does the same job for NFC tags.
//
// The reply carries the link in OpenLink, not in a "scheme" field as
// QueryScheme does.
//
// Only published mini programs can be targeted, and only non-personal
// mainland ones may generate links at all: WeChat answers 85079 and 40002
// otherwise. A daily cap of 500,000 generated links applies, shared with URL
// Link, and weixin:// may trigger a system confirmation dialog before the mini
// program opens.
func (a *Application) GenerateScheme(
	ctx context.Context,
	params GenerateSchemeParams,
) (*GenerateSchemeResponse, error) {
	if err := CheckGenerateSchemeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, generateSchemePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GenerateSchemeResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
