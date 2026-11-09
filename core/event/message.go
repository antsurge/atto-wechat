// Package event parses WeChat's inbound callback messages and routes them to
// handlers.
//
// Parsing and routing are kept apart from HTTP (core/server) so that a handler,
// a test, or a replay tool can feed a message in without standing up a web
// server.
package event

import (
	"github.com/antsurge/atto-wechat/core/codec"
	"github.com/antsurge/atto-wechat/core/errors"
)

// MsgType values WeChat sends in the MsgType field.
const (
	MsgTypeText       = "text"
	MsgTypeImage      = "image"
	MsgTypeVoice      = "voice"
	MsgTypeVideo      = "video"
	MsgTypeShortVideo = "shortvideo"
	MsgTypeLocation   = "location"
	MsgTypeLink       = "link"
	MsgTypeEvent      = "event"
)

// Event values carried in the Event field when MsgType is event.
const (
	EventSubscribe       = "subscribe"
	EventUnsubscribe     = "unsubscribe"
	EventScan            = "SCAN"
	EventLocation        = "LOCATION"
	EventClick           = "CLICK"
	EventView            = "VIEW"
	EventScanCodePush    = "scancode_push"
	EventScanCodeWaitMsg = "scancode_waitmsg"
	EventPicSysPhoto     = "pic_sysphoto"
	EventPicPhotoOrAlbum = "pic_photo_or_album"
	EventPicWeixin       = "pic_weixin"
	EventLocationSelect  = "location_select"

	// EventTemplateSendJob is the async delivery status of a template message.
	EventTemplateSendJob = "TEMPLATESENDJOBFINISH"
)

// Message is a callback message.
//
// Only the fields this SDK understands are decoded; XML elements it does not
// know are ignored rather than treated as errors, because WeChat adds fields to
// existing message types over time and an unknown field must not discard an
// otherwise valid message.
type Message struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	MsgID        int64  `xml:"MsgId"`

	// Content of a text message.
	Content string `xml:"Content"`

	// Event fields.
	Event    string `xml:"Event"`
	EventKey string `xml:"EventKey"`
	Ticket   string `xml:"Ticket"`

	// EventType is the numeric marker the 付费管理 (charge) events carry instead of
	// a message header.
	//
	// WeChat documents those two payloads as a flat JSON body with event_type and
	// none of ToUserName / FromUserName / CreateTime / MsgType / Event, so there is
	// nothing else to route them by.
	//
	// It is filled by ParseMessage rather than by a struct tag on purpose: the field
	// name also appears inside other events (wxa_punish_event), where a value that is
	// not an integer must stay that event's problem instead of failing the whole
	// document here. Tagged decoding would turn such a payload into a parse error for
	// every caller.
	EventType int `xml:"-" json:"-"`

	// Location messages, and the LOCATION event.
	Latitude  float64 `xml:"Latitude"`
	Longitude float64 `xml:"Longitude"`
	Precision float64 `xml:"Precision"`
	LocationX float64 `xml:"Location_X"`
	LocationY float64 `xml:"Location_Y"`
	Scale     int     `xml:"Scale"`
	Label     string  `xml:"Label"`

	// Media messages.
	MediaID      string `xml:"MediaId"`
	ThumbMediaID string `xml:"ThumbMediaId"`
	Format       string `xml:"Format"`
	Recognition  string `xml:"Recognition"`
	PicURL       string `xml:"PicUrl"`

	// Link messages.
	Title       string `xml:"Title"`
	Description string `xml:"Description"`
	URL         string `xml:"Url"`

	// Raw is the document the message was parsed from.
	//
	// The fields above are the ones every platform shares; a platform package
	// decodes the parts it knows from Raw instead of widening this struct with
	// fields that only one message type carries.
	Raw []byte `xml:"-"`

	// Reply is the body sent back to WeChat instead of the default "success".
	//
	// Most callbacks only need an acknowledgement, but some -- the courier
	// (运力方) events are the main case -- expect a structured answer that
	// mirrors the request. A handler fills this through ReplyWith.
	Reply []byte `xml:"-"`
	// ReplyContentType is the Content-Type of Reply, set by ReplyWith.
	ReplyContentType string `xml:"-"`
}

