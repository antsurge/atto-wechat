package miniProgram

import (
	"context"
	"errors"
	"net"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getUserRiskRankPath reads a user's risk level.
const getUserRiskRankPath = "/wxa/getuserriskrank"

// Values of scene, the situation the risk is being judged in.
const (
	// UserRiskSceneRegister is 0: registration.
	UserRiskSceneRegister = 0
	// UserRiskSceneMarketing is 1: marketing abuse.
	UserRiskSceneMarketing = 1
	// UserRiskSceneUGC is 2: user generated content.
	UserRiskSceneUGC = 2
)

// UserRiskSceneMap holds the descriptions of the scene values.
var UserRiskSceneMap = map[int]string{
	UserRiskSceneRegister:  "注册",
	UserRiskSceneMarketing: "营销作弊",
	UserRiskSceneUGC:       "UGC",
}

// UserRiskSceneMsg returns the description of a scene, or "" when unknown.
func UserRiskSceneMsg(scene int) string {
	return UserRiskSceneMap[scene]
}

// Risk ranks. The number is the risk: 0 is the most trustworthy, 4 the least.
//
// WeChat documents the range and the ordering but no per-rank meaning, so none is
// invented here: where to draw the line between "allow" and "block" is a product
// decision, not something the SDK can settle.
const (
	// UserRiskRankLowest is 0.
	UserRiskRankLowest = 0
	// UserRiskRankHighest is 4.
	UserRiskRankHighest = 4
)

// GetUserRiskRankErrCodeMap holds the errcode descriptions WeChat documents for
// this endpoint, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GetUserRiskRankErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效或不是最新的",
	40003: "openid 无效",
	40129: "场景值错误（支持 0 注册、1 营销作弊、2 UGC）",
	41001: "缺少 access_token 参数",
	42001: "access_token 已过期",
	43002: "请求方法错误，须使用 POST",
	43104: "appid 与 openid 不匹配",
	44002: "POST 请求体为空",
	47001: "JSON 解析失败，缺少参数",
	48001: "小程序无该 api 权限，可在 mp.weixin.qq.com 开通",
	61007: "小程序尚未将对应的权限集授权给第三方平台",
	61010: "用户访问记录超时（用户需在近两小时访问过小程序）",
	61080: "当前用户安全服务不可用",
	61081: "用户安全接口调用过于频繁",
}

// GetUserRiskRankDefaultErrMsg is returned for an errcode this endpoint does not
// document.
const GetUserRiskRankDefaultErrMsg = "系统错误，请稍后再试"

// GetUserRiskRankErrMsg returns the description of a getUserRiskRank error, or the
// default when the errcode is undocumented.
func GetUserRiskRankErrMsg(code int) string {
	if msg, ok := GetUserRiskRankErrCodeMap[code]; ok {
		return msg
	}

	return GetUserRiskRankDefaultErrMsg
}

// GetUserRiskRankParams describes the user and the situation to judge.
type GetUserRiskRankParams struct {
	// AppID is the mini program. Leaving it empty fills in this application's
	// appid, which is the only value that can succeed: a mismatched appid and
	// openid answer 43104.
	AppID string

	// OpenID is the user being judged.
	OpenID string

	// Scene is one of the UserRiskScene* values.
	Scene int

	// MobileNo is the user's phone number, when known. More signals mean a better
	// judgement.
	MobileNo string

	// ClientIP is the user's source address.
	//
	// A single address, not a forwarded chain: take the first entry of
	// X-Forwarded-For and strip the port before calling.
	ClientIP string

	// EmailAddress is the user's email address, when known.
	EmailAddress string

	// ExtendedInfo is free-form extra information.
	ExtendedInfo string

	// BankCardNo and CertNo appear in WeChat's request example but not in its
	// parameter table. They are passed through when set.
	BankCardNo string
	CertNo     string

	// IsTest marks the call as a test. Test calls return a result but are not
	// counted as production judgements, so keep it false in production.
	IsTest bool
}

