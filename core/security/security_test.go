package security

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/antsurge/atto-wechat/core/errors"
)

func mustRandom(t *testing.T, n int) []byte {
	t.Helper()

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	return b
}

func TestPKCS7RoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 15, 16, 17, 31, 32} {
		in := mustRandom(t, size)

		padded := PKCS7Pad(in)
		if len(padded)%blockSize != 0 {
			t.Fatalf("size %d: padded length %d is not a multiple of %d", size, len(padded), blockSize)
		}

		out, err := PKCS7Unpad(padded)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if string(out) != string(in) {
			t.Fatalf("size %d: round trip mismatch", size)
		}
	}
}

func TestPKCS7UnpadRejectsInvalidPadding(t *testing.T) {
	valid := PKCS7Pad([]byte("hello"))
	if valid[len(valid)-1] != 11 {
		t.Fatalf("padding byte = %d, want 11", valid[len(valid)-1])
	}

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"not a block multiple", []byte{1, 2, 3}},
		{"zero padding", append(valid, 0x00)},
		{"padding larger than block", append(mustRandom(t, 15), 20)},
		{"inconsistent padding", func() []byte {
			b := append([]byte(nil), valid...)
			b[len(b)-1] = 3
			b[len(b)-2] = 2

			return b
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := PKCS7Unpad(tt.data); !errors.Is(err, errors.CodeCryptoFailed) {
				t.Fatalf("err = %v, want CRYPTO_FAILED", err)
			}
		})
	}
}

func TestCBCRoundTrip(t *testing.T) {
	for _, keySize := range []int{16, 24, 32} {
		key := mustRandom(t, keySize)
		iv := mustRandom(t, blockSize)
		plain := []byte("wechat callback payload")

		sealed, err := EncryptCBC(plain, key, iv)
		if err != nil {
			t.Fatalf("key %d: %v", keySize, err)
		}

		out, err := DecryptCBC(sealed, key, iv)
		if err != nil {
			t.Fatalf("key %d: %v", keySize, err)
		}
		if string(out) != string(plain) {
			t.Fatalf("key %d: round trip mismatch", keySize)
		}
	}
}

func TestCBCRejectsBadParameters(t *testing.T) {
	valid := []byte("0123456789abcdef")
	iv := []byte("0123456789abcdef")

	if _, err := EncryptCBC([]byte("x"), []byte("short"), iv); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("short key err = %v", err)
	}
	if _, err := EncryptCBC([]byte("x"), valid, []byte("short")); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("short iv err = %v", err)
	}
	if _, err := DecryptCBC([]byte("not-a-block"), valid, iv); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("bad ciphertext err = %v", err)
	}
	if _, err := DecryptCBC(nil, valid, iv); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("empty ciphertext err = %v", err)
	}
}

func TestGCMRoundTrip(t *testing.T) {
	key := mustRandom(t, 32)
	nonce := mustRandom(t, 12)
	aad := []byte("transaction")
	plain := []byte(`{"out_trade_no":"123"}`)

	ciphertext, tag, err := EncryptGCM(plain, key, nonce, aad)
	if err != nil {
		t.Fatal(err)
	}

	out, err := DecryptGCM(ciphertext, tag, key, nonce, aad)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("round trip = %q", out)
	}
}

func TestGCMRejectsTampering(t *testing.T) {
	key := mustRandom(t, 32)
	nonce := mustRandom(t, 12)
	aad := []byte("transaction")

	ciphertext, tag, err := EncryptGCM([]byte("secret"), key, nonce, aad)
	if err != nil {
		t.Fatal(err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("empty ciphertext")
	}

	bad := append([]byte(nil), ciphertext...)
	bad[0] ^= 0xff

	if _, err := DecryptGCM(bad, tag, key, nonce, aad); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("tampered ciphertext err = %v", err)
	}
	if _, err := DecryptGCM(ciphertext, tag, key, nonce, []byte("other")); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("wrong aad err = %v", err)
	}
	if _, err := DecryptGCM(ciphertext, tag[:8], key, nonce, aad); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("short tag err = %v", err)
	}
}

