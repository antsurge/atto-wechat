package miniProgram

import (
	"context"
	"strings"
	"testing"

	"github.com/antsurge/atto-wechat/core/event"
)

// providerDispatch delivers payload the way the callback server does and hands the
// message back so the test can inspect the reply.
func providerDispatch(t *testing.T, app *Application, payload string) (*event.Message, error) {
	t.Helper()

	msg, err := event.ParseMessage([]byte(payload))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}

	return msg, app.dispatcher.Dispatch(context.Background(), msg)
}

// providerJSON is a minimal JSON envelope with the given event name and body.
func providerJSON(eventName, body string) string {
	return `{"ToUserName":"gh_mini","FromUserName":"gh_platform",` +
		`"CreateTime":1700000000,"MsgType":"event","Event":"` + eventName + `"` + body + `}`
}

func TestProviderCancelWaybillEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderCancelWaybillEvent

	app.OnProviderCancelWaybill(func(
		_ context.Context, evt *ProviderCancelWaybillEvent,
	) (*ProviderCancelWaybillReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventCancelWaybill,
		`<OrderID>order-1</OrderID>`+
			`<BizID><![CDATA[biz-1]]></BizID>`+
			`<BizPwd><![CDATA[pwd]]></BizPwd>`+
			`<ShopAppID><![CDATA[wx_shop]]></ShopAppID>`+
			`<WayBillID>SF123</WayBillID>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.OrderID != "order-1" || got.BizID != "biz-1" || got.WayBillID != "SF123" {
		t.Fatalf("event = %+v", got)
	}

	// The reply must swap the two account fields: what it addresses is the pusher.
	if !strings.Contains(msg.ReplyContentType, "xml") {
		t.Fatalf("Content-Type = %q, want xml", msg.ReplyContentType)
	}

	body := string(msg.Reply)
	for _, want := range []string{
		`<ToUserName><![CDATA[gh_platform]]></ToUserName>`,
		`<FromUserName><![CDATA[gh_mini]]></FromUserName>`,
		`<OrderID>order-1</OrderID>`,
		`<WayBillID>SF123</WayBillID>`,
		`<ResultCode>0</ResultCode>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("reply missing %q:\n%s", want, body)
		}
	}
	if !strings.HasPrefix(body, "<xml>") {
		t.Fatalf("reply root = %q, want <xml>", body)
	}
}

