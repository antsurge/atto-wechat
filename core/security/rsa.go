package security

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/antsurge/atto-wechat/core/errors"
)

// ParseRSAPrivateKeyPEM parses a PEM-encoded RSA private key in PKCS#1
// ("RSA PRIVATE KEY") or PKCS#8 ("PRIVATE KEY") form.
//
// Both forms are accepted because WeChat Pay's console has handed out each at
// different times, and a caller cannot tell which one it downloaded.
func ParseRSAPrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "invalid pem block")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "parse rsa private key", err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New(errors.CodeInvalidArgument, "pem does not hold an rsa private key")
	}

	return key, nil
}

// ParseRSAPublicKeyPEM parses a PEM-encoded RSA public key in PKIX
// ("PUBLIC KEY") or PKCS#1 ("RSA PUBLIC KEY") form.
func ParseRSAPublicKeyPEM(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "invalid pem block")
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		key, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New(errors.CodeInvalidArgument, "pem does not hold an rsa public key")
		}

		return key, nil
	}

	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "parse rsa public key", err)
	}

	return key, nil
}

// SignSHA256 signs message with RSASSA-PKCS1-v1_5 over SHA-256 and returns the
// signature base64-encoded, which is what Pay v3 expects in Authorization.
func SignSHA256(key *rsa.PrivateKey, message []byte) (string, error) {
	if key == nil {
		return "", errors.New(errors.CodeInvalidArgument, "rsa private key must not be nil")
	}

	digest := sha256.Sum256(message)

	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", errors.Wrap(errors.CodeCryptoFailed, "sign with rsa", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifySHA256 verifies a base64 signature produced by SignSHA256.
func VerifySHA256(key *rsa.PublicKey, message []byte, signature string) error {
	if key == nil {
		return errors.New(errors.CodeInvalidArgument, "rsa public key must not be nil")
	}

	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.Wrap(errors.CodeCryptoFailed, "decode signature", err)
	}

	digest := sha256.Sum256(message)

	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return errors.Wrap(errors.CodeCryptoFailed, "verify rsa signature", err)
	}

	return nil
}

// DecryptOAEP decrypts a base64 RSA-OAEP ciphertext, the scheme WeChat Pay v3
// uses for platform certificates and sensitive fields.
//
// The OAEP digest is SHA-1 because that is what the platform specifies. SHA-1
// here only feeds the mask-generation function; the security of OAEP does not
// rest on its collision resistance.
func DecryptOAEP(key *rsa.PrivateKey, ciphertext string) ([]byte, error) {
	if key == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "rsa private key must not be nil")
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, errors.Wrap(errors.CodeCryptoFailed, "decode ciphertext", err)
	}

	plain, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, key, raw, nil)
	if err != nil {
		return nil, errors.Wrap(errors.CodeCryptoFailed, "decrypt rsa oaep", err)
	}

	return plain, nil
}

// EncryptOAEP encrypts plaintext with RSA-OAEP (SHA-1) and returns base64.
//
// It exists mainly so tests and tooling can round-trip DecryptOAEP; production
// flows only ever decrypt.
func EncryptOAEP(key *rsa.PublicKey, plaintext []byte) (string, error) {
	if key == nil {
		return "", errors.New(errors.CodeInvalidArgument, "rsa public key must not be nil")
	}

	enc, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, key, plaintext, nil)
	if err != nil {
		return "", errors.Wrap(errors.CodeCryptoFailed, "encrypt rsa oaep", err)
	}

	return base64.StdEncoding.EncodeToString(enc), nil
}
