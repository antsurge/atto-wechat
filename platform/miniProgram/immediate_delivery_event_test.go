package miniProgram

import (
	"context"
	"strings"
	"testing"
)

// orderBody is the field block the courier-side callbacks all carry.
const orderBody = `<shopid>shop-1</shopid>` +
	`<shop_order_id>order-1</shop_order_id>` +
	`<shop_no>no-1</shop_no>` +
	`<waybill_id>wb-1</waybill_id>` +
	`<delivery_sign>sign-1</delivery_sign>`

func TestImmediateDeliveryOrderStatusEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ImmediateDeliveryOrderStatusEvent

	app.OnImmediateDeliveryOrderStatus(func(
		_ context.Context, evt *ImmediateDeliveryOrderStatusEvent,
	) (*ImmediateDeliveryReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	// status_client is a merchant-side push: it carries agent and no delivery_sign.
	payload := eventEnvelope(ImmediateDeliveryEventOrderStatus,
		`<shopid>shop-1</shopid><shop_order_id>order-1</shop_order_id>`+
			`<shop_no>no-1</shop_no><waybill_id>wb-1</waybill_id>`+
			`<action_time>1700000001</action_time><order_status>102</order_status>`+
			`<action_msg>骑手已接单</action_msg>`+
			`<agent><name>张三</name><phone>13800000000</phone></agent>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.OrderStatus != ImmediateDeliveryStatusAssigned {
		t.Fatalf("status = %d", got.OrderStatus)
	}
	if got.StatusMsg() != "分配骑手成功" {
		t.Fatalf("StatusMsg() = %q", got.StatusMsg())
	}
	if got.Terminal() {
		t.Fatal("102 must not be terminal")
	}
	if got.Agent == nil || got.Agent.Name != "张三" || got.Agent.Phone != "13800000000" {
		t.Fatalf("agent = %+v", got.Agent)
	}

	// The reply is what WeChat reads, so its header and casing matter.
	body := string(msg.Reply)
	if !strings.Contains(body, "<resultcode>0</resultcode>") {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, "<ToUserName><![CDATA[gh_platform]]></ToUserName>") {
		t.Fatalf("reply header not swapped: %s", body)
	}
}

func TestImmediateDeliveryQueryOrderEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnImmediateDeliveryQueryOrder(func(
		_ context.Context, evt *ImmediateDeliveryQueryOrderEvent,
	) (*ImmediateDeliveryReply, error) {
		return evt.StatusReply(ImmediateDeliveryStatusDelivering, ""), nil
	})

	msg, err := providerDispatch(t, app, eventEnvelope(ImmediateDeliveryEventQueryOrder, orderBody))
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, "<order_status>301</order_status>") {
		t.Fatalf("reply = %s", body)
	}
	// The delivery order id has to come back or the merchant cannot match the reply.
	if !strings.Contains(body, "<waybill_id>wb-1</waybill_id>") {
		t.Fatalf("waybill_id missing: %s", body)
	}
	// The description falls back to the documented table.
	if !strings.Contains(body, "<action_msg>配送中</action_msg>") {
		t.Fatalf("action_msg = %s", body)
	}
}

func TestImmediateDeliveryCancelOrderEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ImmediateDeliveryCancelOrderEvent

	app.OnImmediateDeliveryCancelOrder(func(
		_ context.Context, evt *ImmediateDeliveryCancelOrderEvent,
	) (*ImmediateDeliveryReply, error) {
		got = evt

		return evt.FreeReply("骑手未接单，不收费"), nil
	})

	payload := eventEnvelope(ImmediateDeliveryEventCancelOrder, orderBody+
		`<cancel_reason_id>1</cancel_reason_id><cancel_reason>暂时不需要邮寄</cancel_reason>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if got.CancelReasonID != ImmediateDeliveryCancelReasonNotNeeded {
		t.Fatalf("reason = %d", got.CancelReasonID)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, "<desc>骑手未接单，不收费</desc>") {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, "<resultcode>0</resultcode>") {
		t.Fatalf("reply = %s", body)
	}
}

func TestImmediateDeliveryPreCancelOrderEvent(t *testing.T) {
	app := newEventApp(t)

	app.OnImmediateDeliveryPreCancelOrder(func(
		_ context.Context, evt *ImmediateDeliveryPreCancelOrderEvent,
	) (*ImmediateDeliveryReply, error) {
		reply := evt.NewReply()
		reply.Succeed()
		reply.DeductFee = 2.5
		reply.Desc = "骑手已取货"

		return reply, nil
	})

	payload := eventEnvelope(ImmediateDeliveryEventPreCancelOrder, orderBody+
		`<cancel_reason_id>3</cancel_reason_id>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, "<deduct_fee>2.5</deduct_fee>") {
		t.Fatalf("reply = %s", body)
	}
}

func TestImmediateDeliveryConfirmReturnEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ImmediateDeliveryConfirmReturnEvent

	app.OnImmediateDeliveryConfirmReturn(func(
		_ context.Context, evt *ImmediateDeliveryConfirmReturnEvent,
	) (*ImmediateDeliveryReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	msg, err := providerDispatch(t,
		app, eventEnvelope(ImmediateDeliveryEventConfirmReturn, orderBody))
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() || got.WaybillID != "wb-1" {
		t.Fatalf("event = %+v", got)
	}
	if !strings.Contains(string(msg.Reply), "<resultcode>0</resultcode>") {
		t.Fatalf("reply = %s", msg.Reply)
	}
}

func TestImmediateDeliveryMockUpdateOrderEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ImmediateDeliveryMockUpdateOrderEvent

	app.OnImmediateDeliveryMockUpdateOrder(func(
		_ context.Context, evt *ImmediateDeliveryMockUpdateOrderEvent,
	) (*ImmediateDeliveryReply, error) {
		got = evt

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	payload := eventEnvelope(ImmediateDeliveryEventMockUpdateOrder, orderBody+
		`<order_status>302</order_status><action_time>1700000002</action_time>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if got.OrderStatus != ImmediateDeliveryStatusDelivered || !got.OK() {
		t.Fatalf("event = %+v", got)
	}
	if got.StatusMsg() != "配送成功" {
		t.Fatalf("StatusMsg() = %q", got.StatusMsg())
	}
	if !strings.Contains(string(msg.Reply), "<resultcode>0</resultcode>") {
		t.Fatalf("reply = %s", msg.Reply)
	}
}

func TestImmediateDeliveryJSONRequest(t *testing.T) {
	// The console can be configured to push JSON; the reply must follow the request
	// format, otherwise WeChat cannot parse it.
	app := newEventApp(t)

	app.OnImmediateDeliveryQueryOrder(func(
		_ context.Context, evt *ImmediateDeliveryQueryOrderEvent,
	) (*ImmediateDeliveryReply, error) {
		return evt.StatusReply(ImmediateDeliveryStatusDelivered, ""), nil
	})

	payload := providerJSON(ImmediateDeliveryEventQueryOrder,
		`,"shopid":"shop-1","shop_order_id":"order-1","waybill_id":"wb-1"`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	body := string(msg.Reply)
	if !strings.Contains(body, `"resultcode":0`) {
		t.Fatalf("reply = %s", body)
	}
	if !strings.Contains(body, `"order_status":302`) {
		t.Fatalf("reply = %s", body)
	}
}

func TestImmediateDeliveryReplyMissingReply(t *testing.T) {
	// Returning nil would leave the default "success" body, which WeChat treats as a
	// failed callback, so the framework has to reject it instead of silently
	// answering the wrong thing.
	app := newEventApp(t)

	app.OnImmediateDeliveryQueryOrder(func(
		_ context.Context, _ *ImmediateDeliveryQueryOrderEvent,
	) (*ImmediateDeliveryReply, error) {
		return nil, nil
	})

	_, err := providerDispatch(t,
		app, eventEnvelope(ImmediateDeliveryEventQueryOrder, orderBody))
	if err == nil {
		t.Fatal("expected an error when the handler returns no reply")
	}
}

func TestImmediateDeliveryStatusTable(t *testing.T) {
	if ImmediateDeliveryStatusMsg(102) != "分配骑手成功" {
		t.Fatalf("102 = %q", ImmediateDeliveryStatusMsg(102))
	}

	// An undocumented value must stay unknown rather than be guessed.
	if ImmediateDeliveryStatusMsg(999) != "" {
		t.Fatalf("999 = %q, want empty", ImmediateDeliveryStatusMsg(999))
	}

	terminal := []int{302, 103, 203, 204, 205, 401, 501, 502}
	for _, status := range terminal {
		if !ImmediateDeliveryStatusTerminal(status) {
			t.Fatalf("%d must be terminal", status)
		}
	}

	// 303/304/305 mean the parcel is still moving back to the merchant.
	for _, status := range []int{101, 102, 201, 202, 301, 303, 304, 305} {
		if ImmediateDeliveryStatusTerminal(status) {
			t.Fatalf("%d must not be terminal", status)
		}
	}

	for _, status := range terminal {
		want := status == ImmediateDeliveryStatusDelivered
		if ImmediateDeliveryStatusSucceeded(status) != want {
			t.Fatalf("Succeeded(%d) = %v", status, !want)
		}
	}
}
