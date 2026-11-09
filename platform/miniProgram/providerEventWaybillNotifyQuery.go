package miniProgram

import (
	"context"
)

// ProviderWaybillNotifyQueryEvent is the waybill_notify_query callback: WeChat asks
// for every tracking node of a waybill so it can show the parcel on the tracking
// page.
//
// The reply is the whole history, not a delta: WeChat replaces what it has with
// what the courier returns.
type ProviderWaybillNotifyQueryEvent struct {
	EventNotify

	// WaybillID is the waybill number.
	WaybillID string `xml:"WaybillId" json:"WaybillId"`
	// SenderPhone is the shipper's mobile number.
	SenderPhone string `xml:"SenderPhone" json:"SenderPhone"`
	// ReceiverPhone is the consignee's mobile number.
	ReceiverPhone string `xml:"ReceiverPhone" json:"ReceiverPhone"`
}

// OK reports whether the payload carries a waybill number to look up.
func (e *ProviderWaybillNotifyQueryEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.WaybillID != ""
}

// ProviderNotifyPathNode is one tracking node as this callback spells it.
//
// It differs from ProviderPathNode: this event uses PascalCase tags and carries
// the courier names inline.
type ProviderNotifyPathNode struct {
	// ActionTime is when the node happened, as a Unix timestamp.
	ActionTime int64 `xml:"ActionTime" json:"ActionTime"`
	// ActionType is the node type, one of the ProviderAction* constants.
	ActionType int `xml:"ActionType" json:"ActionType"`
	// ActionMsg is the text shown on the tracking page.
	ActionMsg string `xml:"ActionMsg" json:"ActionMsg"`
	// PickupCourierName is set once a pickup courier is assigned.
	PickupCourierName string `xml:"PickupCourierName" json:"PickupCourierName"`
	// PickupCourierPhone is the pickup courier's number.
	PickupCourierPhone string `xml:"PickupCourierPhone" json:"PickupCourierPhone"`
	// DeliveryCourierName is set once a delivery courier is assigned.
	DeliveryCourierName string `xml:"DeliveryCourierName" json:"DeliveryCourierName"`
	// DeliveryCourierPhone is the delivery courier's number.
	DeliveryCourierPhone string `xml:"DeliveryCourierPhone" json:"DeliveryCourierPhone"`
}

// ProviderWaybillNotifyQueryReply answers the waybill_notify_query callback.
type ProviderWaybillNotifyQueryReply struct {
	ProviderReply

	// WaybillCreateTime is when the waybill was created, as a Unix timestamp.
	WaybillCreateTime int64 `xml:"WaybillCreateTime" json:"WaybillCreateTime"`
	// Path is the full tracking history, oldest first.
	Path []ProviderNotifyPathNode `xml:"Path" json:"Path"`
}

// NewReply starts a reply with the header filled in.
func (e *ProviderWaybillNotifyQueryEvent) NewReply() *ProviderWaybillNotifyQueryReply {
	reply := &ProviderWaybillNotifyQueryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderWaybillNotifyQueryHandler handles a waybill_notify_query callback.
type ProviderWaybillNotifyQueryHandler func(
	ctx context.Context, evt *ProviderWaybillNotifyQueryEvent,
) (*ProviderWaybillNotifyQueryReply, error)

// OnProviderWaybillNotifyQuery registers a handler for the waybill_notify_query
// callback.
func (a *Application) OnProviderWaybillNotifyQuery(fn ProviderWaybillNotifyQueryHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventWaybillNotifyQuery,
		func() eventPayload { return &ProviderWaybillNotifyQueryEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderWaybillNotifyQueryEvent))
		},
	)
}
