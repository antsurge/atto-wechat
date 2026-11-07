package security

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// SHA1Signature builds WeChat's callback signature: the parts are sorted
// lexicographically, concatenated, then hashed with SHA-1 and hex-encoded.
//
// The sort is part of the algorithm, not a convenience: WeChat computes the same
// value from the same unordered set, so any other order never matches.
func SHA1Signature(parts ...string) string {
	sorted := make([]string, len(parts))
	copy(sorted, parts)
	sort.Strings(sorted)

	sum := sha1.Sum([]byte(strings.Join(sorted, "")))

	return hex.EncodeToString(sum[:])
}

// SHA256Hex returns the lowercase hex SHA-256 of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// HMACSHA256Hex returns the lowercase hex HMAC-SHA256 of message under key.
func HMACSHA256Hex(message, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)

	return hex.EncodeToString(mac.Sum(nil))
}

// MD5Sign builds a WeChat Pay v2 signature: parameters with a non-empty value
// are sorted by key, joined as k=v&..., the API key is appended as key=..., and
// the result is MD5-hashed and upper-cased.
//
// Empty values are skipped because WeChat's own signer skips them; including one
// would produce a signature the server never computes.
func MD5Sign(params map[string]string, apiKey string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
		b.WriteByte('&')
	}
	b.WriteString("key=")
	b.WriteString(apiKey)

	sum := md5.Sum([]byte(b.String()))

	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// VerifyMD5Sign recomputes the Pay v2 signature and compares it in constant time.
func VerifyMD5Sign(params map[string]string, apiKey, sign string) bool {
	return ConstantTimeEqual(
		[]byte(MD5Sign(params, apiKey)),
		[]byte(sign),
	)
}

// PayV3Signature builds the WeChat Pay v3 request signature string, which is the
// message an RSA-SHA256 signature is computed over:
//
//	HTTP_METHOD\nURL\nTIMESTAMP\nNONCE\nBODY\n
//
// The trailing newline is required. Leaving it off produces a signature that
// looks right and is always rejected.
func PayV3Signature(method, urlPath, timestamp, nonce, body string) string {
	return method + "\n" + urlPath + "\n" + timestamp + "\n" + nonce + "\n" + body + "\n"
}
