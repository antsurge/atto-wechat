package miniProgram

import (
	"context"
	"strings"
	"testing"

	"github.com/antsurge/atto-wechat/core/event"
)

func TestChargeQuotaNotifyEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ChargeQuotaNotifyEvent

	app.OnChargeQuotaNotify(func(_ context.Context, evt *ChargeQuotaNotifyEvent) error {
		got = evt

		return nil
	})

	// The charge alerts carry no message header at all: no ToUserName, no MsgType,
	// no Event. They are routed by event_type alone.
	payload := `{"event_type":3,"appid":"wxapp","spu_id":1001,` +
		`"spu_name":"AI 能力包","total_quota":1000,"total_used_quota":950}`

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("handler was not called")
	}
	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.SPUID != 1001 || got.TotalQuota != 1000 || got.TotalUsedQuota != 950 {
		t.Fatalf("event = %+v", got)
	}
	if got.RemainingQuota() != 50 {
		t.Fatalf("remaining = %d", got.RemainingQuota())
	}
	if got.RemainingPercent() != 5 {
		t.Fatalf("remaining percent = %v", got.RemainingPercent())
	}
	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestChargeQuotaNotifyRoutesByEventType(t *testing.T) {
	// The two alerts share the empty MsgType, so the only thing keeping them apart
	// is event_type. Registering both proves routing does not merge them.
	app := newEventApp(t)

	var quota, validity bool

	app.OnChargeQuotaNotify(func(_ context.Context, _ *ChargeQuotaNotifyEvent) error {
		quota = true

		return nil
	})
	app.OnChargeValidityNotify(func(_ context.Context, _ *ChargeValidityNotifyEvent) error {
		validity = true

		return nil
	})

	if _, err := providerDispatch(t, app, `{"event_type":3,"spu_id":1,"total_quota":10}`); err != nil {
		t.Fatal(err)
	}
	if !quota || validity {
		t.Fatalf("quota = %v, validity = %v", quota, validity)
	}

	quota = false

	if _, err := providerDispatch(t, app,
		`{"event_type":4,"spu_id":2,"validity_end_time":"2026-12-01 00:00:00"}`); err != nil {
		t.Fatal(err)
	}
	if quota || !validity {
		t.Fatalf("quota = %v, validity = %v", quota, validity)
	}
}

func TestChargeValidityNotifyEvent(t *testing.T) {
	app := newEventApp(t)

	var got *ChargeValidityNotifyEvent

	app.OnChargeValidityNotify(func(_ context.Context, evt *ChargeValidityNotifyEvent) error {
		got = evt

		return nil
	})

	payload := `{"event_type":4,"appid":"wxapp","spu_id":2002,` +
		`"spu_name":"AI 能力包","validity_end_time":"2026-12-01 00:00:00"}`

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	// The layout is undocumented, so it is kept verbatim rather than parsed.
	if got.ValidityEndTime != "2026-12-01 00:00:00" {
		t.Fatalf("validity end = %q", got.ValidityEndTime)
	}
}

func TestChargePayloadWithoutEventTypeIsRejected(t *testing.T) {
	// Relaxing the MsgType requirement must not turn the parser into a catch-all:
	// only a payload that identifies itself by event_type is headerless-legal.
	if _, err := event.ParseMessage([]byte(`{"appid":"wxapp"}`)); err == nil {
		t.Fatal("expected a payload with neither MsgType nor event_type to be rejected")
	}
}

