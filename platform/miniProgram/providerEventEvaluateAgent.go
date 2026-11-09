package miniProgram

import (
	"context"
)

// ProviderEvaluateAgentEvent is the waybill_evaluate_agent callback: the user rated
// the courier who handled the parcel.
//
// Score is what the user gave; the courier decides what to do with it. There is no
// comment field, so a low score arrives without a reason.
type ProviderEvaluateAgentEvent struct {
	EventNotify

	// Token is the token the merchant passed when placing the order.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// WayBillID is the waybill number.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// CourierType is 0 for the pickup courier and 1 for the delivery courier.
	CourierType int `xml:"CourierType" json:"CourierType"`
	// Score is 1 to 5.
	Score int `xml:"Score" json:"Score"`
}

// Couriers a rating can target.
const (
	// ProviderCourierTypePickup is the courier who collected the parcel.
	ProviderCourierTypePickup = 0
	// ProviderCourierTypeDelivery is the courier who delivered it.
	ProviderCourierTypeDelivery = 1
)

// OK reports whether the payload carries a waybill and a score in range.
func (e *ProviderEvaluateAgentEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.WayBillID != "" &&
		e.Score >= 1 &&
		e.Score <= 5
}

// NewReply starts a reply with the header filled in.
//
// The helper expects only the header and the result for this callback, so the
// reply is a plain ProviderReply.
func (e *ProviderEvaluateAgentEvent) NewReply() *ProviderReply {
	reply := &ProviderReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderEvaluateAgentHandler handles a waybill_evaluate_agent callback.
type ProviderEvaluateAgentHandler func(
	ctx context.Context, evt *ProviderEvaluateAgentEvent,
) (*ProviderReply, error)

// OnProviderEvaluateAgent registers a handler for the waybill_evaluate_agent
// callback.
func (a *Application) OnProviderEvaluateAgent(fn ProviderEvaluateAgentHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventEvaluateAgent,
		func() eventPayload { return &ProviderEvaluateAgentEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderEvaluateAgentEvent))
		},
	)
}
