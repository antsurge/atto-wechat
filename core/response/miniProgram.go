package response

// MiniProgramResponse 小程序部分接口（如图片/文本内容安全检测）的回包。
//
// 这些接口在 errcode 之外还返回 resultcode：0 表示检测通过。两者都为 0 才
// 表示调用成功。
type MiniProgramResponse struct {
	WechatResponse

	Msg        string `json:"msg,omitempty"`
	ResultCode string `json:"resultcode,omitempty"`
	ResultMsg  string `json:"resultmsg,omitempty"`
}

// OK reports whether both the call and the content check succeeded.
func (r *MiniProgramResponse) OK() bool {
	if r == nil {
		return false
	}

	if !r.WechatResponse.OK() {
		return false
	}

	return r.ResultCode == "" || r.ResultCode == "0"
}