// CheckGetUserRiskRankParams validates params before anything is sent to WeChat.
//
// appid may be empty: the application fills it in, since any other value is
// rejected anyway.
func CheckGetUserRiskRankParams(params GetUserRiskRankParams) error {
	if params.OpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetUserRiskRank openid is required",
		)
	}

	if _, ok := UserRiskSceneMap[params.Scene]; !ok {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetUserRiskRank scene must be 0, 1 or 2",
		)
	}

	if params.ClientIP == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetUserRiskRank client_ip is required",
		)
	}

	// WeChat judges by the real address; a hostname or a forwarded chain would
	// either be rejected with a generic 参数错误 or be read as a different address,
	// and a risk verdict about the wrong address is worse than a local error.
	if net.ParseIP(params.ClientIP) == nil {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetUserRiskRank client_ip must be a single IP address",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetUserRiskRankParams) body(appID string) *getUserRiskRankBody {
	if p.AppID != "" {
		appID = p.AppID
	}

	return &getUserRiskRankBody{
		AppID:        appID,
		OpenID:       p.OpenID,
		Scene:        p.Scene,
		MobileNo:     p.MobileNo,
		ClientIP:     p.ClientIP,
		EmailAddress: p.EmailAddress,
		ExtendedInfo: p.ExtendedInfo,
		BankCardNo:   p.BankCardNo,
		CertNo:       p.CertNo,
		IsTest:       p.IsTest,
	}
}

// getUserRiskRankBody is the JSON body of /wxa/getuserriskrank.
type getUserRiskRankBody struct {
	AppID        string `json:"appid"`
	OpenID       string `json:"openid"`
	Scene        int    `json:"scene"`
	MobileNo     string `json:"mobile_no,omitempty"`
	ClientIP     string `json:"client_ip"`
	EmailAddress string `json:"email_address,omitempty"`
	ExtendedInfo string `json:"extended_info,omitempty"`
	BankCardNo   string `json:"bank_card_no,omitempty"`
	CertNo       string `json:"cert_no,omitempty"`
	IsTest       bool   `json:"is_test,omitempty"`
}

// GetUserRiskRankResponse is the reply of /wxa/getuserriskrank.
type GetUserRiskRankResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// RiskRank is the risk level, 0-4, higher meaning riskier.
	//
	// It is a pointer because 0 is a legal value -- the most trustworthy one -- so
	// an absent field must not be readable as "0" and let an unknown user through.
	// Use Rank to read it without the pointer.
	RiskRank *int `json:"risk_rank"`

	// UnionID is WeChat's unique id for this request.
	//
	// It is spelled "unoin_id" in the documented payload, and the tag keeps that
	// spelling: renaming the tag would stop the field from decoding.
	UnionID int64 `json:"unoin_id"`
}

// OK reports whether the reply carries a risk level.
func (r *GetUserRiskRankResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.RiskRank != nil
}

// Rank returns the risk level and whether the reply carried one.
func (r *GetUserRiskRankResponse) Rank() (int, bool) {
	if r == nil || r.RiskRank == nil {
		return 0, false
	}

	return *r.RiskRank, true
}

// GetUserRiskRankError turns the errcode from the reply into an error.
func GetUserRiskRankError(r *GetUserRiskRankResponse) error {
	return errors.New(GetUserRiskRankErrMsg(r.ErrCode))
}

// GetUserRiskRank judges how risky a user is, in a given situation.
//
// 48001 means the capability is not enabled for the mini program, and 61080/61081
// mean the service itself is unavailable or being called too often; none of them
// is a verdict, so a caller that blocks on "no verdict" should decide in advance
// whether it wants to fail open or closed.
func (a *Application) GetUserRiskRank(
	ctx context.Context,
	params GetUserRiskRankParams,
) (*GetUserRiskRankResponse, error) {
	if err := CheckGetUserRiskRankParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getUserRiskRankPath, nil, params.body(a.cfg.AppID))
	if err != nil {
		return nil, err
	}

	var result GetUserRiskRankResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
