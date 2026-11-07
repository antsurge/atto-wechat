// Package server implements the HTTP endpoint WeChat calls for callback
// messages: URL verification, signature checking, safe-mode decryption and
// dispatch.
//
// It only depends on the standard library plus core/security and core/event, so
// it can be mounted on any mux (net/http, gin, echo, ...) by registering the
// Server as an http.Handler.
package server

import (
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/antsurge/atto-wechat/core/event"
	"github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/security"
)

// DefaultMaxBodyBytes caps the callback body size. WeChat messages are a few
// kilobytes; the cap keeps a hostile client from allocating unbounded memory.
const DefaultMaxBodyBytes = 1 << 20

// Config holds the endpoint credentials.
//
// At least one of Token and AESKey is required: with neither, anybody who can
// reach the URL can inject messages, so New refuses to build a server that
// cannot tell WeChat from an attacker.
type Config struct {
	// Token is the callback token from the console, used for signatures.
	Token string
	// AESKey is the 43-character EncodingAESKey. Empty means plaintext mode.
	AESKey string
	// AppID is checked against the decrypted message. Empty skips the check.
	AppID string
	// Dispatcher receives parsed messages. A nil dispatcher builds an empty one,
	// which discards messages after deduplication.
	Dispatcher *event.Dispatcher
	// MaxBodyBytes overrides DefaultMaxBodyBytes.
	MaxBodyBytes int64
	// Logger overrides slog.Default.
	Logger *slog.Logger
}

// Server is an http.Handler for the callback endpoint.
type Server struct {
	token      string
	encryptor  *security.MessageEncryptor
	dispatcher *event.Dispatcher
	maxBody    int64
	logger     *slog.Logger
}

// New builds a callback server from cfg.
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "server: config must not be nil")
	}

	if cfg.Token == "" && cfg.AESKey == "" {
		return nil, errors.New(
			errors.CodeInvalidArgument,
			"server: Token or AESKey is required",
		)
	}

	s := &Server{
		token:      cfg.Token,
		dispatcher: cfg.Dispatcher,
		maxBody:    cfg.MaxBodyBytes,
		logger:     cfg.Logger,
	}

	if s.maxBody <= 0 {
		s.maxBody = DefaultMaxBodyBytes
	}
	if s.dispatcher == nil {
		s.dispatcher = event.NewDispatcher()
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}

	if cfg.AESKey != "" {
		enc, err := security.NewMessageEncryptor(cfg.Token, cfg.AESKey, cfg.AppID)
		if err != nil {
			return nil, err
		}
		s.encryptor = enc
	}

	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.verifyURL(w, r)
	case http.MethodPost:
		s.handleMessage(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// verifyURL answers WeChat's one-time URL verification.
//
// Even in safe mode this check uses the plain signature and echoes echostr
// verbatim: encryption applies to message bodies, not to this handshake.
func (s *Server) verifyURL(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if s.token == "" || !s.verifyPlainSignature(q.Get("signature"), q.Get("timestamp"), q.Get("nonce")) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	_, _ = io.WriteString(w, q.Get("echostr"))
}

// handleMessage accepts a callback message and dispatches it.
func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.maxBody))
	if err != nil {
		s.logger.WarnContext(r.Context(), "atto-wechat callback body read failed", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	plain, err := s.decode(r, body)
	if err != nil {
		s.logger.WarnContext(r.Context(), "atto-wechat callback rejected", "error", err)
		http.Error(w, "invalid callback", http.StatusForbidden)

		return
	}

	msg, err := event.ParseMessage(plain)
	if err != nil {
		s.logger.WarnContext(r.Context(), "atto-wechat callback parse failed", "error", err)
		http.Error(w, "invalid callback", http.StatusBadRequest)

		return
	}

	// A dispatch failure -- duplicate or handler error -- is logged by the
	// dispatcher but still answered with success: WeChat retries on failure, and
	// retrying a message the handler already saw helps nobody.
	if err := s.dispatcher.Dispatch(r.Context(), msg); err != nil {
		s.logger.DebugContext(r.Context(), "atto-wechat dispatch concluded with error", "error", err)
	}

	s.logger.DebugContext(r.Context(), "atto-wechat callback handled",
		"msg_type", msg.MsgType,
		"event", msg.Event,
		"elapsed", time.Since(start).String(),
	)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "success")
}

// decode verifies the request and returns the plaintext message XML.
func (s *Server) decode(r *http.Request, body []byte) ([]byte, error) {
	q := r.URL.Query()
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")

	if s.encryptor != nil {
		var wrapper envelope
		if err := xml.Unmarshal(body, &wrapper); err != nil {
			return nil, errors.Wrap(errors.CodeDecodeFailed, "decode encrypted envelope", err)
		}
		if wrapper.Encrypt == "" {
			return nil, errors.New(errors.CodeInvalidResponse, "callback envelope has no Encrypt field")
		}

		return s.encryptor.Decrypt(
			wrapper.Encrypt,
			q.Get("msg_signature"),
			timestamp,
			nonce,
		)
	}

	if !s.verifyPlainSignature(q.Get("signature"), timestamp, nonce) {
		return nil, errors.New(errors.CodeCryptoFailed, "callback signature mismatch")
	}

	return body, nil
}

// verifyPlainSignature checks a plaintext-mode signature in constant time.
func (s *Server) verifyPlainSignature(signature, timestamp, nonce string) bool {
	if s.token == "" {
		return false
	}

	expected := security.SHA1Signature(s.token, timestamp, nonce)

	return security.ConstantTimeEqual([]byte(expected), []byte(signature))
}

// Dispatcher exposes the dispatcher so handlers can be registered after the
// server is built.
func (s *Server) Dispatcher() *event.Dispatcher {
	return s.dispatcher
}

// envelope is the safe-mode callback wrapper.
type envelope struct {
	ToUserName string `xml:"ToUserName"`
	Encrypt    string `xml:"Encrypt"`
	AgentID    string `xml:"AgentID"`
}