// ParseMessage decodes a callback document.
//
// Chat callbacks arrive as XML, but the courier (运力方) callbacks of the
// logistics helper arrive as either XML or JSON depending on how the sender is
// configured, so both shapes are accepted.
func ParseMessage(data []byte) (*Message, error) {
	if len(data) == 0 {
		return nil, errors.New(errors.CodeInvalidArgument, "empty callback message")
	}

	var msg Message
	var err error

	if isJSONDocument(data) {
		err = codec.Unmarshal(data, &msg)
	} else {
		err = codec.XMLUnmarshal(data, &msg)
	}
	if err != nil {
		return nil, errors.Wrap(errors.CodeDecodeFailed, "decode callback message", err)
	}

	if msg.MsgType == "" {
		// A headerless payload is only valid when it identifies itself by event_type;
		// anything else is still rejected, so this does not turn the parser into a
		// catch-all for malformed bodies.
		eventType, ok := headerlessEventType(data)
		if !ok {
			return nil, errors.New(errors.CodeInvalidResponse, "callback message has no MsgType")
		}

		msg.EventType = eventType
	}

	msg.Raw = data

	return &msg, nil
}

// headerlessEventType reads the event_type of a payload that carries no message
// header.
//
// The value is taken out of the body separately instead of through a struct tag so
// that a non-integer event_type stays the owning event's problem: the same field
// name appears in wxa_punish_event, and failing the whole document here would drop
// a push that event's own decoder could have handled.
func headerlessEventType(data []byte) (int, bool) {
	if !isJSONDocument(data) {
		return 0, false
	}

	var probe struct {
		EventType *int `json:"event_type"`
	}
	if err := codec.Unmarshal(data, &probe); err != nil || probe.EventType == nil {
		return 0, false
	}

	return *probe.EventType, true
}

// isJSONDocument reports whether data starts with a JSON object.
func isJSONDocument(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}

	return false
}

// IsJSON reports whether the callback arrived as JSON rather than XML.
//
// A reply must use the same format as the request, which is what ReplyWith
// relies on.
func (m *Message) IsJSON() bool {
	return m != nil && isJSONDocument(m.Raw)
}

// ReplyWith encodes v in the format the callback arrived in and sets it as the
// reply sent to WeChat.
//
// For XML, v must declare its root element with an XMLName field tagged
// `xml:"xml"`; for JSON the tags of v are used as they are. The reply replaces
// the default "success" body, so a handler that calls it takes over the
// response entirely.
func (m *Message) ReplyWith(v any) error {
	if m == nil {
		return errors.New(errors.CodeInvalidArgument, "message must not be nil")
	}
	if v == nil {
		return errors.New(errors.CodeInvalidArgument, "reply value must not be nil")
	}

	if m.IsJSON() {
		body, err := codec.Marshal(v)
		if err != nil {
			return errors.Wrap(errors.CodeEncodeFailed, "encode JSON reply", err)
		}

		m.Reply = body
		m.ReplyContentType = "application/json; charset=utf-8"

		return nil
	}

	body, err := codec.XMLMarshal(v)
	if err != nil {
		return errors.Wrap(errors.CodeEncodeFailed, "encode XML reply", err)
	}

	m.Reply = body
	m.ReplyContentType = "application/xml; charset=utf-8"

	return nil
}

// IsEvent reports whether the message is an event notification rather than a
// user message.
func (m *Message) IsEvent() bool {
	return m != nil && m.MsgType == MsgTypeEvent
}

// Identity returns a stable identifier for deduplication.
//
// WeChat retries a callback when the reply takes longer than five seconds, so
// the same message can arrive more than once. Text and media messages carry a
// MsgId that serves as identity; events do not, so the sender and timestamp are
// combined instead. That is the best available key, not a perfect one.
func (m *Message) Identity() string {
	if m == nil {
		return ""
	}

	if m.MsgID > 0 {
		return "msgid:" + itoa(m.MsgID)
	}

	// A headerless payload has no sender, no time and no event name, so every
	// warning of the same kind would hash to the same identity and deduplication
	// would drop everything after the first one. The body distinguishes them: the
	// quota warning fires at 20%, 10%, 5% and 0%, each a separate payload.
	if m.MsgType == "" {
		return "raw:" + itoa(int64(hashBody(m.Raw)&maxInt63)) + ":" + itoa(int64(m.EventType))
	}

	return "evt:" + m.FromUserName + ":" + itoa(m.CreateTime) + ":" + m.MsgType + ":" + m.Event + ":" + m.EventKey
}

// maxInt63 is the largest positive value itoa can format.
const maxInt63 = 1<<63 - 1

// hashBody is FNV-1a over the raw body.
//
// It is inlined instead of pulling in hash/fnv because the only caller needs a
// short, stable identifier, not a general-purpose hash.
func hashBody(data []byte) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	h := uint64(offset64)
	for _, b := range data {
		h ^= uint64(b)
		h *= prime64
	}

	return h
}

// itoa formats a positive int64 without pulling in strconv semantics for
// negatives, which cannot occur in these identifiers.
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)

	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}

	return string(buf[i:])
}
