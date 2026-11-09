package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// event name of the 短剧媒资 upload callback.
const MiniDramaEventAssetUpload = "secvod_upload_event"

// MiniDramaAssetUpload is the upload outcome.
type MiniDramaAssetUpload struct {
	// MediaID is the id the media was given.
	MediaID int64 `xml:"media_id" json:"media_id"`
	// SourceContext is the value the upload call passed in, echoed back so the
	// caller can match the push to its own record.
	SourceContext string `xml:"source_context" json:"source_context"`
	// ErrCode is non-zero when the upload failed.
	ErrCode int `xml:"errcode" json:"errcode"`
	// ErrMsg explains a failure.
	ErrMsg string `xml:"errmsg" json:"errmsg"`
}

// Succeeded reports whether the media made it in.
func (u MiniDramaAssetUpload) Succeeded() bool {
	return u.ErrCode == 0
}

// MiniDramaAssetUploadEvent is the secvod_upload_event callback: a media upload to
// 短剧媒资 finished.
//
// It is a plain push -- "success" is the only expected answer -- so the handler
// returns an error rather than a reply document.
type MiniDramaAssetUploadEvent struct {
	EventNotify

	// UploadEvent carries the outcome, as the documented XML nests it.
	UploadEvent *MiniDramaAssetUpload `xml:"upload_event" json:"upload_event"`
}

// Upload returns the outcome, or a zero value when the payload carried none.
func (e *MiniDramaAssetUploadEvent) Upload() MiniDramaAssetUpload {
	if e == nil || e.UploadEvent == nil {
		return MiniDramaAssetUpload{}
	}

	return *e.UploadEvent
}

// OK reports whether the payload carries something to act on.
//
// A failed upload is a complete, valid payload: the media id is why the merchant
// needs the push at all.
func (e *MiniDramaAssetUploadEvent) OK() bool {
	if e == nil || !e.EventNotify.OK() {
		return false
	}

	upload := e.Upload()

	return upload.MediaID != 0
}

// MiniDramaAssetUploadHandler handles a secvod_upload_event callback.
type MiniDramaAssetUploadHandler func(
	ctx context.Context, evt *MiniDramaAssetUploadEvent,
) error

// OnMiniDramaAssetUpload registers a handler for the secvod_upload_event callback.
func (a *Application) OnMiniDramaAssetUpload(fn MiniDramaAssetUploadHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+MiniDramaEventAssetUpload,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &MiniDramaAssetUploadEvent{}

			if err := decodeDispatched(msg, []string{MiniDramaEventAssetUpload}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
