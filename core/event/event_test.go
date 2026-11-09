package event

import (
	"context"
	"encoding/xml"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/errors"
)

const textXML = `<xml>
  <ToUserName><![CDATA[gh_1]]></ToUserName>
  <FromUserName><![CDATA[user_1]]></FromUserName>
  <CreateTime>1700000000</CreateTime>
  <MsgType><![CDATA[text]]></MsgType>
  <Content><![CDATA[hello]]></Content>
  <MsgId>123456789</MsgId>
</xml>`

const subscribeXML = `<xml>
  <ToUserName><![CDATA[gh_1]]></ToUserName>
  <FromUserName><![CDATA[user_1]]></FromUserName>
  <CreateTime>1700000001</CreateTime>
  <MsgType><![CDATA[event]]></MsgType>
  <Event><![CDATA[subscribe]]></Event>
  <EventKey><![CDATA[qrscene_1]]></EventKey>
  <Ticket><![CDATA[ticket]]></Ticket>
</xml>`

func TestParseTextMessage(t *testing.T) {
	msg, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}

	if msg.MsgType != MsgTypeText || msg.Content != "hello" {
		t.Fatalf("message = %+v", msg)
	}
	if msg.FromUserName != "user_1" || msg.MsgID != 123456789 {
		t.Fatalf("message = %+v", msg)
	}
	if msg.IsEvent() {
		t.Fatal("a text message was classified as an event")
	}
}

func TestParseEventMessage(t *testing.T) {
	msg, err := ParseMessage([]byte(subscribeXML))
	if err != nil {
		t.Fatal(err)
	}

	if msg.MsgType != MsgTypeEvent || msg.Event != EventSubscribe {
		t.Fatalf("message = %+v", msg)
	}
	if msg.EventKey != "qrscene_1" || msg.Ticket != "ticket" {
		t.Fatalf("message = %+v", msg)
	}
	if !msg.IsEvent() {
		t.Fatal("an event was not classified as one")
	}
}

