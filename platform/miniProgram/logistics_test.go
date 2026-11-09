package miniProgram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// logisticsCall records the last request a logistics endpoint made.
type logisticsCall struct {
	method string
	path   string
	query  url.Values
	body   []byte
	calls  int
}

// newLogisticsServer serves the token endpoint plus one canned reply per path.
//
// It exists instead of extending fakeWechat because logistics asserts on the
// request body, and fakeWechat's default branch answers unknown paths with a
// bare {"errcode":0} without recording anything.
func newLogisticsServer(t *testing.T, replies map[string]string) (*httptest.Server, *logisticsCall) {
	t.Helper()

	call := &logisticsCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == tokenPath {
			_, _ = io.WriteString(w, `{"access_token":"at","expires_in":7200}`)
			return
		}

		call.calls++
		call.method = r.Method
		call.path = r.URL.Path
		call.query = r.URL.Query()
		call.body, _ = io.ReadAll(r.Body)

		reply, ok := replies[r.URL.Path]
		if !ok {
			reply = `{"errcode":0}`
		}
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)

	return srv, call
}

// validAddOrderParams returns a parameter set that passes every check.
func validAddOrderParams() AddOrderParams {
	return AddOrderParams{
		OrderID:    "order-1",
		OpenID:     "o_user",
		DeliveryID: "SF",
		BizID:      "biz-1",
		AddSource:  LogisticsAddSourceMiniProgram,
		Sender: LogisticsPersonInfo{
			Name:     "发货人",
			Mobile:   "13800000000",
			Province: "广东省",
			City:     "深圳市",
			Area:     "南山区",
			Address:  "科技园 1 号",
		},
		Receiver: LogisticsPersonInfo{
			Name:     "收货人",
			Mobile:   "13900000000",
			Province: "北京市",
			City:     "北京市",
			Area:     "海淀区",
			Address:  "中关村 1 号",
		},
		Cargo: LogisticsCargo{
			Count:      1,
			Weight:     0.5,
			SpaceX:     20,
			SpaceY:     15,
			SpaceZ:     10,
			DetailList: []LogisticsCargoDetail{{Name: "图书", Count: 1}},
		},
	}
}

func TestAddOrderSendsDocumentedBody(t *testing.T) {
	reply := `{"errcode":0,"order_id":"order-1","waybill_id":"SF123",` +
		`"waybill_data":[{"key":"k","value":"v"}],"delivery_resultcode":0}`
	srv, call := newLogisticsServer(t, map[string]string{addOrderPath: reply})

	app := newApp(t, srv.URL, nil)

	params := validAddOrderParams()
	params.TagID = 12345
	params.CustomRemark = "易碎物品"
	params.Shop = &LogisticsShop{
		WxaPath:    "pages/order/detail?id=order-1",
		ImgURL:     "https://example.com/goods.png",
		GoodsName:  "图书",
		GoodsCount: 1,
	}
	params.Insured = &LogisticsInsured{UseInsured: 1, InsuredValue: 10000}
	params.Service = &LogisticsService{ServiceType: 1, ServiceName: "标准快递"}

	resp, err := app.AddOrder(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	if call.method != http.MethodPost {
		t.Fatalf("method = %s, want POST", call.method)
	}
	if call.path != addOrderPath {
		t.Fatalf("path = %s, want %s", call.path, addOrderPath)
	}
	if call.query.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", call.query.Get("access_token"))
	}
	if !resp.OK() || resp.WaybillID != "SF123" {
		t.Fatalf("response = %+v", resp)
	}
	if len(resp.WaybillData) != 1 || resp.WaybillData[0].Key != "k" {
		t.Fatalf("WaybillData = %#v", resp.WaybillData)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}

	// The documented names, including the ones that are easy to get wrong:
	// tagid has no underscore, add_source is a number, wx_appid is only for App/H5.
	checks := map[string]any{
		"order_id":      "order-1",
		"openid":        "o_user",
		"delivery_id":   "SF",
		"biz_id":        "biz-1",
		"add_source":    float64(LogisticsAddSourceMiniProgram),
		"tagid":         float64(12345),
		"custom_remark": "易碎物品",
	}
	for key, want := range checks {
		if got, ok := body[key]; !ok || got != want {
			t.Fatalf("body[%q] = %v (present %v), want %v", key, got, ok, want)
		}
	}

	if _, ok := body["wx_appid"]; ok {
		t.Fatal("wx_appid was sent for a mini program order")
	}

	shop, ok := body["shop"].(map[string]any)
	if !ok {
		t.Fatalf("shop = %#v", body["shop"])
	}
	if shop["goods_count"] != float64(1) || shop["wxa_path"] == "" {
		t.Fatalf("shop = %#v", shop)
	}

	insured, ok := body["insured"].(map[string]any)
	if !ok || insured["insured_value"] != float64(10000) {
		t.Fatalf("insured = %#v", body["insured"])
	}

	cargo, ok := body["cargo"].(map[string]any)
	if !ok || len(cargo["detail_list"].([]any)) != 1 {
		t.Fatalf("cargo = %#v", body["cargo"])
	}
}