func TestSHA1Signature(t *testing.T) {
	// Sorting is part of the algorithm: the parts are given out of order on
	// purpose, and the expected digest is of their sorted concatenation.
	expected := sha1.Sum([]byte("abc"))

	got := SHA1Signature("c", "a", "b")
	if got != hex.EncodeToString(expected[:]) {
		t.Fatalf("signature = %s", got)
	}

	if SHA1Signature("b", "a", "c") != got {
		t.Fatal("signature depends on argument order")
	}
}

func TestMD5Sign(t *testing.T) {
	params := map[string]string{
		"b":      "2",
		"a":      "1",
		"empty":  "",
		"sign":   "should-be-ignored",
		"ignor2": "ignored",
	}

	// The empty value and the sign field are skipped exactly as WeChat's own
	// signer does; every other non-empty parameter still contributes.
	sum := md5.Sum([]byte("a=1&b=2&ignor2=ignored&key=k"))

	want := strings.ToUpper(hex.EncodeToString(sum[:]))
	if MD5Sign(params, "k") != want {
		t.Fatalf("signature = %s, want %s", MD5Sign(params, "k"), want)
	}

	if !VerifyMD5Sign(params, "k", want) {
		t.Fatal("VerifyMD5Sign rejected a valid signature")
	}
	if VerifyMD5Sign(params, "k", "00") {
		t.Fatal("VerifyMD5Sign accepted an invalid signature")
	}
}

func TestHMACSHA256Hex(t *testing.T) {
	got := HMACSHA256Hex([]byte("message"), []byte("secret"))

	if len(got) != 64 {
		t.Fatalf("digest length = %d", len(got))
	}
	if got != HMACSHA256Hex([]byte("message"), []byte("secret")) {
		t.Fatal("HMAC is not deterministic")
	}
	if got == HMACSHA256Hex([]byte("message"), []byte("other")) {
		t.Fatal("HMAC does not depend on the key")
	}
}

func TestPayV3Signature(t *testing.T) {
	got := PayV3Signature("POST", "/v3/pay/transactions/jsapi", "123", "nonce", `{"a":1}`)

	want := "POST\n/v3/pay/transactions/jsapi\n123\nnonce\n{\"a\":1}\n"
	if got != want {
		t.Fatalf("signature string = %q, want %q", got, want)
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual([]byte("abc"), []byte("abc")) {
		t.Fatal("equal slices reported unequal")
	}
	if ConstantTimeEqual([]byte("abc"), []byte("abd")) {
		t.Fatal("unequal slices reported equal")
	}
	if ConstantTimeEqual([]byte("abc"), []byte("ab")) {
		t.Fatal("different lengths reported equal")
	}
}

// testAESKey returns a 43-character EncodingAESKey.
func testAESKey(t *testing.T) string {
	t.Helper()

	raw := mustRandom(t, 32)

	key := base64.RawStdEncoding.EncodeToString(raw)
	if len(key) != encodingAESKeyLength {
		t.Fatalf("test key length = %d", len(key))
	}

	return key
}

func TestMessageEncryptorRoundTrip(t *testing.T) {
	key := testAESKey(t)

	enc, err := NewMessageEncryptor("token", key, "wx123")
	if err != nil {
		t.Fatal(err)
	}

	plain := []byte(`<xml><MsgType>text</MsgType><Content>hi</Content></xml>`)

	encrypted, signature, err := enc.Encrypt(string(plain), "1700000000", "nonce")
	if err != nil {
		t.Fatal(err)
	}

	out, err := enc.Decrypt(encrypted, signature, "1700000000", "nonce")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("decrypted = %s", out)
	}
}