func TestParseIgnoresUnknownFields(t *testing.T) {
	// WeChat adds fields over time; an unknown element must not discard a
	// message the SDK could otherwise handle.
	xml := `<xml><MsgType>text</MsgType><Content>hi</Content><Future>1</Future></xml>`

	msg, err := ParseMessage([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "hi" {
		t.Fatalf("message = %+v", msg)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := ParseMessage(nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty err = %v", err)
	}
	if _, err := ParseMessage([]byte(`<xml><Content>x</Content></xml>`)); !errors.Is(err, errors.CodeInvalidResponse) {
		t.Fatalf("no msgtype err = %v", err)
	}
	if _, err := ParseMessage([]byte(`<xml><MsgType>`)); !errors.Is(err, errors.CodeDecodeFailed) {
		t.Fatalf("malformed err = %v", err)
	}
}

func TestParseJSONMessage(t *testing.T) {
	// The courier callbacks arrive as JSON when the sender is configured that way.
	data := `{"ToUserName":"gh_1","FromUserName":"gh_p","CreateTime":1700000000,` +
		`"MsgType":"event","Event":"add_waybill","Token":"tok-1"}`

	msg, err := ParseMessage([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if msg.MsgType != "event" || msg.Event != "add_waybill" {
		t.Fatalf("message = %+v", msg)
	}
	if msg.ToUserName != "gh_1" || msg.FromUserName != "gh_p" || msg.CreateTime != 1700000000 {
		t.Fatalf("header = %+v", msg)
	}
	if !msg.IsJSON() {
		t.Fatal("IsJSON() = false")
	}
}

type xmlReply struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	ResultCode int      `xml:"ResultCode"`
}

type jsonReply struct {
	ToUserName string `json:"ToUserName"`
	ResultCode int    `json:"ResultCode"`
}

func TestReplyWithMatchesRequestFormat(t *testing.T) {
	// A reply must use the format of the request, otherwise WeChat cannot read it.
	jsonMsg, err := ParseMessage([]byte(
		`{"ToUserName":"gh_1","FromUserName":"gh_p","MsgType":"event","Event":"get_quota"}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := jsonMsg.ReplyWith(jsonReply{ToUserName: "gh_p", ResultCode: 0}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonMsg.ReplyContentType, "json") {
		t.Fatalf("Content-Type = %q", jsonMsg.ReplyContentType)
	}
	if !strings.Contains(string(jsonMsg.Reply), `"ToUserName":"gh_p"`) {
		t.Fatalf("reply = %s", jsonMsg.Reply)
	}

	xmlMsg, err := ParseMessage([]byte(`<xml><MsgType>event</MsgType><Event>get_quota</Event></xml>`))
	if err != nil {
		t.Fatal(err)
	}

	if err := xmlMsg.ReplyWith(xmlReply{ToUserName: "gh_p", ResultCode: 0}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xmlMsg.ReplyContentType, "xml") {
		t.Fatalf("Content-Type = %q", xmlMsg.ReplyContentType)
	}
	if !strings.Contains(string(xmlMsg.Reply), "<ResultCode>0</ResultCode>") {
		t.Fatalf("reply = %s", xmlMsg.Reply)
	}
}

func TestReplyWithRejectsNil(t *testing.T) {
	msg, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}

	if err := msg.ReplyWith(nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil value err = %v", err)
	}

	var nilMsg *Message
	if err := nilMsg.ReplyWith(jsonReply{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil message err = %v", err)
	}
}

func TestDispatchIgnoresEventNameCase(t *testing.T) {
	// WeChat documents several event names as case-insensitive.
	var calls atomic.Int32

	d := NewDispatcher().On("event.add_waybill", func(_ context.Context, _ *Message) error {
		calls.Add(1)

		return nil
	})

	msg, err := ParseMessage([]byte(
		`<xml><MsgType>event</MsgType><Event>Add_Waybill</Event></xml>`))
	if err != nil {
		t.Fatal(err)
	}

	if err := d.Dispatch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("delivered %d messages, want 1", calls.Load())
	}
}

func TestIdentity(t *testing.T) {
	text, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}
	eventMsg, err := ParseMessage([]byte(subscribeXML))
	if err != nil {
		t.Fatal(err)
	}

	if text.Identity() == eventMsg.Identity() {
		t.Fatal("different messages share an identity")
	}
	if text.Identity() != text.Identity() {
		t.Fatal("identity is not stable")
	}

	var nilMsg *Message
	if nilMsg.Identity() != "" {
		t.Fatal("nil message has an identity")
	}

	// Events have no MsgId, so identity must still differ between two events.
	other := *eventMsg
	other.CreateTime++
	if other.Identity() == eventMsg.Identity() {
		t.Fatal("events with different timestamps share an identity")
	}
}

func TestDispatcherRoutingPrecedence(t *testing.T) {
	text, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}
	eventMsg, err := ParseMessage([]byte(subscribeXML))
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu       sync.Mutex
		received []string
	)

	record := func(name string) Handler {
		return func(_ context.Context, _ *Message) error {
			mu.Lock()
			received = append(received, name)
			mu.Unlock()

			return nil
		}
	}

	d := NewDispatcher().
		On("*", record("star")).
		On(MsgTypeText, record("text")).
		On(MsgTypeEvent+"."+EventSubscribe, record("subscribe"))

	if err := d.Dispatch(context.Background(), text); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), eventMsg); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("received = %v", received)
	}
	if received[0] != "text" {
		t.Fatalf("text was not routed to the text handler: %v", received)
	}
	if received[1] != "subscribe" {
		t.Fatalf("event was not routed to the specific handler: %v", received)
	}
}

func TestDispatcherFallsBackToStar(t *testing.T) {
	d := NewDispatcher()

	var called atomic.Bool
	d.On("*", func(context.Context, *Message) error {
		called.Store(true)

		return nil
	})

	msg, err := ParseMessage([]byte(subscribeXML))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Fatal("wildcard handler was not called")
	}
}

func TestDispatcherWithoutHandler(t *testing.T) {
	d := NewDispatcher()

	msg, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
}

func TestDispatcherReportsHandlerError(t *testing.T) {
	var observed error

	d := NewDispatcher(WithErrorHandler(func(_ context.Context, _ *Message, err error) {
		observed = err
	}))

	boom := errors.New(errors.CodeRequestFailed, "boom")
	d.On(MsgTypeText, func(context.Context, *Message) error {
		return boom
	})

	msg, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}

	err = d.Dispatch(context.Background(), msg)
	if !errors.Is(err, errors.CodeRequestFailed) {
		t.Fatalf("err = %v", err)
	}
	if observed == nil || observed.Error() != boom.Error() {
		t.Fatalf("observed = %v", observed)
	}
}

func TestDispatcherDeduplicates(t *testing.T) {
	d := NewDispatcher(WithDedup(cache.NewMemoryCache(), time.Hour))

	var calls atomic.Int32
	d.On(MsgTypeText, func(context.Context, *Message) error {
		calls.Add(1)

		return nil
	})

	msg, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}

	if err := d.Dispatch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), msg); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("duplicate err = %v, want INVALID_ARGUMENT", err)
	}

	if calls.Load() != 1 {
		t.Fatalf("handler calls = %d, want 1", calls.Load())
	}
}

func TestDispatcherDedupDoesNotBlockNewMessages(t *testing.T) {
	d := NewDispatcher(WithDedup(cache.NewMemoryCache(), time.Hour))

	var calls atomic.Int32
	d.On("*", func(context.Context, *Message) error {
		calls.Add(1)

		return nil
	})

	first, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}

	second := *first
	second.MsgID++

	if err := d.Dispatch(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), &second); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestDispatcherWithoutDedupStore(t *testing.T) {
	d := NewDispatcher()

	var calls atomic.Int32
	d.On(MsgTypeText, func(context.Context, *Message) error {
		calls.Add(1)

		return nil
	})

	msg, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if err := d.Dispatch(context.Background(), msg); err != nil {
			t.Fatal(err)
		}
	}

	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3 without dedup", calls.Load())
	}
}

func TestDispatcherRejectsNilMessage(t *testing.T) {
	if err := NewDispatcher().Dispatch(context.Background(), nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestDispatcherRegistrationIgnoresInvalidInput(t *testing.T) {
	d := NewDispatcher()

	if d.On("", func(context.Context, *Message) error { return nil }) != d {
		t.Fatal("On did not return the dispatcher")
	}
	if d.On(MsgTypeText, nil) != d {
		t.Fatal("On did not return the dispatcher")
	}
}

func TestDispatcherOptionNilIgnored(t *testing.T) {
	d := NewDispatcher(nil, WithLogger(nil))

	if d == nil || d.dedupTTL != DefaultDedupTTL {
		t.Fatalf("dispatcher = %+v", d)
	}
}

func TestRouteKey(t *testing.T) {
	msg, err := ParseMessage([]byte(subscribeXML))
	if err != nil {
		t.Fatal(err)
	}
	if routeKey(msg) != "event.subscribe" {
		t.Fatalf("route = %q", routeKey(msg))
	}

	text, err := ParseMessage([]byte(textXML))
	if err != nil {
		t.Fatal(err)
	}
	if routeKey(text) != "text" {
		t.Fatalf("route = %q", routeKey(text))
	}

	headerless, err := ParseMessage([]byte(chargeXML))
	if err != nil {
		t.Fatal(err)
	}
	if routeKey(headerless) != "event_type.3" {
		t.Fatalf("route = %q", routeKey(headerless))
	}
}

// chargeXML is the shape the 付费管理 alerts arrive in: business fields plus a
// numeric event_type, with none of the usual message header.
const chargeXML = `{"event_type":3,"appid":"wx1","spu_id":1,` +
	`"total_quota":100,"total_used_quota":80}`

func TestParseHeaderlessEvent(t *testing.T) {
	msg, err := ParseMessage([]byte(chargeXML))
	if err != nil {
		t.Fatal(err)
	}

	if msg.MsgType != "" {
		t.Fatalf("MsgType = %q", msg.MsgType)
	}
	if msg.EventType != 3 {
		t.Fatalf("EventType = %d", msg.EventType)
	}
	// IsEvent stays false: there is no Event field, so nothing may treat this as a
	// subscription-style event.
	if msg.IsEvent() {
		t.Fatal("IsEvent() = true for a headerless payload")
	}
}

func TestParseRejectsDocumentWithoutHeaderOrEventType(t *testing.T) {
	// Only a payload that identifies itself by event_type is headerless-legal.
	if _, err := ParseMessage([]byte(`{"appid":"wx1"}`)); err == nil {
		t.Fatal("expected an error")
	}
}

func TestHeaderlessIdentityIsPerPayload(t *testing.T) {
	// The quota alert fires at 20%, 10%, 5% and 0%. They share MsgType, Event and
	// sender, so identity has to come from the body -- otherwise deduplication would
	// drop three of the four warnings.
	first, err := ParseMessage([]byte(`{"event_type":3,"spu_id":1,"total_used_quota":800}`))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseMessage([]byte(`{"event_type":3,"spu_id":1,"total_used_quota":900}`))
	if err != nil {
		t.Fatal(err)
	}

	if first.Identity() == second.Identity() {
		t.Fatalf("identities collide: %q", first.Identity())
	}
	if first.Identity() == "" {
		t.Fatal("identity is empty")
	}
}

func TestDispatchHeaderlessEvent(t *testing.T) {
	msg, err := ParseMessage([]byte(chargeXML))
	if err != nil {
		t.Fatal(err)
	}

	var called bool

	d := NewDispatcher().On(EventTypeRoute(3), func(_ context.Context, _ *Message) error {
		called = true

		return nil
	})

	if err := d.Dispatch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("handler for event_type.3 was not called")
	}
}
