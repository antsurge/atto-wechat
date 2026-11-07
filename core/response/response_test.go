package response

import "testing"

func TestWechatResponse(t *testing.T) {
	tests := []struct {
		name string
		resp *WechatResponse
		ok   bool
	}{
		{"success", &WechatResponse{ErrCode: 0, ErrMsg: "ok"}, true},
		{"empty", &WechatResponse{}, true},
		{"error", &WechatResponse{ErrCode: 40013, ErrMsg: "invalid appid"}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resp.OK() != tt.ok {
				t.Errorf("OK = %v, want %v", tt.resp.OK(), tt.ok)
			}
			if tt.resp.IsError() == tt.ok {
				t.Errorf("IsError = %v, want %v", tt.resp.IsError(), !tt.ok)
			}
		})
	}
}

func TestPaymentResponse(t *testing.T) {
	tests := []struct {
		name string
		resp *PaymentResponse
		ok   bool
		err  bool
	}{
		{
			name: "success",
			resp: &PaymentResponse{ReturnCode: PaymentSuccess, ResultCode: PaymentSuccess},
			ok:   true,
			err:  false,
		},
		{
			name: "communication failure",
			resp: &PaymentResponse{ReturnCode: PaymentFail, ReturnMsg: "sign error"},
			err:  true,
		},
		{
			name: "business failure",
			resp: &PaymentResponse{
				ReturnCode: PaymentSuccess,
				ResultCode: PaymentFail,
				ErrCode:    "ORDERPAID",
				ErrMsg:     "order paid",
			},
			err: true,
		},
		{
			name: "not a payment reply",
			resp: &PaymentResponse{},
			err:  false,
		},
		{
			name: "nil",
			resp: nil,
			err:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resp.OK() != tt.ok {
				t.Errorf("OK = %v, want %v", tt.resp.OK(), tt.ok)
			}
			if tt.resp.IsError() != tt.err {
				t.Errorf("IsError = %v, want %v", tt.resp.IsError(), tt.err)
			}
		})
	}
}

func TestMiniProgramResponse(t *testing.T) {
	tests := []struct {
		name string
		resp *MiniProgramResponse
		ok   bool
	}{
		{"success", &MiniProgramResponse{}, true},
		{"risky content", &MiniProgramResponse{ResultCode: "1", ResultMsg: "risky"}, false},
		{"api error", &MiniProgramResponse{WechatResponse: WechatResponse{ErrCode: 40001}}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resp.OK() != tt.ok {
				t.Errorf("OK = %v, want %v", tt.resp.OK(), tt.ok)
			}
		})
	}
}

func TestGenericResponse(t *testing.T) {
	if (&Response{}).OK() != true {
		t.Error("empty Response not OK")
	}
	if (&Response{Code: "E1"}).IsError() != true {
		t.Error("coded Response not an error")
	}
	if (*Response)(nil).OK() {
		t.Error("nil Response is OK")
	}
}