func TestMessageEncryptorRejectsBadSignature(t *testing.T) {
	enc, err := NewMessageEncryptor("token", testAESKey(t), "")
	if err != nil {
		t.Fatal(err)
	}

	encrypted, _, err := enc.Encrypt("x", "1", "n")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := enc.Decrypt(encrypted, "deadbeef", "1", "n"); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestMessageEncryptorChecksAppID(t *testing.T) {
	key := testAESKey(t)

	sender, err := NewMessageEncryptor("token", key, "other-app")
	if err != nil {
		t.Fatal(err)
	}

	encrypted, signature, err := sender.Encrypt("x", "1", "n")
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := NewMessageEncryptor("token", key, "my-app")
	if err != nil {
		t.Fatal(err)
	}

	// The signature is legitimate -- same token -- but the message belongs to
	// another application and must not be accepted.
	if _, err := receiver.Decrypt(encrypted, signature, "1", "n"); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestMessageEncryptorValidation(t *testing.T) {
	if _, err := NewMessageEncryptor("", testAESKey(t), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty token err = %v", err)
	}
	if _, err := NewMessageEncryptor("token", "too-short", ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("short key err = %v", err)
	}
	if _, err := NewMessageEncryptor("token", strings.Repeat("!", 43), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("non-base64 key err = %v", err)
	}
}

func TestMessageEncryptorShortPayload(t *testing.T) {
	enc, err := NewMessageEncryptor("token", testAESKey(t), "")
	if err != nil {
		t.Fatal(err)
	}

	// A body that decrypts to fewer than 20 bytes cannot contain the header.
	short, err := EncryptCBC([]byte("abc"), mustRandom(t, 32), mustRandom(t, 16))
	if err != nil {
		t.Fatal(err)
	}

	encrypted := base64.StdEncoding.EncodeToString(short)
	signature := enc.Signature("1", "n", encrypted)

	if _, err := enc.Decrypt(encrypted, signature, "1", "n"); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestRSASignVerify(t *testing.T) {
	key := generateKey(t)

	msg := []byte("pay-v3-request")

	sig, err := SignSHA256(key, msg)
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifySHA256(&key.PublicKey, msg, sig); err != nil {
		t.Fatal(err)
	}

	if err := VerifySHA256(&key.PublicKey, []byte("tampered"), sig); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("tampered message err = %v", err)
	}
	if err := VerifySHA256(&key.PublicKey, msg, "not-base64"); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("bad base64 err = %v", err)
	}
	if err := VerifySHA256(nil, msg, sig); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil key err = %v", err)
	}
	if _, err := SignSHA256(nil, msg); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil sign key err = %v", err)
	}
}

func TestRSAOAEPRoundTrip(t *testing.T) {
	key := generateKey(t)

	ciphertext, err := EncryptOAEP(&key.PublicKey, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	plain, err := DecryptOAEP(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "secret" {
		t.Fatalf("plain = %q", plain)
	}

	if _, err := DecryptOAEP(key, "not-base64"); !errors.Is(err, errors.CodeCryptoFailed) {
		t.Fatalf("bad ciphertext err = %v", err)
	}
	if _, err := DecryptOAEP(nil, ciphertext); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil key err = %v", err)
	}
}

func TestParseRSAKeys(t *testing.T) {
	key := generateKey(t)

	pkcs1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8 := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8Bytes})

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pkix := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	parsedPKCS1, err := ParseRSAPrivateKeyPEM(pkcs1)
	if err != nil {
		t.Fatalf("pkcs1: %v", err)
	}
	parsedPKCS8, err := ParseRSAPrivateKeyPEM(pkcs8)
	if err != nil {
		t.Fatalf("pkcs8: %v", err)
	}
	if parsedPKCS1.N.Cmp(parsedPKCS8.N) != 0 {
		t.Fatal("the two private key forms disagree")
	}

	pub, err := ParseRSAPublicKeyPEM(pkix)
	if err != nil {
		t.Fatalf("pkix: %v", err)
	}
	if pub.N.Cmp(key.PublicKey.N) != 0 {
		t.Fatal("public key does not match")
	}

	if _, err := ParseRSAPrivateKeyPEM([]byte("garbage")); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("garbage err = %v", err)
	}
	if _, err := ParseRSAPublicKeyPEM([]byte("garbage")); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("garbage err = %v", err)
	}
}

func TestParseRSAPublicKeyFromPrivate(t *testing.T) {
	key := generateKey(t)

	// A PEM that holds a private key is not a public key; the parser must say so
	// rather than returning a half-initialized value.
	pkcs1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	if _, err := ParseRSAPublicKeyPEM(pkcs1); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

// generateKey builds a throwaway RSA key for the test.
func generateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	if err := key.Validate(); err != nil {
		t.Fatal(err)
	}

	return key
}