func TestMiniDramaAssetUploadEvent(t *testing.T) {
	app := newEventApp(t)

	var got *MiniDramaAssetUploadEvent

	app.OnMiniDramaAssetUpload(func(_ context.Context, evt *MiniDramaAssetUploadEvent) error {
		got = evt

		return nil
	})

	payload := eventEnvelope(MiniDramaEventAssetUpload,
		`<upload_event><media_id>20001</media_id>`+
			`<source_context><![CDATA[abc12232]]></source_context>`+
			`<errcode>0</errcode><errmsg><![CDATA[OK]]></errmsg></upload_event>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}

	upload := got.Upload()
	if upload.MediaID != 20001 || upload.SourceContext != "abc12232" {
		t.Fatalf("upload = %+v", upload)
	}
	if !upload.Succeeded() {
		t.Fatalf("Succeeded() = false, errcode = %d", upload.ErrCode)
	}

	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestMiniDramaAssetUploadFailed(t *testing.T) {
	app := newEventApp(t)

	var got *MiniDramaAssetUploadEvent

	app.OnMiniDramaAssetUpload(func(_ context.Context, evt *MiniDramaAssetUploadEvent) error {
		got = evt

		return nil
	})

	payload := eventEnvelope(MiniDramaEventAssetUpload,
		`<upload_event><media_id>20002</media_id><errcode>40001</errcode>`+
			`<errmsg><![CDATA[格式不支持]]></errmsg></upload_event>`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	// A failed upload is a complete payload: the merchant still needs to know.
	if !got.OK() {
		t.Fatal("OK() = false for a failed upload")
	}
	if got.Upload().Succeeded() {
		t.Fatal("Succeeded() = true for a failed upload")
	}
	if got.Upload().ErrMsg != "格式不支持" {
		t.Fatalf("errmsg = %q", got.Upload().ErrMsg)
	}
}

func TestMiniDramaReviewStatusEvent(t *testing.T) {
	app := newEventApp(t)

	var got *MiniDramaReviewStatusEvent

	app.OnMiniDramaReviewStatus(func(_ context.Context, evt *MiniDramaReviewStatusEvent) error {
		got = evt

		return nil
	})

	payload := eventEnvelope(MiniDramaEventReviewStatus,
		`<audit_event><drama_id>20001</drama_id><audit_detail>`+
			`<status>3</status><audit_type>1</audit_type>`+
			`<create_time>168625255</create_time><audit_time>168626255</audit_time>`+
			`</audit_detail></audit_event>`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.DramaID() != 20001 {
		t.Fatalf("drama id = %d", got.DramaID())
	}

	audit := got.Audit()
	if audit.Status != MiniDramaAuditStatusApproved || !audit.Settled() {
		t.Fatalf("audit = %+v", audit)
	}
	if audit.StatusMsg() != "审核通过" {
		t.Fatalf("status msg = %q", audit.StatusMsg())
	}
	if audit.AuditType != MiniDramaAuditTypeAgain || audit.AuditTypeMsg() != "再次提审" {
		t.Fatalf("audit type = %d %q", audit.AuditType, audit.AuditTypeMsg())
	}
	if audit.AuditTime != 168626255 {
		t.Fatalf("audit time = %d", audit.AuditTime)
	}
}

func TestMiniDramaAuditTables(t *testing.T) {
	// 审核中 is not an outcome, so a handler must keep waiting.
	reviewing := MiniDramaAuditDetail{Status: MiniDramaAuditStatusReviewing}
	if reviewing.Settled() {
		t.Fatal("审核中 must not be settled")
	}

	// An undocumented status must stay unknown instead of being guessed.
	if MiniDramaAuditStatusMsg(99) != "" {
		t.Fatalf("99 = %q, want empty", MiniDramaAuditStatusMsg(99))
	}
	if MiniDramaAuditTypeMsg(99) != "" {
		t.Fatalf("audit type 99 = %q, want empty", MiniDramaAuditTypeMsg(99))
	}
}

func TestCommentBadScoreEvent(t *testing.T) {
	app := newEventApp(t)

	var got *CommentBadScoreEvent

	app.OnCommentBadScore(func(_ context.Context, evt *CommentBadScoreEvent) error {
		got = evt

		return nil
	})

	// XML nests the review id under <result>.
	payload := eventEnvelope(CommentEventBadScore,
		`<result><comment_id>2272502024443330610</comment_id></result>`)

	msg, err := providerDispatch(t, app, payload)
	if err != nil {
		t.Fatal(err)
	}

	if !got.OK() {
		t.Fatalf("OK() = false, event = %+v", got)
	}
	if got.CommentID() != "2272502024443330610" {
		t.Fatalf("comment id = %q", got.CommentID())
	}

	if len(msg.Reply) != 0 {
		t.Fatalf("reply = %q, want empty", msg.Reply)
	}
}

func TestCommentBadScoreEventJSON(t *testing.T) {
	app := newEventApp(t)

	var got *CommentBadScoreEvent

	app.OnCommentBadScore(func(_ context.Context, evt *CommentBadScoreEvent) error {
		got = evt

		return nil
	})

	payload := providerJSON(CommentEventBadScore,
		`,"result":{"comment_id":"2272502024443330610"}`)

	if _, err := providerDispatch(t, app, payload); err != nil {
		t.Fatal(err)
	}

	if got.CommentID() != "2272502024443330610" {
		t.Fatalf("comment id = %q", got.CommentID())
	}
}

func TestCommentBadScoreIncomplete(t *testing.T) {
	if (&CommentBadScoreEvent{}).OK() {
		t.Fatal("OK() = true without a review id")
	}

	evt := &CommentBadScoreEvent{
		EventNotify: EventNotify{ToUserName: "gh_mini", Event: CommentEventBadScore},
		Result:      &commentBadScoreResult{},
	}
	if evt.OK() {
		t.Fatal("OK() = true with an empty review id")
	}
	if !strings.Contains(evt.Event, "bad_score") {
		t.Fatalf("event = %q", evt.Event)
	}
}
