package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
	"github.com/antsurge/atto-wechat/core/security"
)

const token = "callback-token"

// testAESKey builds a valid 43-character EncodingAESKey.
func testAESKey(t *testing.T) string {
	t.Helper()

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}

	return base64.RawStdEncoding.EncodeToString(raw)
}

// newServer builds a server whose dispatcher records delivered messages.
func newServer(t *testing.T, cfg *Config) (*Server, *recordingHandler) {
	t.Helper()

	rec := &recordingHandler{}

	if cfg.Dispatcher == nil {
		cfg.Dispatcher = event.NewDispatcher().
			On("*", rec.handle)
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return srv, rec
}

type recordingHandler struct {
	mu   sync.Mutex
	msgs []*event.Message
}

func (h *recordingHandler) handle(_ context.Context, msg *event.Message) error {
	h.mu.Lock()
	h.msgs = append(h.msgs, msg)
	h.mu.Unlock()

	return nil
}

func (h *recordingHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.msgs)
}

func (h *recordingHandler) first() *event.Message {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.msgs) == 0 {
		return nil
	}

	return h.msgs[0]
}

const messageXML = `<xml><ToUserName>gh_1</ToUserName><FromUserName>u1</FromUserName><CreateTime>1700000000</CreateTime><MsgType>text</MsgType><Content>hi</Content><MsgId>1</MsgId></xml>`

func TestVerifyURL(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token})

	req := httptest.NewRequest(http.MethodGet, "/callback?"+url.Values{
		"signature": {security.SHA1Signature(token, "123", "n")},
		"timestamp": {"123"},
		"nonce":     {"n"},
		"echostr":   {"hello"},
	}.Encode(), nil)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestVerifyURLRejectsBadSignature(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token})

	req := httptest.NewRequest(http.MethodGet, "/callback?signature=bad&timestamp=1&nonce=n&echostr=x", nil)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestPlaintextMessage(t *testing.T) {
	rec := &recordingHandler{}
	srv, _ := newServer(t, &Config{
		Token:      token,
		Dispatcher: event.NewDispatcher().On("*", rec.handle),
	})

	req := httptest.NewRequest(http.MethodPost, "/callback?"+url.Values{
		"signature": {security.SHA1Signature(token, "123", "n")},
		"timestamp": {"123"},
		"nonce":     {"n"},
	}.Encode(), strings.NewReader(messageXML))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Fatalf("body = %q", w.Body.String())
	}
	if rec.count() != 1 {
		t.Fatalf("delivered %d messages, want 1", rec.count())
	}
	if got := rec.first(); got == nil || got.Content != "hi" {
		t.Fatalf("message = %+v", got)
	}
}

func TestPlaintextMessageWithReply(t *testing.T) {
	// Some callbacks -- the courier ones -- reject "success" and expect their own
	// document back, so a handler that sets a reply has to take over the body.
	srv, _ := newServer(t, &Config{
		Token: token,
		Dispatcher: event.NewDispatcher().On("*", func(_ context.Context, msg *event.Message) error {
			msg.Reply = []byte(`{"ResultCode":0}`)
			msg.ReplyContentType = "application/json; charset=utf-8"

			return nil
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/callback?"+url.Values{
		"signature": {security.SHA1Signature(token, "123", "n")},
		"timestamp": {"123"},
		"nonce":     {"n"},
	}.Encode(), strings.NewReader(messageXML))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Body.String(); got != `{"ResultCode":0}` {
		t.Fatalf("body = %q, want the handler's reply", got)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestPlaintextMessageRejectsBadSignature(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token})

	req := httptest.NewRequest(http.MethodPost, "/callback?signature=bad&timestamp=1&nonce=n",
		strings.NewReader(messageXML))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestSafeModeMessage(t *testing.T) {
	key := testAESKey(t)

	enc, err := security.NewMessageEncryptor(token, key, "wx1")
	if err != nil {
		t.Fatal(err)
	}

	encrypted, signature, err := enc.Encrypt(messageXML, "123", "n")
	if err != nil {
		t.Fatal(err)
	}

	body := `<xml><ToUserName>gh_1</ToUserName><Encrypt>` + encrypted + `</Encrypt></xml>`

	rec := &recordingHandler{}
	srv, _ := newServer(t, &Config{
		Token:      token,
		AESKey:     key,
		AppID:      "wx1",
		Dispatcher: event.NewDispatcher().On("*", rec.handle),
	})

	req := httptest.NewRequest(http.MethodPost, "/callback?"+url.Values{
		"msg_signature": {signature},
		"timestamp":     {"123"},
		"nonce":         {"n"},
	}.Encode(), strings.NewReader(body))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rec.count() != 1 {
		t.Fatalf("delivered %d messages, want 1", rec.count())
	}
	if got := rec.first(); got == nil || got.Content != "hi" {
		t.Fatalf("message = %+v", got)
	}
}

func TestSafeModeRejectsBadSignature(t *testing.T) {
	key := testAESKey(t)

	srv, _ := newServer(t, &Config{Token: token, AESKey: key, AppID: "wx1"})

	body := `<xml><Encrypt>AAAA</Encrypt></xml>`

	req := httptest.NewRequest(http.MethodPost, "/callback?msg_signature=bad&timestamp=1&nonce=n",
		strings.NewReader(body))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestSafeModeRejectsMissingEncrypt(t *testing.T) {
	key := testAESKey(t)

	srv, _ := newServer(t, &Config{Token: token, AESKey: key})

	req := httptest.NewRequest(http.MethodPost, "/callback?msg_signature=x&timestamp=1&nonce=n",
		strings.NewReader(`<xml><ToUserName>gh</ToUserName></xml>`))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestMalformedMessageRejected(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token})

	req := httptest.NewRequest(http.MethodPost, "/callback?"+url.Values{
		"signature": {security.SHA1Signature(token, "123", "n")},
		"timestamp": {"123"},
		"nonce":     {"n"},
	}.Encode(), strings.NewReader(`<xml><MsgType>`))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token})

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/callback", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
	if w.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("allow = %q", w.Header().Get("Allow"))
	}
}

func TestBodyLimit(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token, MaxBodyBytes: 32})

	big := strings.Repeat("x", 64)

	req := httptest.NewRequest(http.MethodPost, "/callback?"+url.Values{
		"signature": {security.SHA1Signature(token, "123", "n")},
		"timestamp": {"123"},
		"nonce":     {"n"},
	}.Encode(), strings.NewReader(big))

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{"nil config", nil},
		{"no credentials", &Config{}},
		{"short aes key", &Config{Token: token, AESKey: "short"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.cfg); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestDefaults(t *testing.T) {
	srv, _ := newServer(t, &Config{Token: token})

	if srv.Dispatcher() == nil {
		t.Fatal("dispatcher = nil")
	}
	if srv.maxBody != DefaultMaxBodyBytes {
		t.Fatalf("maxBody = %d", srv.maxBody)
	}
	if srv.logger == nil {
		t.Fatal("logger = nil")
	}
}
