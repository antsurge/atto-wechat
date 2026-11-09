package miniProgram

import (
	"context"
	"errors"
	"unicode/utf8"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// msgSecCheckPath checks a piece of text for illegal content.
const msgSecCheckPath = "/wxa/msg_sec_check"

// MsgSecCheckVersion is the only version WeChat accepts; 2.0 is the sole spelling
// of the v2 interface, not a value to choose from.
const MsgSecCheckVersion = 2

// MsgSecCheckMaxContentLength is the documented limit, counted in characters.
const MsgSecCheckMaxContentLength = 2500

// Values of scene, the context the text comes from.
const (
	// MsgSecCheckSceneProfile is 1: user profile fields.
	MsgSecCheckSceneProfile = 1
	// MsgSecCheckSceneComment is 2: comments.
	MsgSecCheckSceneComment = 2
	// MsgSecCheckSceneForum is 3: forum posts.
	MsgSecCheckSceneForum = 3
	// MsgSecCheckSceneSocialLog is 4: social logs.
	MsgSecCheckSceneSocialLog = 4
)

// MsgSecCheckSceneMap holds the descriptions of the scene values.
var MsgSecCheckSceneMap = map[int]string{
	MsgSecCheckSceneProfile:   "资料",
	MsgSecCheckSceneComment:   "评论",
	MsgSecCheckSceneForum:     "论坛",
	MsgSecCheckSceneSocialLog: "社交日志",
}

// MsgSecCheckSceneMsg returns the description of a scene, or "" when unknown.
func MsgSecCheckSceneMsg(scene int) string {
	return MsgSecCheckSceneMap[scene]
}

// MsgSecCheckErrCodeMap holds the errcode descriptions WeChat documents for this
// endpoint, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var MsgSecCheckErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效或不是最新的",
	40003: "openid 无效",
	40129: "场景值错误（支持 1 资料、2 评论、3 论坛、4 社交日志）",
	43002: "请求方法错误，须使用 POST",
	43104: "appid 与 openid 不匹配",
	44002: "POST 请求体为空",
	44991: "超出接口每分钟调用限制",
	45009: "超出接口每日调用限额",
	47001: "JSON 解析失败，缺少参数",
	61010: "用户访问记录超时（用户需在近两小时访问过小程序）",
}

// MsgSecCheckDefaultErrMsg is returned for an errcode this endpoint does not
// document.
const MsgSecCheckDefaultErrMsg = "系统错误，请稍后再试"

// MsgSecCheckErrMsg returns the description of a msgSecCheck error, or the
// default when the errcode is undocumented.
func MsgSecCheckErrMsg(code int) string {
	if msg, ok := MsgSecCheckErrCodeMap[code]; ok {
		return msg
	}

	return MsgSecCheckDefaultErrMsg
}

// MsgSecCheckParams is the text to check.
type MsgSecCheckParams struct {
	// Content is the text, at most MsgSecCheckMaxContentLength characters of
	// UTF-8.
	Content string

	// Scene is one of the MsgSecCheckScene* values.
	Scene int

	// OpenID is the user who submitted the content.
	//
	// WeChat requires the user to have visited the mini program within the last
	// two hours, and answers 61010 otherwise.
	OpenID string

	// Title is the title of the text, when it has one.
	Title string

	// Nickname is the user's nickname, for profile checks.
	Nickname string

	// Signature is the user's signature. It only takes effect for
	// MsgSecCheckSceneProfile.
	Signature string
}

// CheckMsgSecCheckParams validates params before anything is sent to WeChat.
func CheckMsgSecCheckParams(params MsgSecCheckParams) error {
	if params.Content == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MsgSecCheck content is required",
		)
	}

	// WeChat counts characters, not bytes, so a Chinese text of 2500 characters
	// (7500 bytes) is still within the limit.
	if utf8.RuneCountInString(params.Content) > MsgSecCheckMaxContentLength {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MsgSecCheck content must not exceed 2500 characters",
		)
	}

	if _, ok := MsgSecCheckSceneMap[params.Scene]; !ok {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MsgSecCheck scene must be 1-4",
		)
	}

	if params.OpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: MsgSecCheck openid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p MsgSecCheckParams) body() *msgSecCheckBody {
	return &msgSecCheckBody{
		Content:   p.Content,
		Version:   MsgSecCheckVersion,
		Scene:     p.Scene,
		OpenID:    p.OpenID,
		Title:     p.Title,
		Nickname:  p.Nickname,
		Signature: p.Signature,
	}
}

// msgSecCheckBody is the JSON body of /wxa/msg_sec_check.
type msgSecCheckBody struct {
	Content   string `json:"content"`
	Version   int    `json:"version"`
	Scene     int    `json:"scene"`
	OpenID    string `json:"openid"`
	Title     string `json:"title,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// MsgSecCheckResponse is the reply of /wxa/msg_sec_check.
type MsgSecCheckResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Verdict is the overall result.
	Verdict *SecCheckResult `json:"result"`
	// Detail is each strategy's opinion.
	Detail []SecCheckDetail `json:"detail"`
	// TraceID identifies this request, for support and log correlation.
	TraceID string `json:"trace_id"`
}

// OK reports whether the check ran.
//
// A risky verdict is a successful call: errcode 0 and a verdict are what it means
// to have run, and the verdict itself is what the caller acts on. Use Passed,
// Risky or Review to read it.
func (r *MsgSecCheckResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Verdict != nil
}

// Passed reports whether the content may be shown as it is.
func (r *MsgSecCheckResponse) Passed() bool {
	return r.OK() && r.Verdict.Passed()
}

// Risky reports whether the content must be blocked.
func (r *MsgSecCheckResponse) Risky() bool {
	return r.OK() && r.Verdict.Risky()
}

// Review reports whether the content needs a human decision.
func (r *MsgSecCheckResponse) Review() bool {
	return r.OK() && r.Verdict.Review()
}

// MsgSecCheckError turns the errcode from the reply into an error.
func MsgSecCheckError(r *MsgSecCheckResponse) error {
	return errors.New(MsgSecCheckErrMsg(r.ErrCode))
}

// MsgSecCheck checks whether a piece of text contains illegal content.
//
// The call is synchronous: the verdict comes back in Verdict, and a risky verdict
// is a normal reply rather than an error. Rate limits are generous for a published
// mini program (4000 calls per minute) but a draft one gets 100 calls per day,
// which is worth knowing before wiring it into a hot path.
func (a *Application) MsgSecCheck(
	ctx context.Context,
	params MsgSecCheckParams,
) (*MsgSecCheckResponse, error) {
	if err := CheckMsgSecCheckParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, msgSecCheckPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result MsgSecCheckResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
