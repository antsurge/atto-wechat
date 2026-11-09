package miniProgram

// KFWorkDefaultErrMsg is returned for an errcode the 微信客服 binding endpoints do
// not document.
const KFWorkDefaultErrMsg = "系统错误，请稍后再试"

// KFWorkErrCodeMap holds the errcode descriptions WeChat documents for the
// /customservice/work endpoints, keyed by the errcode as it appears in the reply.
//
// The family answers in the 1013xxxx range, which is not the 4xxxx/6xxxx range of
// the rest of this package, so it gets its own table.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var KFWorkErrCodeMap = map[int]string{
	10130002: "参数错误",
	10130030: "小程序账号无权限，须为非个人主体的小程序",
	10130600: "corpid 不合法",
	10130601: "corpid 不合法",
	10130602: "corpid 与小程序实体名称不一致",
	10130603: "已经绑定微信客服，请解绑后重试",
}

// KFWorkErrMsg returns the description of a 微信客服 binding error.
func KFWorkErrMsg(code int) string {
	if msg, ok := KFWorkErrCodeMap[code]; ok {
		return msg
	}

	return KFWorkDefaultErrMsg
}

// kfWorkCorpIDBody is the JSON body of the bind and unbind endpoints.
type kfWorkCorpIDBody struct {
	CorpID string `json:"corpid"`
}
