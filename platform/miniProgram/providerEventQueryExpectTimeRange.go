package miniProgram

import (
	"context"
)

// ProviderQueryExpectTimeRangeEvent is the waybill_query_expect_time_range
// callback: WeChat asks which pickup slots are still bookable at an address.
//
// The payload carries only the address, so a courier that schedules by branch has
// to resolve the branch from the district itself.
type ProviderQueryExpectTimeRangeEvent struct {
	EventNotify

	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// Province is the province, e.g. 广东省.
	Province string `xml:"Province" json:"Province"`
	// City is the city, e.g. 广州市.
	City string `xml:"City" json:"City"`
	// Area is the district, e.g. 海珠区.
	Area string `xml:"Area" json:"Area"`
	// Street is the street.
	Street string `xml:"Street" json:"Street"`
	// Address is the rest of the address.
	Address string `xml:"Address" json:"Address"`
	// ID is WeChat's address id.
	ID string `xml:"Id" json:"Id"`
}

// OK reports whether the payload carries enough of an address to answer.
func (e *ProviderQueryExpectTimeRangeEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && (e.City != "" || e.Area != "")
}

// ProviderTimeRange is one bookable pickup slot.
type ProviderTimeRange struct {
	// StartTime is the start hour on a 24-hour clock.
	StartTime int `xml:"StartTime" json:"StartTime"`
	// EndTime is the end hour on a 24-hour clock.
	EndTime int `xml:"EndTime" json:"EndTime"`
	// DayDelta is 0 for today, 1 for tomorrow, 2 for the day after.
	DayDelta int `xml:"DayDelta" json:"DayDelta"`
	// Status is 0 when the slot is bookable and 1 when it is full.
	Status int `xml:"Status" json:"Status"`
}

// Slot states.
const (
	// ProviderTimeRangeAvailable means the slot can be booked.
	ProviderTimeRangeAvailable = 0
	// ProviderTimeRangeFull means the slot is used up.
	ProviderTimeRangeFull = 1
)

// ProviderQueryExpectTimeRangeReply answers the
// waybill_query_expect_time_range callback.
type ProviderQueryExpectTimeRangeReply struct {
	ProviderReply

	// ExpectTimeRange is deprecated; the helper no longer reads it.
	//
	// It is kept because the reply schema still lists it; ValidTimeRange is what
	// drives the picker.
	ExpectTimeRange int `xml:"ExpectTimeRange" json:"ExpectTimeRange"`
	// ValidTimeRange is every slot the courier offers at the address.
	ValidTimeRange []ProviderTimeRange `xml:"ValidTimeRange" json:"ValidTimeRange"`
	// Asap describes the "within two hours" option. Leave it nil when the courier
	// does not offer one: the helper hides the option when the field is absent.
	Asap *ProviderTimeRange `xml:"Asap" json:"Asap"`
}

// NewReply starts a reply with the header filled in.
func (e *ProviderQueryExpectTimeRangeEvent) NewReply() *ProviderQueryExpectTimeRangeReply {
	reply := &ProviderQueryExpectTimeRangeReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderQueryExpectTimeRangeHandler handles a waybill_query_expect_time_range
// callback.
type ProviderQueryExpectTimeRangeHandler func(
	ctx context.Context, evt *ProviderQueryExpectTimeRangeEvent,
) (*ProviderQueryExpectTimeRangeReply, error)

// OnProviderQueryExpectTimeRange registers a handler for the
// waybill_query_expect_time_range callback.
func (a *Application) OnProviderQueryExpectTimeRange(
	fn ProviderQueryExpectTimeRangeHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventQueryExpectTimeRange,
		func() eventPayload { return &ProviderQueryExpectTimeRangeEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderQueryExpectTimeRangeEvent))
		},
	)
}
