package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// event name of the 短剧媒资 review callback.
const MiniDramaEventReviewStatus = "secvod_audit_event"

// 剧目审核状态值。
const (
	// MiniDramaAuditStatusInvalid is 0: no meaningful state yet.
	MiniDramaAuditStatusInvalid = 0
	// MiniDramaAuditStatusReviewing is 1: still under review.
	MiniDramaAuditStatusReviewing = 1
	// MiniDramaAuditStatusRejected is 2: the final review failed.
	MiniDramaAuditStatusRejected = 2
	// MiniDramaAuditStatusApproved is 3: passed.
	MiniDramaAuditStatusApproved = 3
	// MiniDramaAuditStatusResubmit is 4: rejected but can be fixed and resubmitted.
	MiniDramaAuditStatusResubmit = 4
)

// 审核类型，即这一次提审改了什么。
const (
	// MiniDramaAuditTypeFirst is 0: the first submission.
	MiniDramaAuditTypeFirst = 0
	// MiniDramaAuditTypeAgain is 1: resubmitted.
	MiniDramaAuditTypeAgain = 1
	// MiniDramaAuditTypeReplaceEpisodes is 2: episodes were replaced.
	MiniDramaAuditTypeReplaceEpisodes = 2
	// MiniDramaAuditTypeEditInfo is 3: the drama's basic information was edited.
	MiniDramaAuditTypeEditInfo = 3
)

// MiniDramaAuditStatusMap holds the 审核状态 descriptions.
//
// Exported for the same reason the errcode maps are: callers render these states
// without copying the table.
var MiniDramaAuditStatusMap = map[int]string{
	MiniDramaAuditStatusInvalid:   "无效值",
	MiniDramaAuditStatusReviewing: "审核中",
	MiniDramaAuditStatusRejected:  "审核驳回",
	MiniDramaAuditStatusApproved:  "审核通过",
	MiniDramaAuditStatusResubmit:  "驳回重填",
}

// MiniDramaAuditTypeMap holds the 审核类型 descriptions.
var MiniDramaAuditTypeMap = map[int]string{
	MiniDramaAuditTypeFirst:           "首次提审",
	MiniDramaAuditTypeAgain:           "再次提审",
	MiniDramaAuditTypeReplaceEpisodes: "替换剧集提审",
	MiniDramaAuditTypeEditInfo:        "修改剧目基本信息审核",
}

// MiniDramaAuditStatusMsg returns the description of a 审核状态 value, or "" when
// the value is undocumented.
func MiniDramaAuditStatusMsg(status int) string {
	return MiniDramaAuditStatusMap[status]
}

// MiniDramaAuditTypeMsg returns the description of a 审核类型 value, or "" when the
// value is undocumented.
func MiniDramaAuditTypeMsg(auditType int) string {
	return MiniDramaAuditTypeMap[auditType]
}

// MiniDramaAuditDetail is the review outcome of a drama.
type MiniDramaAuditDetail struct {
	// Status is one of the MiniDramaAuditStatus* constants.
	//
	// A single episode can pass while the drama as a whole does not; whether the
	// media is playable follows the drama's status, not the episode's.
	Status int `xml:"status" json:"status"`
	// AuditType is one of the MiniDramaAuditType* constants. Only this event
	// carries it; the query APIs omit it.
	AuditType int `xml:"audit_type" json:"audit_type"`
	// CreateTime is when the review was requested, as a Unix timestamp.
	CreateTime int64 `xml:"create_time" json:"create_time"`
	// AuditTime is when it finished, as a Unix timestamp.
	AuditTime int64 `xml:"audit_time" json:"audit_time"`
}

// StatusMsg returns the documented description of the state, or "" when unknown.
func (d MiniDramaAuditDetail) StatusMsg() string {
	return MiniDramaAuditStatusMsg(d.Status)
}

// AuditTypeMsg returns the documented description of the audit type, or "".
func (d MiniDramaAuditDetail) AuditTypeMsg() string {
	return MiniDramaAuditTypeMsg(d.AuditType)
}

// Settled reports whether the review reached an outcome the merchant acted on.
//
// 审核中 is not settled, and neither is 无效值: both mean "wait".
func (d MiniDramaAuditDetail) Settled() bool {
	switch d.Status {
	case MiniDramaAuditStatusRejected,
		MiniDramaAuditStatusApproved,
		MiniDramaAuditStatusResubmit:
		return true
	}

	return false
}

// MiniDramaReviewStatusEvent is the secvod_audit_event callback: a drama's review
// reached a decision.
type MiniDramaReviewStatusEvent struct {
	EventNotify

	// AuditEvent carries the outcome, as the documented XML nests it.
	AuditEvent *miniDramaReviewAuditEvent `xml:"audit_event" json:"audit_event"`
}

// miniDramaReviewAuditEvent is the inner element of the review push.
type miniDramaReviewAuditEvent struct {
	// DramaID is the drama the review belongs to.
	DramaID int64 `xml:"drama_id" json:"drama_id"`
	// AuditDetail is the outcome.
	AuditDetail *MiniDramaAuditDetail `xml:"audit_detail" json:"audit_detail"`
}

// DramaID returns the reviewed drama, or 0 when the payload carried none.
func (e *MiniDramaReviewStatusEvent) DramaID() int64 {
	if e == nil || e.AuditEvent == nil {
		return 0
	}

	return e.AuditEvent.DramaID
}

// Audit returns the review outcome, or a zero value when the payload carried none.
func (e *MiniDramaReviewStatusEvent) Audit() MiniDramaAuditDetail {
	if e == nil || e.AuditEvent == nil || e.AuditEvent.AuditDetail == nil {
		return MiniDramaAuditDetail{}
	}

	return *e.AuditEvent.AuditDetail
}

// OK reports whether the payload carries a review to act on.
func (e *MiniDramaReviewStatusEvent) OK() bool {
	if e == nil || !e.EventNotify.OK() || e.DramaID() == 0 {
		return false
	}

	return e.AuditEvent.AuditDetail != nil
}

// MiniDramaReviewStatusHandler handles a secvod_audit_event callback.
type MiniDramaReviewStatusHandler func(
	ctx context.Context, evt *MiniDramaReviewStatusEvent,
) error

// OnMiniDramaReviewStatus registers a handler for the secvod_audit_event callback.
func (a *Application) OnMiniDramaReviewStatus(fn MiniDramaReviewStatusHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+MiniDramaEventReviewStatus,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &MiniDramaReviewStatusEvent{}

			if err := decodeDispatched(msg, []string{MiniDramaEventReviewStatus}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
