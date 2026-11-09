package miniProgram

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// sendCustomMessagePath delivers a customer service message to one user.
const (
	sendCustomMessagePath          = "/cgi-bin/message/custom/send"
	SendCustomMessageDefaultErrMsg = "系统错误，请稍后再试"
)

// Message types the mini program customer service endpoint accepts.
//
// The mini program variant is narrower than the official account one: voice,
// video, file and msgmenu exist there but not here.
const (
	CustomMsgTypeText            = "text"
	CustomMsgTypeImage           = "image"
	CustomMsgTypeLink            = "link"
	CustomMsgTypeMiniProgramPage = "miniprogrampage"
)

// SendCustomMessageErrCodeMap holds the errcode descriptions WeChat documents
// for /cgi-bin/message/custom/send, keyed by the errcode as it appears in the
// reply.
//
// 45015 and 45047 are not in the documented table but are the two failures a
// production integration meets: the 48-hour reply window has closed, or the
// 5-message quota of that window is exhausted.
var SendCustomMessageErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40003: "touser 不正确",
	40013: "AppID 无效",
	42001: "access_token 已过期",
	45015: "回复时间超过限制，用户已超过 48 小时未与客服交互",
	45047: "客服接口下行条数超过上限",
	70000: "为保护未成年人权益，该条消息发送失败",
}

// CustomText is the payload of a text message.
type CustomText struct {
	// Content is the message body. It may contain a link that opens the mini
	// program, expressed as HTML with data-miniprogram-appid/data-miniprogram-path.
	Content string `json:"content"`
}

// CustomImage is the payload of an image message.
type CustomImage struct {
	// MediaID is the id returned by UploadTempMedia.
	MediaID string `json:"media_id"`
}

// CustomLink is the payload of a link message.
type CustomLink struct {
	// Title is the headline of the card.
	Title string `json:"title"`

	// Description is the body text.
	Description string `json:"description,omitempty"`

	// URL is the page opened when the card is tapped.
	URL string `json:"url,omitempty"`

	// ThumbURL is the cover image, a JPG or PNG URL.
	ThumbURL string `json:"thumb_url,omitempty"`
}

// CustomMiniProgramPage is the payload of a mini program card message.
type CustomMiniProgramPage struct {
	// Title is the headline of the card.
	Title string `json:"title"`

	// PagePath is the page opened by the card, query included.
	PagePath string `json:"pagepath,omitempty"`

	// ThumbMediaID is the cover image, a media id from UploadTempMedia.
	ThumbMediaID string `json:"thumb_media_id,omitempty"`
}

// SendCustomMessageParams describes one customer service message.
//
// Exactly one payload field is used, the one named by MsgType; the others stay
// nil and are omitted from the body.
type SendCustomMessageParams struct {
	// ToUser is the openid of the recipient.
	ToUser string

	// MsgType is one of the CustomMsgType* constants.
	MsgType string

	// Text is set when MsgType is CustomMsgTypeText.
	Text *CustomText

	// Image is set when MsgType is CustomMsgTypeImage.
	Image *CustomImage

	// Link is set when MsgType is CustomMsgTypeLink.
	Link *CustomLink

	// MiniProgramPage is set when MsgType is CustomMsgTypeMiniProgramPage.
	MiniProgramPage *CustomMiniProgramPage
}

// CheckSendCustomMessageParams validates params before anything is sent to
// WeChat.
func CheckSendCustomMessageParams(params SendCustomMessageParams) error {
	if params.ToUser == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: touser is required",
		)
	}

	switch params.MsgType {
	case CustomMsgTypeText:
		if params.Text == nil || params.Text.Content == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: text.content is required",
			)
		}
	case CustomMsgTypeImage:
		if params.Image == nil || params.Image.MediaID == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: image.media_id is required",
			)
		}
	case CustomMsgTypeLink:
		if params.Link == nil || params.Link.Title == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: link.title is required",
			)
		}
	case CustomMsgTypeMiniProgramPage:
		if params.MiniProgramPage == nil || params.MiniProgramPage.Title == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: miniprogrampage.title is required",
			)
		}
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			fmt.Sprintf(
				"miniprogram: msgtype must be %q, %q, %q or %q",
				CustomMsgTypeText,
				CustomMsgTypeImage,
				CustomMsgTypeLink,
				CustomMsgTypeMiniProgramPage,
			),
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p SendCustomMessageParams) body() *sendCustomMessageBody {
	return &sendCustomMessageBody{
		ToUser:          p.ToUser,
		MsgType:         p.MsgType,
		Text:            p.Text,
		Image:           p.Image,
		Link:            p.Link,
		MiniProgramPage: p.MiniProgramPage,
	}
}

// sendCustomMessageBody is the JSON body of /cgi-bin/message/custom/send.
type sendCustomMessageBody struct {
	ToUser          string                 `json:"touser"`
	MsgType         string                 `json:"msgtype"`
	Text            *CustomText            `json:"text,omitempty"`
	Image           *CustomImage           `json:"image,omitempty"`
	Link            *CustomLink            `json:"link,omitempty"`
	MiniProgramPage *CustomMiniProgramPage `json:"miniprogrampage,omitempty"`
}

// SendCustomMessageResponse is the reply of /cgi-bin/message/custom/send.
//
// It carries no business payload: success is reported by errcode alone.
type SendCustomMessageResponse struct {
	response.WechatResponse
}

// SendCustomMessageErrMsg returns the description of a sendCustomMessage error.
func SendCustomMessageErrMsg(code int) string {
	if msg, ok := SendCustomMessageErrCodeMap[code]; ok {
		return msg
	}
	return SendCustomMessageDefaultErrMsg
}

// SendCustomMessageError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// SendCustomMessageErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func SendCustomMessageError(r *SendCustomMessageResponse) error {
	return errors.New(SendCustomMessageErrMsg(r.ErrCode))
}

// SendCustomMessage replies to a user in a customer service conversation.
//
// The endpoint is not a broadcast channel: WeChat only accepts a message within
// 48 hours of the user's last message, and only five messages fit in that
// window. Outside it the call fails with 45015, which is not retryable; the
// caller has to reach the user another way, such as a subscription message.
//
// Images and card covers must be uploaded first with UploadTempMedia, whose
// media ids live three days.
func (a *Application) SendCustomMessage(
	ctx context.Context,
	params SendCustomMessageParams,
) (*SendCustomMessageResponse, error) {
	if err := CheckSendCustomMessageParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, sendCustomMessagePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result SendCustomMessageResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
