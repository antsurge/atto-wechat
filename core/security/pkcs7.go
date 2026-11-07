// Package security implements the cryptography WeChat requires: AES-CBC for
// callback message encryption, AES-GCM for Pay v3 resources, PKCS#7 padding,
// SHA-1 request signatures, MD5/HMAC-SHA256 Pay v2 signatures and RSA-SHA256.
//
// It is part of Tier 0: only the standard library is used. Nothing here talks to
// the network, so every function is directly testable with fixed vectors.
package security

import (
	"github.com/antsurge/atto-wechat/core/errors"
)

// blockSize is the AES block size, and also the PKCS#7 block size WeChat uses.
const blockSize = 16

// PKCS7Pad appends padding so that len(data) is a multiple of blockSize.
//
// When the input is already aligned a full block of padding is added, as the
// standard requires. Omitting it would make Unpad ambiguous: it could not tell
// data from padding.
func PKCS7Pad(data []byte) []byte {
	pad := blockSize - len(data)%blockSize

	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}

	return out
}

// PKCS7Unpad removes and validates PKCS#7 padding.
//
// The padding bytes are checked, not just the length byte: accepting any length
// would let a corrupted or truncated payload decode into wrong data instead of
// failing.
func PKCS7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New(
			errors.CodeCryptoFailed,
			"invalid padded length",
		)
	}

	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, errors.New(
			errors.CodeCryptoFailed,
			"invalid padding size",
		)
	}

	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, errors.New(
				errors.CodeCryptoFailed,
				"invalid padding bytes",
			)
		}
	}

	return data[:len(data)-pad], nil
}
