package client

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/middleware"
)

// FilePart is one file in a multipart upload.
type FilePart struct {
	// FieldName is the form field name WeChat expects (media, voice, ...).
	FieldName string
	// FileName is the name reported to WeChat; the extension matters, because
	// WeChat validates the media type by suffix.
	FileName string
	// ContentType defaults to application/octet-stream.
	ContentType string
	// Content is the file body. It is held in memory because the client must be
	// able to replay the request on retry: a streamed reader would already be
	// consumed by the first attempt.
	Content []byte
}

// PostMultipart uploads fields and files as multipart/form-data.
func (c *Client) PostMultipart(
	ctx context.Context,
	path string,
	query url.Values,
	fields url.Values,
	files ...*FilePart,
) (*Response, error) {
	body, contentType, err := encodeMultipart(fields, files)
	if err != nil {
		return nil, err
	}

	return c.Do(ctx, &Request{
		Method:   http.MethodPost,
		Path:     path,
		Query:    query,
		Body:     body,
		BodyType: middleware.BodyMultipart,
		Header:   http.Header{"Content-Type": {contentType}},
	})
}

// encodeMultipart builds the multipart body in memory.
func encodeMultipart(fields url.Values, files []*FilePart) ([]byte, string, error) {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)

	for k, vs := range fields {
		for _, v := range vs {
			if err := w.WriteField(k, v); err != nil {
				return nil, "", errors.Wrap(errors.CodeEncodeFailed, "write multipart field", err)
			}
		}
	}

	for _, f := range files {
		if f == nil {
			continue
		}
		if f.FieldName == "" || f.FileName == "" {
			return nil, "", errors.New(
				errors.CodeInvalidArgument,
				"multipart file requires FieldName and FileName",
			)
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="`+escapeQuotes(f.FieldName)+
			`"; filename="`+escapeQuotes(f.FileName)+`"`)

		ct := f.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)

		part, err := w.CreatePart(h)
		if err != nil {
			return nil, "", errors.Wrap(errors.CodeEncodeFailed, "create multipart part", err)
		}
		if _, err := part.Write(f.Content); err != nil {
			return nil, "", errors.Wrap(errors.CodeEncodeFailed, "write multipart part", err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", errors.Wrap(errors.CodeEncodeFailed, "close multipart writer", err)
	}

	return buf.Bytes(), w.FormDataContentType(), nil
}

// quoteEscaper mirrors the escaping mime/multipart applies to header parameters.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// escapeQuotes escapes a header parameter value.
func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
