package codec

import (
	"encoding/xml"
)

// CDATA is a string type that marshals to an XML CDATA section. WeChat requires
// certain reply fields (e.g. message Content) to be wrapped in
// <![CDATA[ ... ]]>, which the standard xml.Marshal does not do automatically.
//
// Declare a field as codec.CDATA and it will serialize as
// <FieldName><![CDATA[value]]></FieldName>. Inbound CDATA content is decoded
// back into the field automatically by encoding/xml.
type CDATA string

// cdataWrapper lets us reuse the standard ",cdata" tag option, which wraps the
// character data of a field in a CDATA section on marshal.
type cdataWrapper struct {
	Value string `xml:",cdata"`
}

// MarshalXML emits the value as a CDATA section under the element name derived
// from the field's xml tag.
func (c CDATA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(cdataWrapper{Value: string(c)}, start)
}

// XMLMarshal serializes v to XML. The root element and field casing are defined
// by the caller's struct tags (WeChat uses a <xml> root with PascalCase
// fields such as ToUserName). No XML declaration is emitted.
func XMLMarshal(v any) ([]byte, error) {
	return xml.Marshal(v)
}

// XMLUnmarshal deserializes XML data into v. Inbound CDATA sections are decoded
// into string (or CDATA) fields transparently.
func XMLUnmarshal(data []byte, v any) error {
	return xml.Unmarshal(data, v)
}

// XMLMarshalIndent serializes v to indented XML, useful for debugging and logs.
func XMLMarshalIndent(v any) ([]byte, error) {
	return xml.MarshalIndent(v, "", "  ")
}
