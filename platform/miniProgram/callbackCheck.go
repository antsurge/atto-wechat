package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// callbackCheckPath diagnoses the callback URL WeChat pushes to.
const callbackCheckPath = "/cgi-bin/callback/check"

// 检测动作。
const (
	// CallbackCheckActionDNS resolves the domain name only.
	CallbackCheckActionDNS = "dns"
	// CallbackCheckActionPing pings the resolved addresses only.
	CallbackCheckActionPing = "ping"
	// CallbackCheckActionAll does both.
	CallbackCheckActionAll = "all"
)

// 检测运营商。
const (
	// CallbackCheckOperatorDefault lets WeChat pick the operator.
	CallbackCheckOperatorDefault = "DEFAULT"
	// CallbackCheckOperatorChinaNet checks from 电信.
	CallbackCheckOperatorChinaNet = "CHINANET"
	// CallbackCheckOperatorUnicom checks from 联通.
	CallbackCheckOperatorUnicom = "UNICOM"
	// CallbackCheckOperatorCAP checks from 腾讯.
	CallbackCheckOperatorCAP = "CAP"
)

// CallbackCheckActionMap holds the 检测动作 descriptions.
var CallbackCheckActionMap = map[string]string{
	CallbackCheckActionDNS:  "域名解析",
	CallbackCheckActionPing: "ping 检测",
	CallbackCheckActionAll:  "域名解析和 ping 检测",
}

// CallbackCheckOperatorMap holds the 检测运营商 descriptions.
var CallbackCheckOperatorMap = map[string]string{
	CallbackCheckOperatorDefault:  "自动选择",
	CallbackCheckOperatorChinaNet: "电信",
	CallbackCheckOperatorUnicom:   "联通",
	CallbackCheckOperatorCAP:      "腾讯",
}

// CallbackCheckActionMsg returns the description of a 检测动作 value, or "" when
// it is not one WeChat documents.
func CallbackCheckActionMsg(action string) string {
	return CallbackCheckActionMap[action]
}

// CallbackCheckOperatorMsg returns the description of a 检测运营商 value, or ""
// when it is not one WeChat documents.
func CallbackCheckOperatorMsg(operator string) string {
	return CallbackCheckOperatorMap[operator]
}

// CallbackCheckParams selects what to check and from where.
type CallbackCheckParams struct {
	// Action is one of the CallbackCheckAction* constants.
	Action string
	// CheckOperator is one of the CallbackCheckOperator* constants.
	CheckOperator string
}

// CheckCallbackCheckParams validates params before anything is sent to WeChat.
//
// The two values are enumerations rather than free text, so an unknown one is
// caught here instead of coming back as 40202 or 40203.
func CheckCallbackCheckParams(params CallbackCheckParams) error {
	if _, ok := CallbackCheckActionMap[params.Action]; !ok {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: CallbackCheck action must be dns, ping or all",
		)
	}

	if _, ok := CallbackCheckOperatorMap[params.CheckOperator]; !ok {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: CallbackCheck check_operator must be CHINANET, UNICOM, CAP or DEFAULT",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p CallbackCheckParams) body() *callbackCheckBody {
	return &callbackCheckBody{Action: p.Action, CheckOperator: p.CheckOperator}
}

// callbackCheckBody is the JSON body of /cgi-bin/callback/check.
type callbackCheckBody struct {
	Action        string `json:"action"`
	CheckOperator string `json:"check_operator"`
}

// CallbackCheckDNS is one DNS resolution result.
type CallbackCheckDNS struct {
	// IP is the resolved address.
	IP string `json:"ip"`
	// RealOperator is the operator the address belongs to, as WeChat sees it.
	RealOperator string `json:"real_operator"`
}

// CallbackCheckPing is one ping result.
type CallbackCheckPing struct {
	// IP is the address that was pinged.
	IP string `json:"ip"`
	// FromOperator is where the ping came from, controlled by CheckOperator.
	FromOperator string `json:"from_operator"`
	// PackageLoss is "0%" or "100%": only one packet is sent, so there is no
	// value in between.
	PackageLoss string `json:"package_loss"`
}

// Lost reports whether the ping was lost.
//
// The field is a percentage string rather than a number, so it is compared with
// the two values WeChat documents instead of being parsed.
func (p CallbackCheckPing) Lost() bool {
	return p.PackageLoss == "100%"
}

// CallbackCheckResponse is the reply of /cgi-bin/callback/check.
type CallbackCheckResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// DNS is the resolution result, empty when Action was ping.
	DNS []CallbackCheckDNS `json:"dns"`
	// Ping is the ping result, empty when Action was dns.
	Ping []CallbackCheckPing `json:"ping"`
}

// OK reports whether the check returned something to look at.
func (r *CallbackCheckResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && (len(r.DNS) > 0 || len(r.Ping) > 0)
}

// CallbackCheckErrMsg returns the description of a callbackCheck error.
func CallbackCheckErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// CallbackCheckError turns the errcode from the reply into an error.
func CallbackCheckError(r *CallbackCheckResponse) error {
	return errors.New(CallbackCheckErrMsg(r.ErrCode))
}

// CallbackCheck diagnoses why WeChat cannot reach the callback URL.
//
// It resolves the configured URL and pings every address from the chosen
// operator, which separates a DNS problem (no address, or one pointing somewhere
// unexpected) from a network one (all packets lost).
func (a *Application) CallbackCheck(
	ctx context.Context,
	params CallbackCheckParams,
) (*CallbackCheckResponse, error) {
	if err := CheckCallbackCheckParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, callbackCheckPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result CallbackCheckResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
