package event

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/errors"
)

// DefaultDedupTTL is how long a processed message is remembered.
//
// WeChat's retry window is seconds, but messages can be redelivered after a
// longer outage; a day of memory costs almost nothing against an in-memory cache.
const DefaultDedupTTL = 24 * time.Hour

// Handler processes one callback message.
//
// Returning an error does not stop the reply to WeChat: the message was
// delivered, and answering with a failure would only make WeChat retry a
// message the handler has already seen. Retries are the caller's decision.
type Handler func(ctx context.Context, msg *Message) error

// Dispatcher routes messages to registered handlers.
//
// Routes are looked up in order of decreasing specificity:
//
//	"event.subscribe"  exact event
//	"event"            all events
//	"text"             all text messages
//	"*"                anything
//
// The first match wins, so registering a broad route and a specific one gives
// the specific one priority.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler

	dedup    cache.SetNXer
	dedupTTL time.Duration

	onError func(ctx context.Context, msg *Message, err error)
	logger  *slog.Logger
}

// DispatcherOption mutates a Dispatcher.
type DispatcherOption func(*Dispatcher)

// WithDedup enables deduplication using an atomic SetNX store.
//
// Without a store, dispatch is at-most-once per handler invocation only: a
// WeChat retry will run the handler again. Pass a cache.SetNXer (MemoryCache
// satisfies it) when handlers are not idempotent.
func WithDedup(store cache.SetNXer, ttl time.Duration) DispatcherOption {
	return func(d *Dispatcher) {
		d.dedup = store
		if ttl > 0 {
			d.dedupTTL = ttl
		}
	}
}

// WithErrorHandler installs a callback invoked when a handler fails. It runs
// after the error is logged, and its own panic-free execution is the caller's
// responsibility.
func WithErrorHandler(fn func(ctx context.Context, msg *Message, err error)) DispatcherOption {
	return func(d *Dispatcher) {
		d.onError = fn
	}
}

// WithLogger overrides the logger used for handler failures.
func WithLogger(l *slog.Logger) DispatcherOption {
	return func(d *Dispatcher) {
		if l != nil {
			d.logger = l
		}
	}
}

// NewDispatcher creates an empty dispatcher.
func NewDispatcher(opts ...DispatcherOption) *Dispatcher {
	d := &Dispatcher{
		handlers: make(map[string]Handler),
		dedupTTL: DefaultDedupTTL,
		logger:   slog.Default(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	return d
}

// On registers a handler for key and returns the dispatcher, so registrations
// can be chained.
//
// Registering the same key twice replaces the previous handler: a builder that
// runs twice should end up with one handler, not two.
func (d *Dispatcher) On(key string, h Handler) *Dispatcher {
	if key == "" || h == nil {
		return d
	}

	d.mu.Lock()
	d.handlers[key] = h
	d.mu.Unlock()

	return d
}

// Dispatch delivers msg to the handler selected for it.
//
// It reports an error when the message was already processed, so the caller can
// tell a duplicate from a fresh delivery. The error is informational: WeChat
// still expects a success reply.
func (d *Dispatcher) Dispatch(ctx context.Context, msg *Message) error {
	if msg == nil {
		return errors.New(errors.CodeInvalidArgument, "message must not be nil")
	}

	key := routeKey(msg)

	if dup, err := d.claim(msg); err != nil {
		// A broken dedup store must not drop messages: dispatch anyway and let
		// the handler, which knows whether it is idempotent, decide.
		d.logger.WarnContext(ctx, "atto-wechat dedup store failed",
			"error", err, "route", key)
	} else if dup {
		d.logger.DebugContext(ctx, "atto-wechat duplicate message ignored",
			"route", key, "identity", msg.Identity())

		return errors.New(errors.CodeInvalidArgument, "duplicate callback message")
	}

	h := d.lookup(msg)
	if h == nil {
		d.logger.DebugContext(ctx, "atto-wechat no handler for message",
			"route", key)

		return nil
	}

	if err := h(ctx, msg); err != nil {
		d.logger.ErrorContext(ctx, "atto-wechat handler failed",
			"route", key, "error", err)

		if d.onError != nil {
			d.onError(ctx, msg, err)
		}

		return err
	}

	return nil
}

// lookup returns the most specific handler for msg.
func (d *Dispatcher) lookup(msg *Message) Handler {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if msg.IsEvent() && msg.Event != "" {
		if h, ok := d.handlers[msg.MsgType+"."+msg.Event]; ok {
			return h
		}
	}

	if h, ok := d.handlers[msg.MsgType]; ok {
		return h
	}

	return d.handlers["*"]
}

// claim atomically marks msg as processed.
//
// SetNX is used rather than Get-then-Set because two concurrent deliveries of
// the same message would both observe a miss and both proceed, which is exactly
// the retry case deduplication exists to catch.
func (d *Dispatcher) claim(msg *Message) (bool, error) {
	if d.dedup == nil {
		return false, nil
	}

	key := "atto-wechat:callback:dedup:" + msg.Identity()

	stored, err := d.dedup.SetNX(key, []byte("1"), d.dedupTTL)
	if err != nil {
		return false, err
	}

	return !stored, nil
}

// routeKey renders the route a message is dispatched under, for logging.
func routeKey(msg *Message) string {
	if msg.IsEvent() && msg.Event != "" {
		return msg.MsgType + "." + msg.Event
	}

	return msg.MsgType
}
