package miniProgram

// OpenAPIDefaultErrMsg is returned for an errcode the openApi management
// endpoints do not document.
const OpenAPIDefaultErrMsg = "系统错误，请稍后再试"

// OpenAPIErrCodeMap holds the errcode descriptions WeChat documents for the
// /cgi-bin/openapi, /cgi-bin/clear_quota and /cgi-bin/callback endpoints, keyed
// by the errcode as it appears in the reply.
//
// The 76xxx range is shared by the whole family: quota queries, quota resets and
// rid lookups all answer with the same failures, so the table lives here instead
// of being repeated in six files.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var OpenAPIErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40013: "appid 无效",
	40201: "检测的 url 无效",
	40202: "检测的 action 无效",
	40203: "检测的 check_operator 无效",
	41001: "缺少 access_token",
	41002: "缺少 appid",
	42001: "access_token 已过期",
	44002: "POST 数据为空",
	45009: "接口调用超过天级别上限",
	48006: "重置次数已达上限，禁止重置",
	50002: "用户受限",
	76001: "rid 不存在（rid 仅保留 7 天）",
	76002: "rid 格式错误",
	76003: "该 rid 无权查询，只能查本账号的 rid",
	76004: "rid 时间错误",
	76021: "cgi_path 不存在",
	76022: "该 cgi_path 无权使用，token 与接口所属账号不符",
}

// OpenAPIErrMsg returns the description of an openApi management error.
func OpenAPIErrMsg(code int) string {
	if msg, ok := OpenAPIErrCodeMap[code]; ok {
		return msg
	}

	return OpenAPIDefaultErrMsg
}

// APIQuota is the daily quota of one server API.
type APIQuota struct {
	// DailyLimit is how many times the account may call the API today.
	DailyLimit int `json:"daily_limit"`
	// Used is how many times it has been called today.
	Used int `json:"used"`
	// Remain is what is left for today.
	Remain int `json:"remain"`
}

// APIRateLimit is the call-frequency limit of one server API.
type APIRateLimit struct {
	// CallCount is how many calls are allowed per period.
	CallCount int `json:"call_count"`
	// RefreshSecond is the length of the period, in seconds.
	RefreshSecond int `json:"refresh_second"`
}
