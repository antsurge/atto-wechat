package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// ChargeQuotaNotifyEvent is the charge_mp_service_quota_notify alert: the SPU the
// merchant bought is running out.
//
// WeChat pushes it when the remaining quota reaches 20%, 10%, 5% and 0%, so the
// same SPU produces four separate payloads and a handler must not treat the first
// one as "already warned".
type ChargeQuotaNotifyEvent struct {
	// EventType is always ChargeEventTypeQuota.
	EventType int `json:"event_type"`
	// AppID is the mini program the SPU belongs to.
	AppID string `json:"appid"`
	// SPUID is the purchased product.
	SPUID int64 `json:"spu_id"`
	// SPUName is its name, for logging.
	SPUName string `json:"spu_name"`
	// TotalQuota is how much the purchase allows in total.
	TotalQuota int64 `json:"total_quota"`
	// TotalUsedQuota is how much has been used so far.
	TotalUsedQuota int64 `json:"total_used_quota"`
}

// OK reports whether the payload carries a quota to reason about.
func (e *ChargeQuotaNotifyEvent) OK() bool {
	return e != nil &&
		e.EventType == ChargeEventTypeQuota &&
		e.SPUID != 0 &&
		e.TotalQuota > 0
}

// RemainingQuota is what is left, never negative.
func (e *ChargeQuotaNotifyEvent) RemainingQuota() int64 {
	if e == nil {
		return 0
	}

	remaining := e.TotalQuota - e.TotalUsedQuota
	if remaining < 0 {
		return 0
	}

	return remaining
}

// RemainingPercent is the share of the quota still available, in percent.
//
// It returns 0 when the total is unknown, so a caller that switches on the value to
// decide how loudly to warn stays quiet rather than dividing by zero.
func (e *ChargeQuotaNotifyEvent) RemainingPercent() float64 {
	if e == nil || e.TotalQuota <= 0 {
		return 0
	}

	return float64(e.RemainingQuota()) / float64(e.TotalQuota) * 100
}

// ChargeQuotaNotifyHandler handles a charge_mp_service_quota_notify alert.
//
// Returning nil leaves the default "success" body, which is what this callback
// expects; returning an error only logs, because WeChat retries anyway.
type ChargeQuotaNotifyHandler func(
	ctx context.Context, evt *ChargeQuotaNotifyEvent,
) error

// OnChargeQuotaNotify registers a handler for the charge_mp_service_quota_notify
// alert.
func (a *Application) OnChargeQuotaNotify(fn ChargeQuotaNotifyHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onChargeEvent(
		event.EventTypeRoute(ChargeEventTypeQuota),
		chargeHandler(ChargeEventTypeQuota, fn),
	)
}
