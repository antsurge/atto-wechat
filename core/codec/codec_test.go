package codec

import (
	"encoding/xml"
	"strings"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestMarshalDoesNotEscapeHTML(t *testing.T) {
	got, err := MarshalToString(sample{Name: "<a>&b", Value: 1})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(got, `\u003c`) || strings.Contains(got, `\u0026`) {
		t.Fatalf("html was escaped: %s", got)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatalf("trailing newline not trimmed: %q", got)
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	in := sample{Name: "x", Value: 42}

	raw, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out sample
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round trip = %+v, want %+v", out, in)
	}
}

func TestMarshalIndent(t *testing.T) {
	got, err := MarshalIndent(sample{Name: "x", Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\n  ") {
		t.Fatalf("not indented: %s", got)
	}
}

func TestUnmarshalStringAndToMap(t *testing.T) {
	var s sample
	if err := UnmarshalString(`{"name":"a","value":2}`, &s); err != nil {
		t.Fatal(err)
	}
	if s.Name != "a" || s.Value != 2 {
		t.Fatalf("decoded %+v", s)
	}

	m, err := ToMap([]byte(`{"a":1,"b":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if m["b"] != "x" {
		t.Fatalf("map = %v", m)
	}
}

func TestUnmarshalRejectsMalformed(t *testing.T) {
	var s sample
	if err := Unmarshal([]byte(`{"name":`), &s); err == nil {
		t.Fatal("malformed json decoded without error")
	}
}

type reply struct {
	XMLName xml.Name `xml:"xml"`
	ToUser  string   `xml:"ToUserName"`
	Content CDATA    `xml:"Content"`
}

func TestXMLMarshalCDATA(t *testing.T) {
	raw, err := XMLMarshal(reply{ToUser: "u", Content: "a<b"})
	if err != nil {
		t.Fatal(err)
	}

	got := string(raw)
	if !strings.Contains(got, "<![CDATA[a<b]]>") {
		t.Fatalf("cdata missing: %s", got)
	}
	if strings.Contains(got, "<?xml") {
		t.Fatalf("unexpected xml declaration: %s", got)
	}
}

func TestXMLRoundTrip(t *testing.T) {
	in := reply{ToUser: "u", Content: "hello"}

	raw, err := XMLMarshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out reply
	if err := XMLUnmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.ToUser != "u" || out.Content != "hello" {
		t.Fatalf("round trip = %+v", out)
	}
}

func TestXMLUnmarshalMalformed(t *testing.T) {
	var out reply
	if err := XMLUnmarshal([]byte(`<xml><ToUserName>`), &out); err == nil {
		t.Fatal("malformed xml decoded without error")
	}
}
