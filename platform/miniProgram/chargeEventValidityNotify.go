package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// ChargeValidityNotifyEvent is the charge_mp_service_validity_notify alert: the SPU
// the merchant bought is about to expire.
//
// WeChat pushes it 15, 7, 3 and 1 days before the end date, so one purchase
// produces four alerts.
type ChargeValidityNotifyEvent struct {
	// EventType is always ChargeEventTypeValidity.
	EventType int `json:"event_type"`
	// AppID is the mini program the SPU belongs to.
	AppID string `json:"appid"`
	// SPUID is the purchased product.
	SPUID int64 `json:"spu_id"`
	// SPUName is its name, for logging.
	SPUName string `json:"spu_name"`
	// ValidityEndTime is the expiry date, as the string WeChat sent.
	//
	// The documented type is string and the documentation does not state a layout,
	// so it is kept verbatim instead of being parsed into a time.Time that could
	// silently mean the wrong instant. Parse it with the layout the console shows
	// for the account, once that is known.
	ValidityEndTime string `json:"validity_end_time"`
}

// OK reports whether the payload carries an expiry to warn about.
func (e *ChargeValidityNotifyEvent) OK() bool {
	return e != nil &&
		e.EventType == ChargeEventTypeValidity &&
		e.SPUID != 0 &&
		e.ValidityEndTime != ""
}

// ChargeValidityNotifyHandler handles a charge_mp_service_validity_notify alert.
//
// Returning nil leaves the default "success" body, which is what this callback
// expects.
type ChargeValidityNotifyHandler func(
	ctx context.Context, evt *ChargeValidityNotifyEvent,
) error

// OnChargeValidityNotify registers a handler for the
// charge_mp_service_validity_notify alert.
func (a *Application) OnChargeValidityNotify(fn ChargeValidityNotifyHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onChargeEvent(
		event.EventTypeRoute(ChargeEventTypeValidity),
		chargeHandler(ChargeEventTypeValidity, fn),
	)
}
