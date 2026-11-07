package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// generateNFCSchemePath turns a page into a weixin:// scheme that an NFC tag
// can carry.
const (
	generateNFCSchemePath          = "/wxa/generatenfcscheme"
	GenerateNFCSchemeDefaultErrMsg = "系统错误，请稍后再试"
)

// GenerateNFCSchemeErrCodeMap holds the errcode descriptions WeChat documents
// for /wxa/generatenfcscheme, keyed by the errcode as it appears in the reply.
//
// The 9800xxx codes are NFC specific: they report on the model_id and on the
// sn that the device model requires or forbids.
var GenerateNFCSchemeErrCodeMap = map[int]string{
	-1:      "系统繁忙",
	40001:   "access_token 无效",
	40002:   "暂无生成权限：非个人主体小程序无权限，或未申请 NFC 能力的小程序无权限",
	40013:   "生成权限被封禁",
	40165:   "path 填写错误",
	40212:   "query 填写错误，query 格式遵循 URL 标准，即 k1=v1&k2=v2",
	44990:   "调用频率过快，超过 100 次/秒",
	44993:   "单天生成 Scheme + URL Link 数量超过上限 50 万",
	85079:   "小程序没有线上版本，即小程序尚未发布",
	85400:   "长期有效的 Scheme 或 short link 达到生成上限 10 万",
	85402:   "env_version 填写错误",
	9800003: "model_id 检查不通过",
	9800007: "此 model_id 尚未获得该能力，请能力申请通过后再试",
	9800008: "能力类型为一机一码，sn 不能为空",
	9800009: "能力类型为一型一码，sn 需为空",
}

// GenerateNFCSchemeParams describes one NFC scheme link.
//
// JumpWXA is the target page, exactly as in GenerateScheme. ModelID names the
// NFC device model the link is bound to, and SN is only needed when that model
// is provisioned one code per device rather than one code per model.
type GenerateNFCSchemeParams struct {
	// JumpWXA is the target mini program page. Nil opens the home page.
	JumpWXA *JumpWXA

	// ModelID is the model_id of the device the tag is written for. Required.
	ModelID string

	// SN is the serial number of the device, required by models configured as
	// one code per device and forbidden by models configured as one code per
	// model. Whether it is needed is decided by WeChat, which answers
	// 9800008 or 9800009 when it does not match the model.
	SN string
}

// CheckGenerateNFCSchemeParams validates params before anything is sent to
// WeChat.
//
// JumpWXA follows the same rules as GenerateScheme, so the shared check is
// reused. ModelID is the only field WeChat documents as mandatory; the sn
// requirement depends on the model configuration, so it is left to WeChat.
func CheckGenerateNFCSchemeParams(params GenerateNFCSchemeParams) error {
	if err := checkJumpWXA(params.JumpWXA); err != nil {
		return err
	}

	if params.ModelID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: model_id is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GenerateNFCSchemeParams) body() *generateNFCSchemeBody {
	var jump *jumpWXABody
	if p.JumpWXA != nil {
		jump = &jumpWXABody{
			Path:       p.JumpWXA.Path,
			Query:      p.JumpWXA.Query,
			EnvVersion: p.JumpWXA.EnvVersion,
		}
	}

	return &generateNFCSchemeBody{
		JumpWXA: jump,
		ModelID: p.ModelID,
		SN:      p.SN,
	}
}

// generateNFCSchemeBody is the JSON body of /wxa/generatenfcscheme.
//
// Unlike generateSchemeBody it carries no expire fields: an NFC scheme is
// written into a tag and has no lifetime of its own.
type generateNFCSchemeBody struct {
	JumpWXA *jumpWXABody `json:"jump_wxa,omitempty"`
	ModelID string       `json:"model_id"`
	SN      string       `json:"sn,omitempty"`
}

// GenerateNFCSchemeResponse is the reply of /wxa/generatenfcscheme.
type GenerateNFCSchemeResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OpenLink string `json:"openlink"`
}

// OK reports whether the reply carries a scheme link.
func (r *GenerateNFCSchemeResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.OpenLink != ""
}

// GenerateNFCSchemeErrMsg returns the description of a generateNFCScheme error.
func GenerateNFCSchemeErrMsg(code int) string {
	if msg, ok := GenerateNFCSchemeErrCodeMap[code]; ok {
		return msg
	}
	return GenerateNFCSchemeDefaultErrMsg
}

// GenerateNFCSchemeError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GenerateNFCSchemeErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func GenerateNFCSchemeError(r *GenerateNFCSchemeResponse) error {
	return errors.New(GenerateNFCSchemeErrMsg(r.ErrCode))
}

// GenerateNFCScheme returns a weixin:// scheme made for an NFC tag.
//
// The link is written into the tag, and tapping the phone on it then opens the
// mini program page. Only mainland non-personal mini programs that have applied
// for the NFC capability may generate these links, and only published ones:
// WeChat answers 40002 and 85079 otherwise.
//
// model_id must be a model WeChat approved; whether sn has to be supplied is
// decided by the model's provisioning mode (one code per device versus one code
// per model), and WeChat rejects a mismatched combination with 9800008 or
// 9800009. Long-lived schemes share a cap of 100,000 with short links, reported
// as 85400.
func (a *Application) GenerateNFCScheme(
	ctx context.Context,
	params GenerateNFCSchemeParams,
) (*GenerateNFCSchemeResponse, error) {
	if err := CheckGenerateNFCSchemeParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, generateNFCSchemePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GenerateNFCSchemeResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