func TestProviderCheckBizEventJSON(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderCheckBiz(func(
		_ context.Context, evt *ProviderCheckBizEvent,
	) (*ProviderCheckBizReply, error) {
		reply := evt.NewReply()
		reply.Quota = 12345
		reply.Succeed()

		return reply, nil
	})

	payload := providerJSON(ProviderEventCheckBiz,
		`,"BizID":"biz-1","BizPwd":"pwd","ShopAppID":"wx_shop"`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg.ReplyContentType, "json") {
		t.Fatalf("Content-Type = %q, want json", msg.ReplyContentType)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `"ToUserName":"gh_platform"`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `"Quota":12345`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderGetQuotaEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderGetQuota(func(
		_ context.Context, evt *ProviderGetQuotaEvent,
	) (*ProviderGetQuotaReply, error) {
		reply := evt.NewReply()
		// No explicit message: Result fills in the documented description.
		reply.Result(20007, "")

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventGetQuota,
		`<BizID><![CDATA[biz-1]]></BizID><BizPwd><![CDATA[pwd]]></BizPwd>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `<ResultCode>20007</ResultCode>`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, ProviderResultMsg(20007)) {
		t.Fatalf("reply has no documented message: %s", body)
	}
}

func TestProviderAddWaybillEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderAddWaybillEvent

	app.OnProviderAddWaybill(func(
		_ context.Context, evt *ProviderAddWaybillEvent,
	) (*ProviderAddWaybillReply, error) {
		got = evt

		data, err := ProviderWaybillData("SF_bagAddr", "广州", "SF_mark", "888-666-666")
		if err != nil {
			return nil, err
		}

		reply := evt.NewReply()
		reply.WayBillID = "SF999"
		reply.WaybillData = data
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventAddWaybill,
		`<Token>tok-1</Token>`+
			`<OrderID>order-1</OrderID>`+
			`<BizID><![CDATA[biz-1]]></BizID>`+
			`<WayBillID><![CDATA[0]]></WayBillID>`+
			`<Remark><![CDATA[易碎物品]]></Remark>`+
			`<Sender><Name><![CDATA[发货人]]></Name><Mobile><![CDATA[13800000000]]></Mobile>`+
			`<Province><![CDATA[广东省]]></Province><City><![CDATA[深圳市]]></City>`+
			`<Area><![CDATA[南山区]]></Area><Address><![CDATA[科技园 1 号]]></Address></Sender>`+
			`<Receiver><Name><![CDATA[收货人]]></Name><Mobile><![CDATA[13900000000]]></Mobile>`+
			`<Province><![CDATA[北京市]]></Province><Address><![CDATA[中关村 1 号]]></Address></Receiver>`+
			`<Cargo><Weight>1.5</Weight><Space_X>20.5</Space_X><Space_Y>15</Space_Y>`+
			`<Space_Z>10</Space_Z><Count>1</Count></Cargo>`+
			`<Insured><UseInsured>1</UseInsured><InsuredValue>10000</InsuredValue></Insured>`+
			`<Service><ServiceType>1</ServiceType><ServiceName><![CDATA[标准快递]]></ServiceName></Service>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.Token != "tok-1" || got.Sender.Name != "发货人" || got.Receiver.Address != "中关村 1 号" {
		t.Fatalf("event = %+v", got)
	}
	if got.Cargo.Weight != 1.5 || got.Cargo.SpaceX != 20.5 || got.Insured.InsuredValue != 10000 {
		t.Fatalf("cargo/insured = %+v / %+v", got.Cargo, got.Insured)
	}
	if got.Service.ServiceName != "标准快递" {
		t.Fatalf("service = %+v", got.Service)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `<WayBillID>SF999</WayBillID>`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `##SF_bagAddr##广州##SF_mark##888-666-666##`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `<Token>tok-1</Token>`) {
		t.Fatalf("token not echoed: %s", body)
	}
}

func TestProviderWaybillNotifyQueryEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderWaybillNotifyQuery(func(
		_ context.Context, evt *ProviderWaybillNotifyQueryEvent,
	) (*ProviderWaybillNotifyQueryReply, error) {
		reply := evt.NewReply()
		reply.WaybillCreateTime = 1700000000
		reply.Path = []ProviderNotifyPathNode{
			{
				ActionTime:         1700000100,
				ActionType:         ProviderActionPicked,
				ActionMsg:          "已揽收",
				PickupCourierName:  "张三",
				PickupCourierPhone: "13800000000",
			},
		}
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventWaybillNotifyQuery,
		`<WaybillId><![CDATA[SF123]]></WaybillId>`+
			`<SenderPhone><![CDATA[13800000000]]></SenderPhone>`+
			`<ReceiverPhone><![CDATA[13900000000]]></ReceiverPhone>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `<WaybillCreateTime>1700000000</WaybillCreateTime>`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `<ActionMsg>已揽收</ActionMsg>`) {
		t.Fatalf("path node not written: %s", body)
	}
	if !strings.Contains(body, `<PickupCourierPhone>13800000000</PickupCourierPhone>`) {
		t.Fatalf("courier not written: %s", body)
	}
}

func TestProviderRemindWaybillEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderRemindWaybill(func(
		_ context.Context, evt *ProviderRemindWaybillEvent,
	) (*ProviderRemindWaybillReply, error) {
		reply := evt.NewReply()
		reply.WayBillID = "SF123"
		reply.EstimatedDeliveryTime = "2026-11-10"
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventRemindWaybill,
		`<Token><![CDATA[tok-1]]></Token><OrderID><![CDATA[order-1]]></OrderID>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `<EstimatedDeliveryTime>2026-11-10</EstimatedDeliveryTime>`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderAddSingleWaybillEventJSON(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderAddSingleWaybillEvent

	app.OnProviderAddSingleWaybill(func(
		_ context.Context, evt *ProviderAddSingleWaybillEvent,
	) (*ProviderAddSingleWaybillReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.WayBillID = "SF123"
		reply.PickCode = "8-8-8888"
		reply.Succeed()

		return reply, nil
	})

	payload := providerJSON(ProviderEventAddSingleWaybill,
		`,"Token":"tok-1","OrderID":"order-1",`+
			`"Sender":{"Name":"发货人","Mobile":"13800000000","City":"深圳市","Street":"科技园路"},`+
			`"Receiver":{"Name":"收货人","Mobile":"13900000000","City":"北京市"},`+
			`"GoodDetail":{"Weight":1.2,"Special":1},`+
			`"Settingment":2,"Account":"acc","ExpectStartTime":1700000000`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.Sender.Street != "科技园路" || got.GoodDetail.Weight != 1.2 {
		t.Fatalf("event = %+v", got)
	}
	if got.Settingment != ProviderSettlementMonthly || got.Account != "acc" {
		t.Fatalf("settlement = %d / %q", got.Settingment, got.Account)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `"PickCode":"8-8-8888"`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderCancelSingleWaybillEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderCancelSingleWaybill(func(
		_ context.Context, evt *ProviderCancelSingleWaybillEvent,
	) (*ProviderReply, error) {
		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	// No WayBillID: the order was cancelled before a number was issued.
	payload := eventEnvelope(ProviderEventCancelSingleWaybill,
		`<Token><![CDATA[tok-1]]></Token><OrderID><![CDATA[order-1]]></OrderID>`+
			`<CancelID>1</CancelID><CancelReason><![CDATA[用户取消]]></CancelReason>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `<Event>cancel_single_waybill</Event>`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `<ResultCode>0</ResultCode>`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderQuerySingleWaybillFeeEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderQuerySingleWaybillFee(func(
		_ context.Context, evt *ProviderQuerySingleWaybillFeeEvent,
	) (*ProviderQuerySingleWaybillFeeReply, error) {
		reply := evt.NewReply()
		reply.Fee = 1500
		reply.OriginalFee = 1800
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventQuerySingleWaybillFee,
		`<Sender><City><![CDATA[深圳市]]></City></Sender>`+
			`<Receiver><City><![CDATA[北京市]]></City></Receiver>`+
			`<GoodDetail><Weight>1.2</Weight><Special>1</Special></GoodDetail>`+
			`<Scene>1</Scene>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `<Fee>1500</Fee>`) || !strings.Contains(body, `<OriginalFee>1800</OriginalFee>`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderQuerySingleWaybillEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderQuerySingleWaybill(func(
		_ context.Context, evt *ProviderQuerySingleWaybillEvent,
	) (*ProviderQuerySingleWaybillReply, error) {
		reply := evt.NewReply()
		reply.Fee = 1500
		reply.OnlinePay = 1
		reply.PayResult = ProviderPayResultOnline
		reply.PathInfo = &ProviderPathInfo{Nodes: []ProviderPathNode{
			{ActionTime: 1700000100, ActionType: ProviderActionPicked, ActionMsg: "已揽收"},
		}}
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventQuerySingleWaybill,
		`<Token><![CDATA[tok-1]]></Token><WaybillId><![CDATA[SF123]]></WaybillId>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	// The tracking nodes sit one level deeper and in snake_case.
	if !strings.Contains(body, `<PathInfo><path_info>`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `<action_msg>已揽收</action_msg>`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `<PayResult>1</PayResult>`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderQueryExpectTimeRangeEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderQueryExpectTimeRange(func(
		_ context.Context, evt *ProviderQueryExpectTimeRangeEvent,
	) (*ProviderQueryExpectTimeRangeReply, error) {
		reply := evt.NewReply()
		reply.ValidTimeRange = []ProviderTimeRange{
			{StartTime: 9, EndTime: 11, DayDelta: 0, Status: ProviderTimeRangeAvailable},
			{StartTime: 14, EndTime: 16, DayDelta: 1, Status: ProviderTimeRangeFull},
		}
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventQueryExpectTimeRange,
		`<Province><![CDATA[广东省]]></Province><City><![CDATA[深圳市]]></City>`+
			`<Area><![CDATA[南山区]]></Area><Street><![CDATA[科技园路]]></Street>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if strings.Contains(body, "<Asap>") {
		t.Fatalf("nil Asap must be omitted: %s", body)
	}
	if !strings.Contains(body, `<EndTime>11</EndTime>`) || !strings.Contains(body, `<Status>1</Status>`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestProviderAddExpressPathEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderAddExpressPathEvent

	app.OnProviderAddExpressPath(func(
		_ context.Context, evt *ProviderAddExpressPathEvent,
	) error {
		got = evt

		return nil
	})

	payload := eventEnvelope(ProviderEventAddExpressPath,
		`<DeliveryID><![CDATA[SF]]></DeliveryID>`+
			`<WayBillId><![CDATA[SF123]]></WayBillId>`+
			`<OrderId><![CDATA[order-1]]></OrderId>`+
			`<Version>2</Version><Count>1</Count>`+
			`<Actions><ActionTime>1700000100</ActionTime>`+
			`<ActionType>100001</ActionType><ActionMsg><![CDATA[已揽收]]></ActionMsg></Actions>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.Version != 2 || len(got.Actions) != 1 || got.Actions[0].ActionMsg != "已揽收" {
		t.Fatalf("event = %+v", got)
	}

	// This callback is a plain push: no reply body, so the server answers success.
	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestProviderNotifySingleWaybillPayEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnProviderNotifySingleWaybillPay(func(
		_ context.Context, evt *ProviderNotifySingleWaybillPayEvent,
	) (*ProviderReply, error) {
		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventNotifySingleWaybillPay,
		`<Token><![CDATA[tok-1]]></Token><WayBillID><![CDATA[SF123]]></WayBillID>`+
			`<Fee>1500</Fee><OriginalFee>1800</OriginalFee>`+
			`<PayOrderId><![CDATA[pay-1]]></PayOrderId><PayFinishTime>1700000200</PayFinishTime>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(msg.Reply), `<ResultCode>0</ResultCode>`) {
		t.Fatalf("reply = %s", msg.Reply)
	}
}

func TestProviderEvaluateAgentEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderEvaluateAgentEvent

	app.OnProviderEvaluateAgent(func(
		_ context.Context, evt *ProviderEvaluateAgentEvent,
	) (*ProviderReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventEvaluateAgent,
		`<Token><![CDATA[tok-1]]></Token><WayBillID><![CDATA[SF123]]></WayBillID>`+
			`<CourierType>1</CourierType><Score>5</Score>`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.CourierType != ProviderCourierTypeDelivery || got.Score != 5 {
		t.Fatalf("event = %+v", got)
	}
}

func TestProviderComplaintEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderComplaintEvent

	app.OnProviderComplaint(func(
		_ context.Context, evt *ProviderComplaintEvent,
	) (*ProviderReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ProviderEventComplaint,
		`<Token><![CDATA[tok-1]]></Token><WayBillID><![CDATA[SF123]]></WayBillID>`+
			`<ComplainType>1</ComplainType><Content><![CDATA[包装破损]]></Content>`+
			`<Pic><![CDATA[https://a/1.jpg|https://a/2.jpg]]></Pic>`+
			`<Name><![CDATA[张三]]></Name><Phone><![CDATA[13800000000]]></Phone>`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}

	pics := got.Pictures()
	if len(pics) != 2 || pics[1] != "https://a/2.jpg" {
		t.Fatalf("Pictures() = %v", pics)
	}
}

func TestProviderComplaintPicturesEmpty(t *testing.T) {
	if pics := (&ProviderComplaintEvent{}).Pictures(); pics != nil {
		t.Fatalf("Pictures() = %v, want nil", pics)
	}

	evt := &ProviderComplaintEvent{Pic: "  "}
	if pics := evt.Pictures(); len(pics) != 0 {
		t.Fatalf("Pictures() = %v, want empty", pics)
	}
}

func TestProviderEventUnregisteredIsIgnored(t *testing.T) {
	app := newEventApp(t)

	// Only check_biz is handled; an add_waybill push must not reach it.
	app.OnProviderCheckBiz(func(
		_ context.Context, evt *ProviderCheckBizEvent,
	) (*ProviderCheckBizReply, error) {
		t.Fatal("handler must not run for an event it was not registered for")

		return nil, nil
	})

	payload := eventEnvelope(ProviderEventAddWaybill,
		`<BizID><![CDATA[biz-1]]></BizID><Token><![CDATA[tok-1]]></Token>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatalf("unregistered event must be ignored, got %v", err)
	}
	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestProviderEventNameIsCaseInsensitive(t *testing.T) {
	app := newEventApp(t)

	called := false

	app.OnProviderGetQuota(func(
		_ context.Context, evt *ProviderGetQuotaEvent,
	) (*ProviderGetQuotaReply, error) {
		called = true

		reply := evt.NewReply()
		reply.Quota = 1
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope("GET_QUOTA", `<BizID><![CDATA[biz-1]]></BizID>`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("handler did not run for a differently-cased event name")
	}
}

func TestProviderBindResultEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderBindResultEvent

	app.OnProviderBindResult(func(_ context.Context, evt *ProviderBindResultEvent) error {
		got = evt

		return nil
	})

	// The business fields are snake_case while the header is PascalCase.
	payload := eventEnvelope(ProviderEventBindResult,
		`<errcode>0</errcode><errmsg><![CDATA[ok]]></errmsg>`+
			`<delivery_id><![CDATA[EMS]]></delivery_id>`+
			`<biz_id><![CDATA[1234567]]></biz_id>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if !got.Succeeded() {
		t.Fatalf("Succeeded() = false, errcode = %d", got.ErrCode)
	}
	if got.DeliveryID != "EMS" || got.BizID != "1234567" || got.ErrMsg != "ok" {
		t.Fatalf("event = %+v", got)
	}

	// A plain push: no reply body, so the server answers success.
	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestProviderBindResultFailedAudit(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderBindResultEvent

	app.OnProviderBindResult(func(_ context.Context, evt *ProviderBindResultEvent) error {
		got = evt

		return nil
	})

	payload := eventEnvelope(ProviderEventBindResult,
		`<errcode>40001</errcode><errmsg><![CDATA[账户不存在]]></errmsg>`+
			`<delivery_id><![CDATA[SF]]></delivery_id><biz_id><![CDATA[999]]></biz_id>`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	// A failed audit is a valid payload, not a broken one.
	if !got.OK() {
		t.Fatalf("OK() = false for a failed audit")
	}
	if got.Succeeded() {
		t.Fatal("Succeeded() = true for a failed audit")
	}
	if got.ErrCode != 40001 || got.ErrMsg != "账户不存在" {
		t.Fatalf("event = %+v", got)
	}
}

func TestProviderClaimResultEventXML(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderClaimResultEvent

	app.OnProviderClaimResult(func(_ context.Context, evt *ProviderClaimResultEvent) error {
		got = evt

		return nil
	})

	// The XML form nests the claim fields under upload_event.
	payload := eventEnvelope(ProviderEventClaimResult,
		`<upload_event><OrderNo><![CDATA[4200002166]]></OrderNo>`+
			`<Status>5</Status><FinishTime>1234455555</FinishTime></upload_event>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}

	claim := got.Claim()
	if claim.OrderNo != "4200002166" || claim.Status != ProviderClaimStatusSucceeded {
		t.Fatalf("claim = %+v", claim)
	}
	if claim.FinishTime != 1234455555 {
		t.Fatalf("finish time = %d", claim.FinishTime)
	}

	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestProviderClaimResultEventJSON(t *testing.T) {
	app := newEventApp(t)

	var got *ProviderClaimResultEvent

	app.OnProviderClaimResult(func(_ context.Context, evt *ProviderClaimResultEvent) error {
		got = evt

		return nil
	})

	// The JSON form sends the same fields at the top level.
	payload := providerJSON(ProviderEventClaimResult,
		`,"OrderNo":"4200002166","Status":6,"PayFailReason":"账户冻结"`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	claim := got.Claim()
	if claim.Status != ProviderClaimStatusFailed {
		t.Fatalf("status = %d, want failed", claim.Status)
	}
	if claim.OrderNo != "4200002166" || claim.PayFailReason != "账户冻结" {
		t.Fatalf("claim = %+v", claim)
	}
}

func TestProviderClaimResultIncomplete(t *testing.T) {
	// No order number: nothing to reconcile against.
	if (&ProviderClaimResultEvent{}).OK() {
		t.Fatal("OK() = true for an empty event")
	}

	// A status outside the documented pair is not a claim outcome.
	evt := &ProviderClaimResultEvent{
		EventNotify: EventNotify{ToUserName: "a", FromUserName: "b", Event: ProviderEventClaimResult},
		OrderNo:     "4200002166",
		Status:      1,
	}
	if evt.OK() {
		t.Fatal("OK() = true for an undocumented status")
	}
}

func TestProviderWaybillData(t *testing.T) {
	got, err := ProviderWaybillData("SF_bagAddr", "广州", "SF_mark", "888-666-666")
	if err != nil {
		t.Fatal(err)
	}

	want := "##SF_bagAddr##广州##SF_mark##888-666-666##"
	if got != want {
		t.Fatalf("ProviderWaybillData = %q, want %q", got, want)
	}

	if _, err := ProviderWaybillData("SF_bagAddr"); err == nil {
		t.Fatal("expected an error for an odd number of arguments")
	}
	if _, err := ProviderWaybillData("", "广州"); err == nil {
		t.Fatal("expected an error for an empty key")
	}
}

func TestProviderReplySucceedAndResult(t *testing.T) {
	reply := &ProviderReply{}
	reply.Succeed()

	if !reply.OK() {
		t.Fatalf("OK() = false after Succeed")
	}

	reply.Result(30002, "")
	if reply.OK() {
		t.Fatal("OK() = true for a failure code")
	}
	if reply.ResultMsg != ProviderResultMsg(30002) {
		t.Fatalf("ResultMsg = %q", reply.ResultMsg)
	}

	reply.Result(30002, "自定义原因")
	if reply.ResultMsg != "自定义原因" {
		t.Fatalf("ResultMsg = %q, want the caller's message", reply.ResultMsg)
	}
}

func TestProviderEventHandlersRejectNil(t *testing.T) {
	app := newEventApp(t)

	// Registering nothing must not panic and must not register a handler.
	app.OnProviderAddWaybill(nil).
		OnProviderCheckBiz(nil).
		OnProviderComplaint(nil).
		OnProviderAddExpressPath(nil)

	payload := eventEnvelope(ProviderEventAddWaybill, `<Token><![CDATA[tok]]></Token>`)

	// With nothing registered the push is simply unrouted: no handler, no reply.
	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatalf("dispatch = %v, want nil", err)
	}
	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}
