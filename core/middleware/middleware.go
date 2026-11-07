// Package middleware defines the request/response abstractions shared by the
// HTTP client and the middleware chain it composes.
//
// The types live here rather than in core/client so that packages which must
// wrap requests -- auth token injection, logging, retry -- can do so without
// importing the client itself, which would create an import cycle.
package middleware

import (
	"context"
	"net/http"
	"net/url"

	"github.com/antsurge/atto-wechat/core/codec"
	"github.com/antsurge/atto-wechat/core/errors"
)

// BodyType selects how a request body is serialized and which Content-Type is
// sent with it.
type BodyType int

const (
	// BodyJSON sends application/json.
	BodyJSON BodyType = iota
	// BodyForm sends application/x-www-form-urlencoded.
	BodyForm
	// BodyXML sends application/xml.
	BodyXML
	// BodyMultipart sends multipart/form-data; the body must already be encoded.
	BodyMultipart
	// BodyRaw sends the body as-is and leaves Content-Type to the caller.
	BodyRaw
)

// Request is a platform-agnostic HTTP request description. Path is appended to
// the client's BaseURL and Query is URL-encoded onto it.
type Request struct {
	Method   string
	Path     string
	Query    url.Values
	Header   http.Header
	Body     []byte
	BodyType BodyType

	// BaseURL overrides the client's BaseURL for this request. WeChat serves a
	// few resources from other hosts (QR code images come from mp.weixin.qq.com),
	// and a relative Path cannot reach them.
	BaseURL string

	// NoAuth skips credential injection.
	//
	// Public endpoints need no access_token, and injecting one is worse than
	// useless: the token would travel to another host and end up in browser
	// history, referrers and CDN logs.
	NoAuth bool
}

// Response is a buffered HTTP response. The body is read fully into Body so it
// can be decoded repeatedly without touching the wire again.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// OK reports whether the status code is in the 2xx range.
func (r *Response) OK() bool {
	return r != nil && r.StatusCode >= 200 && r.StatusCode < 300
}

// DecodeJSON decodes the body as JSON into v.
func (r *Response) DecodeJSON(v any) error {
	if r == nil {
		return errors.New(errors.CodeInvalidResponse, "nil response")
	}

	if err := codec.Unmarshal(r.Body, v); err != nil {
		return errors.Wrap(errors.CodeDecodeFailed, "decode json response", err)
	}

	return nil
}

// DecodeXML decodes the body as XML into v.
func (r *Response) DecodeXML(v any) error {
	if r == nil {
		return errors.New(errors.CodeInvalidResponse, "nil response")
	}

	if err := codec.XMLUnmarshal(r.Body, v); err != nil {
		return errors.Wrap(errors.CodeDecodeFailed, "decode xml response", err)
	}

	return nil
}

// Handler is the unit of request processing. Middleware wraps a Handler to add
// cross-cutting behaviour.
type Handler func(ctx context.Context, req *Request) (*Response, error)

// Middleware transforms one Handler into another, forming a chain. The first
// middleware of a slice runs outermost: first on the way in, last on the way
// out.
type Middleware func(Handler) Handler
