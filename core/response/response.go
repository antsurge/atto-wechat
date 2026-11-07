// Package response holds the reply shapes returned by WeChat APIs.
//
// A reply that carries an errcode is a normal, successfully transported
// response: the SDK decodes it into one of these types and lets the caller
// decide what the code means for its business. Nothing here is converted into
// an error.
package response

// Response is the minimal reply shape used by endpoints that answer with a
// string code and message instead of WeChat's numeric errcode.
type Response struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// OK reports whether the reply carries no error code.
func (r *Response) OK() bool {
	return r != nil && r.Code == ""
}

// IsError reports whether the reply carries an error code.
func (r *Response) IsError() bool {
	return !r.OK()
}
