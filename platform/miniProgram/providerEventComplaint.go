package miniProgram

import (
	"context"
	"strings"
)

// ProviderComplaintEvent is the waybill_complaint callback: the user filed a
// complaint about a waybill.
//
// The courier acknowledges here and answers later through
// ProviderUpdateComplaintResult; the helper shows the user whatever that call
// carries.
type ProviderComplaintEvent struct {
	EventNotify

	// Token is the token the merchant passed when placing the order.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// WayBillID is the waybill number.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// ComplainType is the complaint category.
	ComplainType int `xml:"ComplainType" json:"ComplainType"`
	// Content is what the user wrote.
	Content string `xml:"Content" json:"Content"`
	// Pic holds the attachment URLs, joined by "|".
	Pic string `xml:"Pic" json:"Pic"`
	// Name is the contact's name.
	Name string `xml:"Name" json:"Name"`
	// Phone is the contact's phone number.
	Phone string `xml:"Phone" json:"Phone"`
}

// OK reports whether the payload carries a waybill to complain about.
func (e *ProviderComplaintEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.WayBillID != ""
}

// Pictures splits the attachment URLs the user uploaded.
//
// The helper joins them with "|", so a single attachment has no separator and a
// blank field yields no entries.
func (e *ProviderComplaintEvent) Pictures() []string {
	if e == nil || e.Pic == "" {
		return nil
	}

	parts := strings.Split(e.Pic, "|")
	pics := make([]string, 0, len(parts))

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			pics = append(pics, p)
		}
	}

	return pics
}

// NewReply starts a reply with the header filled in.
//
// The helper expects only the header and the result for this callback, so the
// reply is a plain ProviderReply.
func (e *ProviderComplaintEvent) NewReply() *ProviderReply {
	reply := &ProviderReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderComplaintHandler handles a waybill_complaint callback.
type ProviderComplaintHandler func(
	ctx context.Context, evt *ProviderComplaintEvent,
) (*ProviderReply, error)

// OnProviderComplaint registers a handler for the waybill_complaint callback.
func (a *Application) OnProviderComplaint(fn ProviderComplaintHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventComplaint,
		func() eventPayload { return &ProviderComplaintEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderComplaintEvent))
		},
	)
}
