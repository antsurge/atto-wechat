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
	EventSubscribe        = "subscribe"
	EventUnsubscribe      = "unsubscribe"
	EventScan             = "SCAN"
	EventLocation         = "LOCATION"
	EventClick            = "CLICK"
	EventView             = "VIEW"
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
}

// ParseMessage decodes a callback XML document.
func ParseMessage(data []byte) (*Message, error) {
	if len(data) == 0 {
		return nil, errors.New(errors.CodeInvalidArgument, "empty callback message")
	}

	var msg Message
	if err := codec.XMLUnmarshal(data, &msg); err != nil {
		return nil, errors.Wrap(errors.CodeDecodeFailed, "decode callback message", err)
	}

	if msg.MsgType == "" {
		return nil, errors.New(errors.CodeInvalidResponse, "callback message has no MsgType")
	}

	return &msg, nil
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

	return "evt:" + m.FromUserName + ":" + itoa(m.CreateTime) + ":" + m.MsgType + ":" + m.Event + ":" + m.EventKey
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
