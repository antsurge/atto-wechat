package response

// Payment status values used by WeChat Pay v2.
const (
	PaymentSuccess = "SUCCESS"
	PaymentFail    = "FAIL"
)

// PaymentResponse 微信支付响应（XML）。
//
// return_code 表示通信是否成功，result_code 表示业务是否成功：两者都为
// SUCCESS 才算成功。err_code/err_code_des 只在业务失败时出现。
type PaymentResponse struct {
	ReturnCode string `xml:"return_code" json:"return_code,omitempty"`
	ReturnMsg  string `xml:"return_msg" json:"return_msg,omitempty"`

	ResultCode string `xml:"result_code" json:"result_code,omitempty"`
	ErrCode    string `xml:"err_code" json:"err_code,omitempty"`
	ErrMsg     string `xml:"err_code_des" json:"err_code_des,omitempty"`
}

// OK 判断支付接口是否成功：通信与业务都成功才算成功。
func (r *PaymentResponse) OK() bool {
	if r == nil {
		return false
	}

	return r.ReturnCode == PaymentSuccess && r.ResultCode == PaymentSuccess
}

// IsError 判断支付接口是否返回错误。
//
// 空结构体（既没有 return_code 也没有 result_code）不算错误：它不是一次支付
// 回包，判成错误只会掩盖真正的问题。
func (r *PaymentResponse) IsError() bool {
	if r == nil {
		return false
	}
	if r.ReturnCode == "" && r.ResultCode == "" && r.ErrCode == "" {
		return false
	}

	return !r.OK()
}
