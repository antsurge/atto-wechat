package response

// WechatResponse is the common header shared by WeChat API replies:
// {"errcode": 0, "errmsg": "ok"}.
//
// errcode 0 (or an absent errcode) means the call succeeded.
type WechatResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

// OK reports whether WeChat accepted the call.
func (r *WechatResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// IsError reports whether WeChat rejected the call.
func (r *WechatResponse) IsError() bool {
	return !r.OK()
}
