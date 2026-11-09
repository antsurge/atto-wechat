package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// event name of the 差评 callback.
const CommentEventBadScore = "wxa_comment_bad_score"

// CommentBadScoreEvent is the wxa_comment_bad_score callback: a user left a bad
// review.
//
// The point of the push is the follow-up: after receiving it the merchant can reset
// its customer-service quota and then message the reviewer from the service account,
// which is otherwise not allowed. That is why the reply is only "success" -- the
// action happens through the客服 APIs, not through this reply.
type CommentBadScoreEvent struct {
	EventNotify

	// Result carries the review id.
	Result *commentBadScoreResult `xml:"result" json:"result"`
}

// commentBadScoreResult is the inner element of the bad-review push.
type commentBadScoreResult struct {
	// CommentID is the review the push is about.
	CommentID string `xml:"comment_id" json:"comment_id"`
}

// CommentID returns the review id, or "" when the payload carried none.
func (e *CommentBadScoreEvent) CommentID() string {
	if e == nil || e.Result == nil {
		return ""
	}

	return e.Result.CommentID
}

// OK reports whether the payload identifies the review.
func (e *CommentBadScoreEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.CommentID() != ""
}

// CommentBadScoreHandler handles a wxa_comment_bad_score callback.
type CommentBadScoreHandler func(
	ctx context.Context, evt *CommentBadScoreEvent,
) error

// OnCommentBadScore registers a handler for the wxa_comment_bad_score callback.
func (a *Application) OnCommentBadScore(fn CommentBadScoreHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+CommentEventBadScore,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &CommentBadScoreEvent{}

			if err := decodeDispatched(msg, []string{CommentEventBadScore}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
