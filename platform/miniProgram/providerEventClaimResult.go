package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// ProviderClaimResult is the outcome of a 无忧退 claim.
type ProviderClaimResult struct {
	// OrderNo is the payment order number the claim belongs to.
	OrderNo string `xml:"OrderNo" json:"OrderNo"`
	// Status is one of the ProviderClaimStatus* constants.
	Status int `xml:"Status" json:"Status"`
	// FinishTime is when the payout finished, as a Unix timestamp. It is only
	// meaningful on success.
	FinishTime int64 `xml:"FinishTime" json:"FinishTime"`
	// PayFailReason explains a failed payout and is empty otherwise.
	PayFailReason string `xml:"PayFailReason" json:"PayFailReason"`
}

// Claim states.
const (
	// ProviderClaimStatusSucceeded means the payout went through.
	ProviderClaimStatusSucceeded = 5
	// ProviderClaimStatusFailed means it did not.
	ProviderClaimStatusFailed = 6
)

// ProviderClaimResultEvent is the wxainsurance_claim_result callback: a 无忧退
// claim reached a final state.
//
// It is a plain push -- the helper wants "success" or an empty body back -- so the
// handler returns an error instead of a reply document.
type ProviderClaimResultEvent struct {
	EventNotify

	// UploadEvent carries the claim fields as the XML form nests them.
	UploadEvent *ProviderClaimResult `xml:"upload_event" json:"upload_event"`

	// The JSON form sends the same fields at the top level instead of nesting
	// them, so they are decoded separately here. Read them through Claim rather
	// than picking one side, otherwise a handler works with one format and
	// silently reads zeroes with the other.
	OrderNo       string `xml:"-" json:"OrderNo"`
	Status        int    `xml:"-" json:"Status"`
	FinishTime    int64  `xml:"-" json:"FinishTime"`
	PayFailReason string `xml:"-" json:"PayFailReason"`
}

// Claim returns the claim outcome, whichever form carried it.
func (e *ProviderClaimResultEvent) Claim() ProviderClaimResult {
	if e == nil {
		return ProviderClaimResult{}
	}

	if e.UploadEvent != nil {
		return *e.UploadEvent
	}

	return ProviderClaimResult{
		OrderNo:       e.OrderNo,
		Status:        e.Status,
		FinishTime:    e.FinishTime,
		PayFailReason: e.PayFailReason,
	}
}

// OK reports whether the payload carries a claim outcome.
func (e *ProviderClaimResultEvent) OK() bool {
	if e == nil || !e.EventNotify.OK() {
		return false
	}

	claim := e.Claim()

	return claim.OrderNo != "" && (claim.Status == ProviderClaimStatusSucceeded ||
		claim.Status == ProviderClaimStatusFailed)
}

// ProviderClaimResultHandler handles a wxainsurance_claim_result callback.
//
// Returning nil leaves the default "success" body, which is what this callback
// expects.
type ProviderClaimResultHandler func(
	ctx context.Context, evt *ProviderClaimResultEvent,
) error

// OnProviderClaimResult registers a handler for the wxainsurance_claim_result
// callback.
func (a *Application) OnProviderClaimResult(fn ProviderClaimResultHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+ProviderEventClaimResult,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &ProviderClaimResultEvent{}

			if err := decodeDispatched(msg, []string{ProviderEventClaimResult}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
