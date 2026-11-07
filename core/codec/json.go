// Package codec provides JSON and XML encoding/decoding utilities for WeChat
// API interactions.
//
// It is part of Tier 0: it depends only on the standard library. Callers (such
// as the HTTP client) decide how to wrap serialization failures into unified
// errors, keeping this package free of internal dependencies.
package codec

import (
	"bytes"
	"encoding/json"
)

// Marshal serializes v to JSON. Unlike encoding/json.Marshal, it disables HTML
// escaping so that characters like '<', '>', and '&' are written verbatim —
// this matches what WeChat expects for message content and keeps payloads
// human-readable.
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encoder.Encode appends a trailing newline; trim it for clean payloads.
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

// Unmarshal deserializes JSON data into v.
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// MarshalToString is a convenience wrapper returning a string.
func MarshalToString(v any) (string, error) {
	b, err := Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalString is a convenience wrapper over a string payload.
func UnmarshalString(s string, v any) error {
	return Unmarshal([]byte(s), v)
}

// MarshalIndent serializes v to indented JSON, useful for debugging and logs.
func MarshalIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

// ToMap deserializes JSON data into a generic map, convenient for flexible
// event/notification parsing where the shape is not known up front.
func ToMap(data []byte) (map[string]any, error) {
	m := make(map[string]any)
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