func TestAddOrderOptionalSectionsAreOmitted(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		addOrderPath: `{"errcode":0,"waybill_id":"SF123"}`,
	})

	app := newApp(t, srv.URL, nil)

	if _, err := app.AddOrder(context.Background(), validAddOrderParams()); err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"shop", "insured", "service", "wx_appid", "tagid"} {
		if _, ok := body[key]; ok {
			t.Fatalf("%q was sent although it was not set", key)
		}
	}
}

func TestAddOrderCourierRejectionIsNotSuccess(t *testing.T) {
	// WeChat accepts the call but the courier refuses it: errcode 0, no waybill.
	srv, _ := newLogisticsServer(t, map[string]string{
		addOrderPath: `{"errcode":0,"delivery_resultcode":10001,"delivery_resultmsg":"收件地址不派送"}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.AddOrder(context.Background(), validAddOrderParams())
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true, want false when the courier refused the order")
	}
	if err := AddOrderError(resp); err == nil || err.Error() != "收件地址不派送" {
		t.Fatalf("AddOrderError = %v, want the courier message", err)
	}

	err = resp.CourierError()
	if err == nil || err.Error() != "收件地址不派送" {
		t.Fatalf("CourierError = %v", err)
	}
}

func TestAddOrderCourierRejectionWithoutMessage(t *testing.T) {
	resp := &AddOrderResponse{DeliveryResultCode: 10001}
	if err := resp.CourierError(); err == nil || err.Error() != "运力方拒绝下单" {
		t.Fatalf("CourierError = %v", err)
	}

	if (&AddOrderResponse{WaybillID: "SF123"}).CourierError() != nil {
		t.Fatal("CourierError returned an error for a successful reply")
	}
}

func TestCheckAddOrderParams(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AddOrderParams)
		want   bool
	}{
		{name: "Valid", mutate: func(*AddOrderParams) {}, want: true},
		{
			name:   "MissingOrderID",
			mutate: func(p *AddOrderParams) { p.OrderID = "" },
			want:   false,
		},
		{
			name:   "MissingDeliveryID",
			mutate: func(p *AddOrderParams) { p.DeliveryID = "" },
			want:   false,
		},
		{
			name:   "MissingBizID",
			mutate: func(p *AddOrderParams) { p.BizID = "" },
			want:   false,
		},
		{
			name:   "MiniProgramOrderWithoutOpenID",
			mutate: func(p *AddOrderParams) { p.OpenID = "" },
			want:   false,
		},
		{
			name: "AppOrderWithoutAppID",
			mutate: func(p *AddOrderParams) {
				p.AddSource = LogisticsAddSourceAppOrH5
				p.OpenID = ""
			},
			want: false,
		},
		{
			name: "AppOrderWithAppID",
			mutate: func(p *AddOrderParams) {
				p.AddSource = LogisticsAddSourceAppOrH5
				p.OpenID = ""
				p.WxAppID = "wxapp"
			},
			want: true,
		},
		{
			name:   "UnknownAddSource",
			mutate: func(p *AddOrderParams) { p.AddSource = 1 },
			want:   false,
		},
		{
			name:   "SenderWithoutPhone",
			mutate: func(p *AddOrderParams) { p.Sender.Mobile = "" },
			want:   false,
		},
		{
			name: "SenderWithLandline",
			mutate: func(p *AddOrderParams) {
				p.Sender.Mobile = ""
				p.Sender.Tel = "010-12345678"
			},
			want: true,
		},
		{
			name:   "ReceiverWithoutName",
			mutate: func(p *AddOrderParams) { p.Receiver.Name = "" },
			want:   false,
		},
		{
			name:   "CargoWithoutCount",
			mutate: func(p *AddOrderParams) { p.Cargo.Count = 0 },
			want:   false,
		},
		{
			name:   "CargoWithoutItems",
			mutate: func(p *AddOrderParams) { p.Cargo.DetailList = nil },
			want:   false,
		},
		{
			name: "CargoItemWithoutCount",
			mutate: func(p *AddOrderParams) {
				p.Cargo.DetailList = []LogisticsCargoDetail{{Name: "图书"}}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validAddOrderParams()
			tt.mutate(&params)

			err := CheckAddOrderParams(params)
			if tt.want && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !tt.want && err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestLogisticsValidationSkipsTheRequest(t *testing.T) {
	srv, call := newLogisticsServer(t, nil)
	app := newApp(t, srv.URL, nil)

	params := validAddOrderParams()
	params.OrderID = ""

	if _, err := app.AddOrder(context.Background(), params); err == nil {
		t.Fatal("err = nil, want a validation error")
	}
	if call.calls != 0 {
		t.Fatalf("calls = %d, want 0 for an invalid parameter", call.calls)
	}
}

func TestGetAllAccount(t *testing.T) {
	reply := `{"errcode":0,"count":1,"list":[{"biz_id":"biz-1","delivery_id":"SF",` +
		`"create_time":1700000000,"update_time":1700000001,"status_code":0,` +
		`"alias":"顺丰月结","quota_num":100,"quota_update_time":1700000002,` +
		`"service_type":[{"service_type":1,"service_name":"标准快递"}]}]}`
	srv, call := newLogisticsServer(t, map[string]string{getAllAccountPath: reply})

	app := newApp(t, srv.URL, nil)

	resp, err := app.GetAllAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if call.method != http.MethodGet {
		t.Fatalf("method = %s, want GET", call.method)
	}
	if !resp.OK() || resp.Count != 1 || len(resp.List) != 1 {
		t.Fatalf("response = %+v", resp)
	}

	account := resp.List[0]
	if !account.OK() || account.BizID != "biz-1" || account.QuotaNum != 100 {
		t.Fatalf("account = %+v", account)
	}
	if len(account.ServiceType) != 1 || account.ServiceType[0].ServiceName != "标准快递" {
		t.Fatalf("ServiceType = %#v", account.ServiceType)
	}
}

func TestGetAllDelivery(t *testing.T) {
	reply := `{"errcode":0,"count":1,"data":[{"delivery_id":"SF","delivery_name":"顺丰速运",` +
		`"can_use_cash":1,"can_get_quota":1,"cash_biz_id":"biz-cash",` +
		`"service_type":[{"service_type":1,"service_name":"标准快递"}]}]}`
	srv, _ := newLogisticsServer(t, map[string]string{getAllDeliveryPath: reply})

	app := newApp(t, srv.URL, nil)

	resp, err := app.GetAllDelivery(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.Count != 1 {
		t.Fatalf("response = %+v", resp)
	}

	company := resp.Find("SF")
	if company == nil || company.DeliveryName != "顺丰速运" || company.CanGetQuota != 1 {
		t.Fatalf("company = %+v", company)
	}
	if resp.Find("YTO") != nil {
		t.Fatal("Find returned a courier that is not in the list")
	}
}

func TestCancelOrder(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		cancelOrderPath: `{"errcode":0,"delivery_resultcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.CancelOrder(context.Background(), CancelOrderParams{
		OrderID:    "order-1",
		OpenID:     "o_user",
		DeliveryID: "SF",
		WaybillID:  "SF123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if call.method != http.MethodPost || !resp.OK() {
		t.Fatalf("method = %s, response = %+v", call.method, resp)
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["waybill_id"] != "SF123" || body["order_id"] != "order-1" {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCancelOrderCourierRejection(t *testing.T) {
	resp := &CancelOrderResponse{DeliveryResultCode: 10002, DeliveryResultMsg: "已揽收不可取消"}
	if resp.OK() {
		t.Fatal("OK() = true, want false")
	}
	if err := resp.CourierError(); err == nil || err.Error() != "已揽收不可取消" {
		t.Fatalf("CourierError = %v", err)
	}

	empty := &CancelOrderResponse{DeliveryResultCode: 10002}
	if err := empty.CourierError(); err == nil || err.Error() != "运力方拒绝取消运单" {
		t.Fatalf("CourierError = %v", err)
	}
}

func TestGetOrder(t *testing.T) {
	reply := `{"errcode":0,"print_html":"<html></html>",` +
		`"waybill_data":[{"key":"k","value":"v"}],"order_id":"order-1",` +
		`"delivery_id":"SF","waybill_id":"SF123","order_status":0}`
	srv, call := newLogisticsServer(t, map[string]string{getOrderPath: reply})

	app := newApp(t, srv.URL, nil)

	resp, err := app.GetOrder(context.Background(), GetOrderParams{
		OrderID:    "order-1",
		DeliveryID: "SF",
		WaybillID:  "SF123",
		PrintType:  LogisticsPrintTypeOnePart,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.PrintHTML != "<html></html>" || resp.OrderStatus != LogisticsOrderStatusNormal {
		t.Fatalf("response = %+v", resp)
	}
	if !resp.LogisticsOrderData.OK() {
		t.Fatal("LogisticsOrderData.OK() = false")
	}

	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["print_type"] != float64(LogisticsPrintTypeOnePart) {
		t.Fatalf("print_type = %v", body["print_type"])
	}
}

func TestGetOrderWithoutWaybillIsNotSuccess(t *testing.T) {
	resp := &GetOrderResponse{}
	resp.ErrCode = 0

	if resp.OK() {
		t.Fatal("OK() = true, want false without waybill_id")
	}
}

func TestBatchGetOrder(t *testing.T) {
	reply := `{"errcode":0,"order_list":[{"order_id":"o1","delivery_id":"SF","waybill_id":"SF1"},` +
		`{"order_id":"o2","delivery_id":"SF","waybill_id":"SF2"}]}`
	srv, call := newLogisticsServer(t, map[string]string{batchGetOrderPath: reply})

	app := newApp(t, srv.URL, nil)

	resp, err := app.BatchGetOrder(context.Background(), BatchGetOrderParams{
		OrderList: []LogisticsOrderQuery{
			{OrderID: "o1", DeliveryID: "SF"},
			{OrderID: "o2", DeliveryID: "SF"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || len(resp.OrderList) != 2 || resp.OrderList[1].WaybillID != "SF2" {
		t.Fatalf("response = %+v", resp)
	}

	var body struct {
		OrderList []map[string]any `json:"order_list"`
	}
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.OrderList) != 2 || body.OrderList[0]["order_id"] != "o1" {
		t.Fatalf("body = %s", call.body)
	}
	// print_type is not part of a batch item; sending it would be rejected.
	if _, ok := body.OrderList[0]["print_type"]; ok {
		t.Fatal("print_type was sent inside order_list")
	}
}

func TestCheckBatchGetOrderParams(t *testing.T) {
	if err := CheckBatchGetOrderParams(BatchGetOrderParams{}); err == nil {
		t.Fatal("empty order_list accepted")
	}

	tooMany := make([]LogisticsOrderQuery, BatchGetOrderMaxSize+1)
	for i := range tooMany {
		tooMany[i] = LogisticsOrderQuery{OrderID: "o", DeliveryID: "SF"}
	}
	if err := CheckBatchGetOrderParams(BatchGetOrderParams{OrderList: tooMany}); err == nil {
		t.Fatal("101 orders accepted")
	}

	ok := make([]LogisticsOrderQuery, BatchGetOrderMaxSize)
	for i := range ok {
		ok[i] = LogisticsOrderQuery{OrderID: "o", DeliveryID: "SF"}
	}
	if err := CheckBatchGetOrderParams(BatchGetOrderParams{OrderList: ok}); err != nil {
		t.Fatalf("100 orders rejected: %v", err)
	}

	if err := CheckBatchGetOrderParams(BatchGetOrderParams{
		OrderList: []LogisticsOrderQuery{{OrderID: "o"}},
	}); err == nil {
		t.Fatal("item without delivery_id accepted")
	}
}

func TestGetPath(t *testing.T) {
	reply := `{"errcode":0,"openid":"o_user","delivery_id":"SF","waybill_id":"SF123",` +
		`"path_item_num":2,"path_item_list":[` +
		`{"action_time":1700000000,"action_type":100001,"action_msg":"已揽件"},` +
		`{"action_time":1700000100,"action_type":100002,"action_msg":"运输中"}]}`
	srv, call := newLogisticsServer(t, map[string]string{getPathPath: reply})

	app := newApp(t, srv.URL, nil)

	resp, err := app.GetPath(context.Background(), GetPathParams{
		DeliveryID: "SF",
		WaybillID:  "SF123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.PathItemNum != 2 {
		t.Fatalf("response = %+v", resp)
	}

	latest := resp.Latest()
	if latest == nil || latest.ActionMsg != "运输中" {
		t.Fatalf("Latest = %+v", latest)
	}

	if call.query.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", call.query.Get("access_token"))
	}
}

func TestGetPathEmptyIsSuccess(t *testing.T) {
	// A freshly created waybill has no tracking events yet; that is not an error.
	resp := &GetPathResponse{}
	resp.ErrCode = 0

	if !resp.OK() {
		t.Fatal("OK() = false for an empty tracking list")
	}
	if resp.Latest() != nil {
		t.Fatal("Latest returned an item for an empty list")
	}
}

func TestGetQuota(t *testing.T) {
	srv, _ := newLogisticsServer(t, map[string]string{
		getQuotaPath: `{"errcode":0,"quota_num":88,"quota_update_time":1700000000}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.GetQuota(context.Background(), GetQuotaParams{DeliveryID: "SF", BizID: "biz-1"})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.QuotaNum != 88 || resp.QuotaUpdateTime != 1700000000 {
		t.Fatalf("response = %+v", resp)
	}
}

func TestUpdatePrinter(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		updatePrinterPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.UpdatePrinter(context.Background(), UpdatePrinterParams{
		OpenID:     "o_printer",
		TagIDList:  "123,456",
		UpdateType: LogisticsPrinterUpdateBind,
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
	if body["update_type"] != LogisticsPrinterUpdateBind || body["tagid_list"] != "123,456" {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCheckUpdatePrinterParams(t *testing.T) {
	tests := []struct {
		name   string
		params UpdatePrinterParams
		want   bool
	}{
		{name: "Valid", params: UpdatePrinterParams{OpenID: "o", UpdateType: LogisticsPrinterUpdateBind}, want: true},
		{name: "MissingOpenID", params: UpdatePrinterParams{UpdateType: LogisticsPrinterUpdateBind}, want: false},
		{name: "UnknownType", params: UpdatePrinterParams{OpenID: "o", UpdateType: "set"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckUpdatePrinterParams(tt.params)
			if tt.want && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !tt.want && err == nil {
				t.Fatal("err = nil, want a validation error")
			}
		})
	}
}

func TestGetAllPrinter(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		getAllPrinterPath: `{"errcode":0,"count":2,"openid":["o1","o2"]}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.GetAllPrinter(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if call.method != http.MethodGet {
		t.Fatalf("method = %s, want GET", call.method)
	}
	if !resp.OK() || resp.Count != 2 || len(resp.OpenID) != 2 {
		t.Fatalf("response = %+v", resp)
	}
}

func TestTestUpdateOrder(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		testUpdateOrderPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.TestUpdateOrder(context.Background(), TestUpdateOrderParams{
		BizID:      "test_biz_id",
		OrderID:    "order-1",
		DeliveryID: "TEST",
		WaybillID:  "TEST123",
		ActionTime: 1700000000,
		ActionType: 100001,
		ActionMsg:  "已揽件",
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
	// biz_id is required by this endpoint even though AddOrder needs it too; it is
	// the value that makes the endpoint pick the test account.
	if body["biz_id"] != "test_biz_id" || body["action_msg"] != "已揽件" {
		t.Fatalf("body = %s", call.body)
	}
}

func TestBindAccount(t *testing.T) {
	srv, call := newLogisticsServer(t, map[string]string{
		bindAccountPath: `{"errcode":0}`,
	})

	app := newApp(t, srv.URL, nil)

	resp, err := app.BindAccount(context.Background(), BindAccountParams{
		Type:          LogisticsBindTypeBind,
		BizID:         "biz-1",
		DeliveryID:    "SF",
		Password:      "pwd",
		RemarkContent: "电话/联系人/服务类型/发货地址",
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
	if body["type"] != LogisticsBindTypeBind || body["biz_id"] != "biz-1" {
		t.Fatalf("body = %s", call.body)
	}
}

func TestCheckBindAccountParams(t *testing.T) {
	valid := BindAccountParams{Type: LogisticsBindTypeBind, BizID: "biz", DeliveryID: "SF"}
	if err := CheckBindAccountParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	if err := CheckBindAccountParams(BindAccountParams{Type: "add", BizID: "biz", DeliveryID: "SF"}); err == nil {
		t.Fatal("unknown type accepted")
	}
	if err := CheckBindAccountParams(BindAccountParams{Type: LogisticsBindTypeBind, DeliveryID: "SF"}); err == nil {
		t.Fatal("missing biz_id accepted")
	}
	if err := CheckBindAccountParams(BindAccountParams{Type: LogisticsBindTypeBind, BizID: "biz"}); err == nil {
		t.Fatal("missing delivery_id accepted")
	}
}

func TestCheckCancelOrderParams(t *testing.T) {
	valid := CancelOrderParams{OrderID: "o", DeliveryID: "SF", WaybillID: "SF1"}
	if err := CheckCancelOrderParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	for _, mutate := range []func(*CancelOrderParams){
		func(p *CancelOrderParams) { p.OrderID = "" },
		func(p *CancelOrderParams) { p.DeliveryID = "" },
		func(p *CancelOrderParams) { p.WaybillID = "" },
	} {
		params := valid
		mutate(&params)
		if err := CheckCancelOrderParams(params); err == nil {
			t.Fatalf("invalid params accepted: %+v", params)
		}
	}
}

func TestCheckGetOrderParams(t *testing.T) {
	valid := GetOrderParams{OrderID: "o", DeliveryID: "SF"}
	if err := CheckGetOrderParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	bad := valid
	bad.PrintType = 2
	if err := CheckGetOrderParams(bad); err == nil {
		t.Fatal("print_type 2 accepted")
	}

	bad = valid
	bad.DeliveryID = ""
	if err := CheckGetOrderParams(bad); err == nil {
		t.Fatal("missing delivery_id accepted")
	}
}

func TestCheckGetPathParams(t *testing.T) {
	if err := CheckGetPathParams(GetPathParams{DeliveryID: "SF", WaybillID: "SF1"}); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	if err := CheckGetPathParams(GetPathParams{WaybillID: "SF1"}); err == nil {
		t.Fatal("missing delivery_id accepted")
	}
	if err := CheckGetPathParams(GetPathParams{DeliveryID: "SF"}); err == nil {
		t.Fatal("missing waybill_id accepted")
	}
}

func TestCheckGetQuotaParams(t *testing.T) {
	if err := CheckGetQuotaParams(GetQuotaParams{DeliveryID: "SF", BizID: "biz"}); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	if err := CheckGetQuotaParams(GetQuotaParams{BizID: "biz"}); err == nil {
		t.Fatal("missing delivery_id accepted")
	}
	if err := CheckGetQuotaParams(GetQuotaParams{DeliveryID: "SF"}); err == nil {
		t.Fatal("missing biz_id accepted")
	}
}

func TestCheckTestUpdateOrderParams(t *testing.T) {
	valid := TestUpdateOrderParams{BizID: "test_biz_id", OrderID: "o", DeliveryID: "TEST", WaybillID: "T1"}
	if err := CheckTestUpdateOrderParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	bad := valid
	bad.BizID = ""
	if err := CheckTestUpdateOrderParams(bad); err == nil {
		t.Fatal("missing biz_id accepted")
	}
}

func TestLogisticsErrMsg(t *testing.T) {
	if got := AddOrderErrMsg(40001); got != "access_token 无效或 AppSecret 错误" {
		t.Fatalf("AddOrderErrMsg = %q", got)
	}

	// An undocumented code falls back to the endpoint default rather than an empty
	// string, so a log line always says something.
	if got := AddOrderErrMsg(999999); got != AddOrderDefaultErrMsg {
		t.Fatalf("AddOrderErrMsg = %q", got)
	}

	if AddOrderErrCodeMap[40001] == "" {
		t.Fatal("AddOrderErrCodeMap lost the shared codes")
	}

	// Each endpoint gets its own copy, so one endpoint cannot mutate another's.
	AddOrderErrCodeMap[1] = "changed"
	if _, ok := GetQuotaErrCodeMap[1]; ok {
		t.Fatal("endpoint errcode maps share state")
	}
}
