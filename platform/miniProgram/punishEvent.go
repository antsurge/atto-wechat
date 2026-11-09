package miniProgram

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"strings"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
)

const (
	// PunishEventName is "wxa_punish_event": the platform recorded a violation
	// against the mini program.
	PunishEventName = "wxa_punish_event"

	// PunishBanDaysPermanent is the number of days that means "forever" in every
	// ban-day field. Zero is a permanent ban, not "no ban".
	PunishBanDaysPermanent = 0
)

// PunishType values carried in EventType.
const (
	// PunishTypeWarning is a warning: the platform recorded the violation but has
	// not restricted the mini program yet.
	PunishTypeWarning = 1
	// PunishTypeFunctionBan disables specific features.
	PunishTypeFunctionBan = 2
	// PunishTypeSuspend removes the mini program from search and access.
	PunishTypeSuspend = 3
	// PunishTypeAccountBan bans the account.
	PunishTypeAccountBan = 4
	// PunishTypePageBan blocks specific pages.
	PunishTypePageBan = 10
)

// PunishTypeMap holds the EventType descriptions WeChat documents.
var PunishTypeMap = map[int]string{
	PunishTypeWarning:     "警告",
	PunishTypeFunctionBan: "功能封禁",
	PunishTypeSuspend:     "下架",
	PunishTypeAccountBan:  "账号封禁",
	PunishTypePageBan:     "页面封禁",
}

// WarnedType values carried in PunishWarningDetail.WarnedType.
const (
	// WarnedTypeAccountBan warns that the account may be banned.
	WarnedTypeAccountBan = 1
	// WarnedTypeFunctionBan warns that features may be banned.
	WarnedTypeFunctionBan = 2
	// WarnedTypeSuspend warns that the mini program may be taken down.
	WarnedTypeSuspend = 3
)

// WarnedTypeMap holds the WarnedType descriptions WeChat documents.
var WarnedTypeMap = map[int]string{
	WarnedTypeAccountBan:  "警告账号封禁",
	WarnedTypeFunctionBan: "警告功能封禁",
	WarnedTypeSuspend:     "警告下架",
}

// IllegalContent is the IllegalContent field of a punish event.
//
// The field is documented as a string but arrives as an array in the official
// example, so the type accepts both shapes instead of failing to decode.
type IllegalContent []string

// UnmarshalXML decodes the element's character data as a single entry.
func (c *IllegalContent) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value string
	if err := d.DecodeElement(&value, &start); err != nil {
		return err
	}

	*c = newIllegalContent(value)

	return nil
}

// UnmarshalJSON accepts both a JSON array and a bare string.
func (c *IllegalContent) UnmarshalJSON(data []byte) error {
	var items []string
	if err := codec.Unmarshal(data, &items); err == nil {
		*c = items

		return nil
	}

	var value string
	if err := codec.Unmarshal(data, &value); err != nil {
		return err
	}

	*c = newIllegalContent(value)

	return nil
}

// newIllegalContent keeps an empty element empty instead of producing one blank
// entry that a caller would have to filter out.
func newIllegalContent(value string) IllegalContent {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return IllegalContent{value}
}

// PunishEvent is the payload of a violation notification.
type PunishEvent struct {
	EventNotify

	// EventType is one of the PunishType constants.
	EventType int `xml:"event_type" json:"event_type"`
	// PunishID identifies this violation.
	PunishID string `xml:"punish_id" json:"punish_id"`
	// AppID is the punished mini program.
	AppID string `xml:"appid" json:"appid"`
	// PunishTime is the violation time as a Unix timestamp in seconds.
	PunishTime int64 `xml:"punish_time" json:"punish_time"`
	// IllegalReason is the violation reason.
	IllegalReason string `xml:"illegal_reason" json:"illegal_reason"`
	// IllegalContent is the flagged content.
	IllegalContent IllegalContent `xml:"illegal_content" json:"illegal_content"`
	// RuleName is the violated rule.
	RuleName string `xml:"rule_name" json:"rule_name"`
	// RuleURL links to the violated rule.
	RuleURL string `xml:"rule_url" json:"rule_url"`
	// AdjustGuideURL links to the appeal and rectification guide.
	AdjustGuideURL string `xml:"adjust_guide_url" json:"adjust_guide_url"`
	// Detail is a JSON document whose shape depends on EventType. Parse it with
	// ParseDetail rather than by hand.
	Detail string `xml:"detail" json:"detail"`
}

// OK reports whether the event identifies a punishment.
func (e *PunishEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.PunishID != "" &&
		e.AppID != ""
}

// TypeName returns the description of EventType, or the empty string when the
// value is not documented.
func (e *PunishEvent) TypeName() string {
	if e == nil {
		return ""
	}

	return PunishTypeMap[e.EventType]
}

