package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"

	"github.com/antsurge/atto-wechat/core/errors"
)

// EncryptCBC encrypts plaintext with AES-CBC and PKCS#7 padding.
//
// key must be 16, 24 or 32 bytes; iv must be 16 bytes and should be random per
// message. A fixed IV is not a minor weakness: identical plaintexts then produce
// identical ciphertexts, which leaks structure.
func EncryptCBC(plaintext, key, iv []byte) ([]byte, error) {
	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	if len(iv) != blockSize {
		return nil, errors.New(errors.CodeInvalidArgument, "iv must be 16 bytes")
	}

	padded := PKCS7Pad(plaintext)
	out := make([]byte, len(padded))

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)

	return out, nil
}

// DecryptCBC decrypts AES-CBC ciphertext and strips PKCS#7 padding.
func DecryptCBC(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	if len(iv) != blockSize {
		return nil, errors.New(errors.CodeInvalidArgument, "iv must be 16 bytes")
	}

	if len(ciphertext) == 0 || len(ciphertext)%blockSize != 0 {
		return nil, errors.New(
			errors.CodeCryptoFailed,
			"ciphertext length must be a multiple of 16",
		)
	}

	out := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ciphertext)

	return PKCS7Unpad(out)
}

// EncryptGCM encrypts plaintext with AES-GCM, returning ciphertext and tag
// separately, which is the shape WeChat Pay v3 uses.
func EncryptGCM(plaintext, key, nonce, additionalData []byte) (ciphertext, tag []byte, err error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}

	sealed := gcm.Seal(nil, nonce, plaintext, additionalData)

	// Seal appends the tag to the ciphertext; Pay v3 keeps them apart.
	split := len(sealed) - gcm.Overhead()

	return sealed[:split], sealed[split:], nil
}

// DecryptGCM decrypts AES-GCM ciphertext with a separate tag.
//
// GCM is authenticated: a tampered payload or a wrong additionalData makes this
// return an error rather than garbage. Callers must therefore treat any error as
// "reject the notification", never as "try to parse what came out".
func DecryptGCM(ciphertext, tag, key, nonce, additionalData []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	if len(tag) != gcm.Overhead() {
		return nil, errors.New(errors.CodeCryptoFailed, "invalid gcm tag length")
	}

	sealed := make([]byte, 0, len(ciphertext)+len(tag))
	sealed = append(sealed, ciphertext...)
	sealed = append(sealed, tag...)

	plain, err := gcm.Open(nil, nonce, sealed, additionalData)
	if err != nil {
		return nil, errors.Wrap(errors.CodeCryptoFailed, "decrypt gcm payload", err)
	}

	return plain, nil
}

// newBlock validates the key length and builds an AES cipher.
func newBlock(key []byte) (cipher.Block, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, errors.New(
			errors.CodeInvalidArgument,
			"aes key must be 16, 24 or 32 bytes",
		)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(errors.CodeCryptoFailed, "create aes cipher", err)
	}

	return block, nil
}

// newGCM builds an AES-GCM AEAD for key.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := newBlock(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(errors.CodeCryptoFailed, "create gcm", err)
	}

	return aead, nil
}

// ConstantTimeEqual compares two byte slices without leaking where they differ.
//
// Signature comparison must not use bytes.Equal: it returns early on the first
// mismatch, and the timing difference is enough to forge a signature byte by
// byte.
func ConstantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
