package miniProgram

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// validPreviewCustom returns an order that passes CheckAddOrderParams.
func validPreviewCustom() AddOrderParams {
	return validAddOrderParams()
}

func TestProviderUpdateBusiness(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerUpdateBusinessPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderUpdateBusiness(context.Background(), ProviderUpdateBusinessParams{
		ShopAppID:  "wxapp",
		BizID:      "biz-1",
		ResultCode: 1001,
		ResultMsg:  "营业执照不清晰",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}
	if call.method != http.MethodPost || call.path != providerUpdateBusinessPath {
		t.Fatalf("call = %s %s", call.method, call.path)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}

	// The provider endpoints use shop_app_id, not the merchant side's wx_appid.
	if body["shop_app_id"] != "wxapp" || body["result_code"] != float64(1001) {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCheckProviderUpdateBusinessParams(t *testing.T) {
	tests := []struct {
		name   string
		params ProviderUpdateBusinessParams
		want   bool
	}{
		{
			name:   "Approved",
			params: ProviderUpdateBusinessParams{ShopAppID: "wxapp", BizID: "biz"},
			want:   true,
		},
		{
			name: "RejectedWithReason",
			params: ProviderUpdateBusinessParams{
				ShopAppID: "wxapp", BizID: "biz", ResultCode: 1001, ResultMsg: "营业执照不清晰",
			},
			want: true,
		},
		{
			name:   "MissingShopAppID",
			params: ProviderUpdateBusinessParams{BizID: "biz"},
			want:   false,
		},
		{
			name:   "MissingBizID",
			params: ProviderUpdateBusinessParams{ShopAppID: "wxapp"},
			want:   false,
		},
		{
			name:   "RejectedWithoutReason",
			params: ProviderUpdateBusinessParams{ShopAppID: "wxapp", BizID: "biz", ResultCode: 1001},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckProviderUpdateBusinessParams(tt.params)
			if tt.want && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !tt.want && err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestProviderUpdatePath(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerUpdatePathPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderUpdatePath(context.Background(), ProviderUpdatePathParams{
		Token:      "token-1",
		WaybillID:  "SF123",
		ActionTime: 1700000000,
		ActionType: ProviderActionSigned,
		ActionMsg:  "已签收，签收人：本人",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	// token, not order_id, is how the provider side addresses the waybill.
	if body["token"] != "token-1" || body["action_type"] != float64(ProviderActionSigned) {
		t.Fatalf("body = %s", call.body)
	}
	if _, ok := body["order_id"]; ok {
		t.Fatal("order_id was sent to a provider endpoint")
	}
}

func TestCheckProviderUpdatePathParams(t *testing.T) {
	valid := ProviderUpdatePathParams{
		Token:      "token-1",
		WaybillID:  "SF123",
		ActionTime: 1700000000,
		ActionType: ProviderActionPicked,
		ActionMsg:  "已揽件",
	}

	if err := CheckProviderUpdatePathParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	mutations := map[string]func(*ProviderUpdatePathParams){
		"MissingToken":     func(p *ProviderUpdatePathParams) { p.Token = "" },
		"MissingWaybillID": func(p *ProviderUpdatePathParams) { p.WaybillID = "" },
		"ZeroActionTime":   func(p *ProviderUpdatePathParams) { p.ActionTime = 0 },
		"ZeroActionType":   func(p *ProviderUpdatePathParams) { p.ActionType = 0 },
		"MissingActionMsg": func(p *ProviderUpdatePathParams) { p.ActionMsg = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			params := valid
			mutate(&params)

			if err := CheckProviderUpdatePathParams(params); err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestProviderPreviewTemplate(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerPreviewTemplatePath: `{"errcode":0,"waybill_id":"SF123",` +
			`"rendered_waybill_template":"PGh0bWw+"}`,
	})

	app := newApp(t, srv.URL, nil)

	waybillData, err := WaybillDataJSON([]LogisticsWaybillData{{Key: "k", Value: "v"}})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := app.ProviderPreviewTemplate(context.Background(), ProviderPreviewTemplateParams{
		WaybillID:       "SF123",
		WaybillTemplate: "PGh0bWw+",
		WaybillData:     waybillData,
		Custom:          validPreviewCustom(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.RenderedWaybillTemplate != "PGh0bWw+" {
		t.Fatalf("response = %+v", resp)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}

	// waybill_data must be a JSON string, not a nested array.
	if _, ok := body["waybill_data"].(string); !ok {
		t.Fatalf("waybill_data = %#v, want a string", body["waybill_data"])
	}

	// custom is the merchant addOrder body, so it keeps that side's field names.
	custom, ok := body["custom"].(map[string]any)
	if !ok || custom["order_id"] != "order-1" || custom["add_source"] == nil {
		t.Fatalf("custom = %#v", body["custom"])
	}
}

func TestProviderPreviewTemplateInvalidCustom(t *testing.T) {
	app := newApp(t, "https://example.invalid", nil)

	custom := validPreviewCustom()
	custom.OrderID = ""

	_, err := app.ProviderPreviewTemplate(context.Background(), ProviderPreviewTemplateParams{
		WaybillID:       "SF123",
		WaybillTemplate: "PGh0bWw+",
		WaybillData:     "[]",
		Custom:          custom,
	})
	if err == nil {
		t.Fatal("err = nil, want a validation error for an unusable custom order")
	}
}

func TestCheckProviderPreviewTemplateParams(t *testing.T) {
	valid := ProviderPreviewTemplateParams{
		WaybillID:       "SF123",
		WaybillTemplate: "PGh0bWw+",
		WaybillData:     "[]",
		Custom:          validPreviewCustom(),
	}

	if err := CheckProviderPreviewTemplateParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	mutations := map[string]func(*ProviderPreviewTemplateParams){
		"MissingWaybillID":       func(p *ProviderPreviewTemplateParams) { p.WaybillID = "" },
		"MissingWaybillTemplate": func(p *ProviderPreviewTemplateParams) { p.WaybillTemplate = "" },
		"MissingWaybillData":     func(p *ProviderPreviewTemplateParams) { p.WaybillData = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			params := valid
			mutate(&params)

			if err := CheckProviderPreviewTemplateParams(params); err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestWaybillDataJSON(t *testing.T) {
	got, err := WaybillDataJSON([]LogisticsWaybillData{
		{Key: "k1", Value: "v1"},
		{Key: "k2", Value: "v2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got != `[{"key":"k1","value":"v1"},{"key":"k2","value":"v2"}]` {
		t.Fatalf("got %s", got)
	}
}

func TestProviderGetContact(t *testing.T) {
	srv, _ := newLogisticsServer(t, map[string]string{
		providerGetContactPath: `{"errcode":0,"waybill_id":"SF123",` +
			`"sender":{"name":"发货人","mobile":"13800000000","address":"广东省深圳市南山区科技园 1 号"},` +
			`"receiver":{"name":"收货人","tel":"010-12345678","address":"北京市海淀区中关村 1 号"}}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderGetContact(context.Background(), ProviderGetContactParams{
		Token:     "token-1",
		WaybillID: "SF123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.WaybillID != "SF123" {
		t.Fatalf("response = %+v", resp)
	}
	if resp.Sender.Name != "发货人" || resp.Receiver.Tel != "010-12345678" {
		t.Fatalf("sender/receiver = %+v / %+v", resp.Sender, resp.Receiver)
	}
	if resp.Receiver.Address == "" {
		t.Fatal("address was not decoded")
	}
}

func TestProviderGetContactWithoutWaybillIsNotSuccess(t *testing.T) {
	resp := &ProviderGetContactResponse{}
	resp.ErrCode = 0

	if resp.OK() {
		t.Fatal("OK() = true, want false without waybill_id")
	}
}

func TestProviderCancelOrder(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerCancelOrderPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderCancelOrder(context.Background(), ProviderCancelOrderParams{
		Token:  "token-1",
		Reason: "收件地址超出派送范围",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}
	if call.path != providerCancelOrderPath {
		t.Fatalf("path = %s", call.path)
	}
}

func TestProviderUpdateOrderFee(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerUpdateOrderFeePath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderUpdateOrderFee(context.Background(), ProviderUpdateOrderFeeParams{
		Token:        "token-1",
		WaybillID:    "SF123",
		NeedPay:      ProviderNeedPayWechat,
		Fee:          1200,
		OriginalFee:  1500,
		BaseFee:      1200,
		InsuredFee:   200,
		OtherFee:     100,
		Remark:       "包装费",
		PayGoodsName: "运费",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	// Amounts stay in fen; 1500 fen is 15 yuan.
	if body["fee"] != float64(1200) || body["original_fee"] != float64(1500) {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCheckProviderUpdateOrderFeeParams(t *testing.T) {
	valid := ProviderUpdateOrderFeeParams{
		Token:       "token-1",
		WaybillID:   "SF123",
		NeedPay:     ProviderNeedPayWechat,
		Fee:         1500,
		OriginalFee: 1500,
		BaseFee:     1500,
	}

	if err := CheckProviderUpdateOrderFeeParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*ProviderUpdateOrderFeeParams)
	}{
		{name: "MissingToken", mutate: func(p *ProviderUpdateOrderFeeParams) { p.Token = "" }},
		{name: "MissingWaybillID", mutate: func(p *ProviderUpdateOrderFeeParams) { p.WaybillID = "" }},
		{name: "UnknownNeedPay", mutate: func(p *ProviderUpdateOrderFeeParams) { p.NeedPay = 3 }},
		{name: "NegativeFee", mutate: func(p *ProviderUpdateOrderFeeParams) { p.Fee = -1 }},
		{
			name: "TotalDoesNotAddUp",
			mutate: func(p *ProviderUpdateOrderFeeParams) {
				p.InsuredFee = 100
			},
		},
		{
			name: "FeeAboveOriginal",
			mutate: func(p *ProviderUpdateOrderFeeParams) {
				p.Fee = 2000
				p.OriginalFee = 1500
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := valid
			tt.mutate(&params)

			if err := CheckProviderUpdateOrderFeeParams(params); err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestProviderUpdateOrderFeeDiscountIsAllowed(t *testing.T) {
	// A coupon makes fee smaller than original_fee; that must not be rejected.
	params := ProviderUpdateOrderFeeParams{
		Token:       "token-1",
		WaybillID:   "SF123",
		NeedPay:     ProviderNeedPayScore,
		Fee:         1000,
		OriginalFee: 1500,
		BaseFee:     1500,
	}

	if err := CheckProviderUpdateOrderFeeParams(params); err != nil {
		t.Fatalf("discounted fee rejected: %v", err)
	}
}

func TestProviderRefundOrder(t *testing.T) {
	srv, _ := newLogisticsServer(t, map[string]string{
		providerRefundOrderPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderRefundOrder(context.Background(), ProviderRefundOrderParams{
		Token: "token-1",
		Fee:   500,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}

	if err := CheckProviderRefundOrderParams(ProviderRefundOrderParams{Token: "token-1"}); err == nil {
		t.Fatal("zero fee accepted")
	}
}

func TestProviderGetBillFile(t *testing.T) {
	csv := "waybill_id,fee\nSF123,1500\n"
	srv, call := newLogisticsServer(t, map[string]string{
		providerGetBillPath: csv,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderGetBill(context.Background(), ProviderGetBillParams{
		Date: "20250101",
		Type: ProviderBillTypeAll,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}
	if string(resp.Content) != csv {
		t.Fatalf("content = %q", resp.Content)
	}
	if resp.ErrCode != 0 {
		t.Fatalf("errcode = %d", resp.ErrCode)
	}
	if call.method != http.MethodPost {
		t.Fatalf("method = %s", call.method)
	}

	if ProviderGetBillError(resp) != nil {
		t.Fatal("ProviderGetBillError returned an error for a successful download")
	}
}

func TestProviderGetBillJSONError(t *testing.T) {
	srv, _ := newLogisticsServer(t, map[string]string{
		providerGetBillPath: `{"errcode":47001,"errmsg":"data format error"}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderGetBill(context.Background(), ProviderGetBillParams{
		Date: "20250101",
		Type: ProviderBillTypeSuccess,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for a JSON error reply")
	}
	if resp.ErrCode != 47001 {
		t.Fatalf("errcode = %d", resp.ErrCode)
	}
	if len(resp.Content) != 0 {
		t.Fatalf("content = %q, want empty", resp.Content)
	}

	err = ProviderGetBillError(resp)
	if err == nil || err.Error() != ProviderGetBillErrMsg(47001) {
		t.Fatalf("ProviderGetBillError = %v", err)
	}
}

func TestCheckProviderGetBillParams(t *testing.T) {
	valid := ProviderGetBillParams{Date: "20250101", Type: ProviderBillTypeAll}
	if err := CheckProviderGetBillParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	tests := []struct {
		name   string
		params ProviderGetBillParams
	}{
		{name: "ShortDate", params: ProviderGetBillParams{Date: "2025-1-1", Type: ProviderBillTypeAll}},
		{name: "NonDigitDate", params: ProviderGetBillParams{Date: "2025-01-", Type: ProviderBillTypeAll}},
		{name: "UnknownType", params: ProviderGetBillParams{Date: "20250101", Type: "PAID"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckProviderGetBillParams(tt.params); err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestIsJSONBody(t *testing.T) {
	tests := []struct {
		body string
		want bool
	}{
		{body: `{"errcode":0}`, want: true},
		{body: "\n\t {\"errcode\":0}", want: true},
		{body: "waybill_id,fee\n", want: false},
		{body: "", want: false},
	}

	for _, tt := range tests {
		if got := isJSONBody([]byte(tt.body)); got != tt.want {
			t.Fatalf("isJSONBody(%q) = %v, want %v", tt.body, got, tt.want)
		}
	}
}

func TestProviderUpdateComplaintResult(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerUpdateComplaintResultPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderUpdateComplaintResult(
		context.Background(),
		ProviderUpdateComplaintResultParams{
			Token:     "token-1",
			WaybillID: "SF123",
			Result:    "已赔付",
			Desc:      "已按保价金额赔付，3 个工作日内到账",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	// access_token is a query parameter here; the body documents it too but sending
	// it twice adds nothing.
	if _, ok := body["access_token"]; ok {
		t.Fatal("access_token was sent in the body")
	}
	if body["result"] != "已赔付" || body["desc"] == "" {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCheckProviderUpdateComplaintResultParams(t *testing.T) {
	valid := ProviderUpdateComplaintResultParams{
		Token:     "token-1",
		WaybillID: "SF123",
		Result:    "已赔付",
		Desc:      "已按保价金额赔付",
	}

	if err := CheckProviderUpdateComplaintResultParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	mutations := map[string]func(*ProviderUpdateComplaintResultParams){
		"MissingToken":     func(p *ProviderUpdateComplaintResultParams) { p.Token = "" },
		"MissingWaybillID": func(p *ProviderUpdateComplaintResultParams) { p.WaybillID = "" },
		"MissingResult":    func(p *ProviderUpdateComplaintResultParams) { p.Result = "" },
		"MissingDesc":      func(p *ProviderUpdateComplaintResultParams) { p.Desc = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			params := valid
			mutate(&params)

			if err := CheckProviderUpdateComplaintResultParams(params); err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestProviderUpdateOrderStatus(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		providerUpdateOrderStatusPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.ProviderUpdateOrderStatus(
		context.Background(),
		ProviderUpdateOrderStatusParams{
			Token:               "token-1",
			ActionTime:          1700000000,
			ActionType:          ProviderActionCourierAssigned,
			ActionMsg:           "已分配快递员",
			PickupCourierName:   "张三",
			PickupCourierPhone:  "13800000000",
			DeliveryCourierName: "",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("response = %+v", resp)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	// waybill_id is optional at the pre-pickup stages, so it must be omitted when
	// empty rather than sent as "".
	if _, ok := body["waybill_id"]; ok {
		t.Fatal("empty waybill_id was sent")
	}
	if body["pickup_courier_name"] != "张三" {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCheckProviderUpdateOrderStatusParams(t *testing.T) {
	valid := ProviderUpdateOrderStatusParams{
		Token:      "token-1",
		ActionTime: 1700000000,
		ActionType: ProviderActionSiteAccepted,
		ActionMsg:  "网点已接单",
	}

	if err := CheckProviderUpdateOrderStatusParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	t.Run("HalfFilledCourier", func(t *testing.T) {
		params := valid
		params.PickupCourierName = "张三"

		if err := CheckProviderUpdateOrderStatusParams(params); err == nil {
			t.Fatal("a courier name without a phone was accepted")
		}

		params.PickupCourierName = ""
		params.PickupCourierPhone = "13800000000"
		if err := CheckProviderUpdateOrderStatusParams(params); err == nil {
			t.Fatal("a courier phone without a name was accepted")
		}
	})

	t.Run("BothCouriersFilled", func(t *testing.T) {
		params := valid
		params.PickupCourierName = "张三"
		params.PickupCourierPhone = "13800000000"
		params.DeliveryCourierName = "李四"
		params.DeliveryCourierPhone = "13900000000"

		if err := CheckProviderUpdateOrderStatusParams(params); err != nil {
			t.Fatalf("complete couriers rejected: %v", err)
		}
	})

	mutations := map[string]func(*ProviderUpdateOrderStatusParams){
		"MissingToken":     func(p *ProviderUpdateOrderStatusParams) { p.Token = "" },
		"ZeroActionTime":   func(p *ProviderUpdateOrderStatusParams) { p.ActionTime = 0 },
		"ZeroActionType":   func(p *ProviderUpdateOrderStatusParams) { p.ActionType = 0 },
		"MissingActionMsg": func(p *ProviderUpdateOrderStatusParams) { p.ActionMsg = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			params := valid
			mutate(&params)

			if err := CheckProviderUpdateOrderStatusParams(params); err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestProviderValidationSkipsTheRequest(t *testing.T) {
	srv, call := newLogisticsServer(t, nil)
	app := newApp(t, srv.URL, nil)

	_, err := app.ProviderUpdatePath(context.Background(), ProviderUpdatePathParams{WaybillID: "SF123"})
	if err == nil {
		t.Fatal("err = nil, want a validation error")
	}
	if call.calls != 0 {
		t.Fatalf("calls = %d, want 0 for an invalid parameter", call.calls)
	}
}

func TestProviderErrMsg(t *testing.T) {
	if got := ProviderUpdatePathErrMsg(40001); got != "access_token 无效或 AppSecret 错误" {
		t.Fatalf("ProviderUpdatePathErrMsg = %q", got)
	}

	if got := ProviderUpdatePathErrMsg(999999); got != ProviderUpdatePathDefaultErrMsg {
		t.Fatalf("ProviderUpdatePathErrMsg = %q", got)
	}

	// Each endpoint owns its map, so one endpoint cannot mutate another's.
	ProviderUpdatePathErrCodeMap[1] = "changed"
	if _, ok := ProviderUpdateBusinessErrCodeMap[1]; ok {
		t.Fatal("provider errcode maps share state")
	}
}

func TestProviderActionTypeValues(t *testing.T) {
	// The values are the ones the documentation lists; a typo here would send a
	// stage WeChat does not know.
	tests := []struct {
		name string
		got  int
		want int
	}{
		{name: "SiteAccepted", got: ProviderActionSiteAccepted, want: 90001},
		{name: "CourierAssigned", got: ProviderActionCourierAssigned, want: 90002},
		{name: "CourierReassigned", got: ProviderActionCourierReassigned, want: 90003},
		{name: "WaitingPayment", got: ProviderActionWaitingPayment, want: 90010},
		{name: "Paid", got: ProviderActionPaid, want: 90011},
		{name: "Picked", got: ProviderActionPicked, want: 100001},
		{name: "PickFailed", got: ProviderActionPickFailed, want: 100002},
		{name: "InTransit", got: ProviderActionInTransit, want: 200001},
		{name: "Delivering", got: ProviderActionDelivering, want: 300002},
		{name: "Signed", got: ProviderActionSigned, want: 300003},
		{name: "SignFailed", got: ProviderActionSignFailed, want: 300004},
		{name: "ThirdPartyStored", got: ProviderActionThirdPartyStored, want: 300005},
		{name: "ThirdPartyTaken", got: ProviderActionThirdPartyTaken, want: 300006},
		{name: "ProxySigned", got: ProviderActionProxySigned, want: 300007},
		{name: "Cancelled", got: ProviderActionCancelled, want: 400001},
		{name: "Stranded", got: ProviderActionStranded, want: 400002},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("action type = %d, want %d", tt.got, tt.want)
			}
		})
	}
}