// PunishWarningDetail is the Detail of a warning (EventType 1).
type PunishWarningDetail struct {
	// WarnedType is one of the WarnedType constants.
	WarnedType int `json:"warned_type"`
	// RectifyDeadline is the deadline to fix the violation, as a Unix timestamp.
	RectifyDeadline int64 `json:"rectify_deadline"`
	// WarnedFunctionNames lists the features at risk; it is only filled when
	// WarnedType is WarnedTypeFunctionBan.
	WarnedFunctionNames []string `json:"warned_function_names"`
	// WarnedBanDays is index-aligned with WarnedFunctionNames.
	WarnedBanDays []int `json:"warned_ban_days"`
}

// TypeName returns the description of WarnedType.
func (d *PunishWarningDetail) TypeName() string {
	if d == nil {
		return ""
	}

	return WarnedTypeMap[d.WarnedType]
}

// PunishFunctionBanDetail is the Detail of a feature ban (EventType 2).
type PunishFunctionBanDetail struct {
	// BannedDays is index-aligned with BannedFunctionNames.
	BannedDays []int `json:"banned_days"`
	// BannedFunctionNames lists the banned features.
	BannedFunctionNames []string `json:"banned_function_names"`
}

// PunishSuspendDetail is the Detail of a takedown (EventType 3).
type PunishSuspendDetail struct {
	// SuspendedDays is how long the mini program stays offline.
	SuspendedDays int `json:"suspended_days"`
}

// PunishAccountBanDetail is the Detail of an account ban (EventType 4).
type PunishAccountBanDetail struct {
	// BannedDays is how long the account stays banned.
	BannedDays int `json:"banned_days"`
}

// PunishPageBanDetail is the Detail of a page ban (EventType 10).
type PunishPageBanDetail struct {
	// Path is the blocked page path.
	Path string `json:"path"`
}

// PunishDetail is the decoded Detail of a punish event.
//
// Exactly one field is set, chosen by EventType: the documented detail
// structures disagree on the type of the ban-day fields, so one struct per
// EventType is the only way to keep "array" and "number" apart.
type PunishDetail struct {
	Warning     *PunishWarningDetail
	FunctionBan *PunishFunctionBanDetail
	Suspend     *PunishSuspendDetail
	AccountBan  *PunishAccountBanDetail
	PageBan     *PunishPageBanDetail
}

// ParseDetail decodes Detail according to EventType.
func (e *PunishEvent) ParseDetail() (*PunishDetail, error) {
	if e == nil {
		return nil, cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: punish event must not be nil")
	}

	if strings.TrimSpace(e.Detail) == "" {
		return nil, cerrors.New(cerrors.CodeInvalidResponse, "miniprogram: punish event has no detail")
	}

	detail := &PunishDetail{}

	switch e.EventType {
	case PunishTypeWarning:
		detail.Warning = &PunishWarningDetail{}
		return detail, decodePunishDetail(e.Detail, detail.Warning)
	case PunishTypeFunctionBan:
		detail.FunctionBan = &PunishFunctionBanDetail{}
		return detail, decodePunishDetail(e.Detail, detail.FunctionBan)
	case PunishTypeSuspend:
		detail.Suspend = &PunishSuspendDetail{}
		return detail, decodePunishDetail(e.Detail, detail.Suspend)
	case PunishTypeAccountBan:
		detail.AccountBan = &PunishAccountBanDetail{}
		return detail, decodePunishDetail(e.Detail, detail.AccountBan)
	case PunishTypePageBan:
		detail.PageBan = &PunishPageBanDetail{}
		return detail, decodePunishDetail(e.Detail, detail.PageBan)
	default:
		return nil, cerrors.New(
			cerrors.CodeInvalidResponse,
			"miniprogram: punish event has an unknown event_type",
		)
	}
}

// decodePunishDetail decodes the detail JSON into dst.
func decodePunishDetail(detail string, dst any) error {
	// Reject unknown fields instead of silently dropping them: a new field in a
	// documented structure is worth noticing, unlike the top-level push where
	// WeChat may add fields to any message.
	decoder := json.NewDecoder(strings.NewReader(detail))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return cerrors.Wrap(cerrors.CodeDecodeFailed, "miniprogram: decode punish detail", err)
	}

	return nil
}

// PunishHandler handles a violation notification.
type PunishHandler func(ctx context.Context, evt *PunishEvent) error

// ParsePunishEvent decodes a violation notification push.
func ParsePunishEvent(data []byte) (*PunishEvent, error) {
	evt := &PunishEvent{}

	if err := decodeEventNotify(data, []string{PunishEventName}, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// OnPunish registers a handler for violation notifications.
//
// The callback server must be configured (Config.Token or Config.AESKey) and
// mounted for the handler to ever run.
func (a *Application) OnPunish(fn PunishHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+PunishEventName,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &PunishEvent{}
			if err := decodeDispatched(msg, []string{PunishEventName}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
