package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// MediaCheckEventResult is the event name of the 多媒体内容安全识别 result push.
const MediaCheckEventResult = "wxa_media_check"

// MediaCheckDownloadFailed is the errcode a result push carries when WeChat could
// not download the media.
//
// It is not an errcode of the submission but of the check itself, and it is the
// usual outcome of a URL that is not publicly reachable -- worth distinguishing
// from "the media is clean" when reading a push.
const MediaCheckDownloadFailed = -1008

// MediaCheckResultEvent is the wxa_media_check callback: the verdict of a
// MediaCheckAsync submission.
//
// It is the only place the verdict appears; MediaCheckAsyncResponse carries just
// the trace_id that ties the two together.
type MediaCheckResultEvent struct {
	EventNotify

	// AppID is the mini program the media was submitted for.
	AppID string `xml:"appid" json:"appid"`
	// TraceID is the value MediaCheckAsyncResponse returned.
	TraceID string `xml:"trace_id" json:"trace_id"`
	// Version is the interface version, 2 for the current one.
	Version int `xml:"version" json:"version"`
	// ErrCode is 0 when Verdict is valid, MediaCheckDownloadFailed when WeChat
	// could not download the media.
	ErrCode int `xml:"errcode" json:"errcode"`
	// ErrMsg is WeChat's own description of ErrCode.
	ErrMsg string `xml:"errmsg" json:"errmsg"`
	// Verdict is the overall result.
	Verdict *SecCheckResult `xml:"result" json:"result"`
	// Detail is each strategy's opinion.
	Detail []SecCheckDetail `xml:"detail" json:"detail"`
}

// OK reports whether the push carries a usable verdict.
//
// A download failure is not OK: there is no verdict to act on, and treating it as
// "nothing was found" would let unchecked media through.
func (e *MediaCheckResultEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.ErrCode == 0 && e.Verdict != nil
}

// DownloadFailed reports whether the check could not run because WeChat failed to
// fetch the media.
func (e *MediaCheckResultEvent) DownloadFailed() bool {
	return e != nil && e.ErrCode == MediaCheckDownloadFailed
}

// Passed reports whether the media may be published as it is.
func (e *MediaCheckResultEvent) Passed() bool {
	return e.OK() && e.Verdict.Passed()
}

// Risky reports whether the media must be blocked.
func (e *MediaCheckResultEvent) Risky() bool {
	return e.OK() && e.Verdict.Risky()
}

// Review reports whether the media needs a human decision.
func (e *MediaCheckResultEvent) Review() bool {
	return e.OK() && e.Verdict.Review()
}

// MediaCheckResultHandler handles a wxa_media_check callback.
type MediaCheckResultHandler func(ctx context.Context, evt *MediaCheckResultEvent) error

// OnMediaCheckResult registers a handler for the wxa_media_check callback.
//
// The handler runs for every submission, including the ones whose media could not
// be downloaded: deciding what to do about DownloadFailed is the caller's call,
// and silently dropping those pushes would leave the content in limbo forever.
func (a *Application) OnMediaCheckResult(fn MediaCheckResultHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+MediaCheckEventResult,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &MediaCheckResultEvent{}

			if err := decodeDispatched(msg, []string{MediaCheckEventResult}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
