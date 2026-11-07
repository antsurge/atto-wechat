package security

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"

	"github.com/antsurge/atto-wechat/core/errors"
)

// encodingAESKeyLength is the length of the EncodingAESKey shown in the WeChat
// console: 43 base64 characters without padding, i.e. 32 bytes.
const encodingAESKeyLength = 43

// MessageEncryptor implements WeChat's callback message encryption: the
// "safe mode" envelope described by the platform's message encryption spec.
//
// The envelope is not just AES-CBC. The plaintext is
//
//	random(16) | len(msg) as uint32 big-endian | msg | appID
//
// and the trailing appID is what proves the message was addressed to this
// application: a valid signature only proves the sender knows the token, which
// is shared by every app configured with it.
type MessageEncryptor struct {
	token string
	key   []byte
	iv    []byte
	appID string
}

// NewMessageEncryptor builds an encryptor from the console credentials.
//
// appID may be empty to skip the appID check, but then a message meant for
// another application on the same token would be accepted.
func NewMessageEncryptor(token, encodingAESKey, appID string) (*MessageEncryptor, error) {
	if token == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "callback token must not be empty")
	}

	if len(encodingAESKey) != encodingAESKeyLength {
		return nil, errors.Newf(
			errors.CodeInvalidArgument,
			"encoding aes key must be %d characters",
			encodingAESKeyLength,
		)
	}

	// The console key omits base64 padding, so one '=' restores it.
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil || len(key) != 32 {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "decode encoding aes key", err)
	}

	return &MessageEncryptor{
		token: token,
		key:   key,
		// The spec fixes IV to the first 16 bytes of the key. That is safe only
		// because each message starts with 16 random bytes.
		iv:    key[:blockSize],
		appID: appID,
	}, nil
}

// Signature computes the msg_signature for a callback request.
func (e *MessageEncryptor) Signature(timestamp, nonce, encrypt string) string {
	return SHA1Signature(e.token, timestamp, nonce, encrypt)
}

// VerifySignature checks a msg_signature in constant time.
func (e *MessageEncryptor) VerifySignature(signature, timestamp, nonce, encrypt string) bool {
	return ConstantTimeEqual(
		[]byte(e.Signature(timestamp, nonce, encrypt)),
		[]byte(signature),
	)
}

// Decrypt verifies the signature and decrypts the message body.
//
// The signature is checked before decryption, and the appID after: an attacker
// who can reach this code should not get to feed arbitrary ciphertext into the
// AES path, and a message that decrypts but belongs to another app is still not
// ours.
func (e *MessageEncryptor) Decrypt(
	encrypt, signature, timestamp, nonce string,
) ([]byte, error) {
	if !e.VerifySignature(signature, timestamp, nonce, encrypt) {
		return nil, errors.New(errors.CodeCryptoFailed, "callback signature mismatch")
	}

	raw, err := base64.StdEncoding.DecodeString(encrypt)
	if err != nil {
		return nil, errors.Wrap(errors.CodeCryptoFailed, "decode encrypted message", err)
	}

	plain, err := DecryptCBC(raw, e.key, e.iv)
	if err != nil {
		return nil, err
	}

	// 16 random bytes + 4 length bytes is the minimum header.
	if len(plain) < blockSize+4 {
		return nil, errors.New(errors.CodeCryptoFailed, "decrypted message is too short")
	}

	msgLen := int(binary.BigEndian.Uint32(plain[blockSize : blockSize+4]))
	if msgLen < 0 || msgLen > len(plain)-(blockSize+4) {
		return nil, errors.New(errors.CodeCryptoFailed, "invalid message length")
	}

	msg := plain[blockSize+4 : blockSize+4+msgLen]
	fromAppID := string(plain[blockSize+4+msgLen:])

	if e.appID != "" && fromAppID != e.appID {
		return nil, errors.New(
			errors.CodeCryptoFailed,
			"message appID does not match this application",
		)
	}

	// Return a copy so the caller cannot mutate the decrypted buffer in place.
	out := make([]byte, len(msg))
	copy(out, msg)

	return out, nil
}

// Encrypt wraps reply with the same envelope and returns the ciphertext and its
// signature.
func (e *MessageEncryptor) Encrypt(msg, timestamp, nonce string) (encrypt, signature string, err error) {
	plain := make([]byte, blockSize+4+len(msg)+len(e.appID))

	if _, err := rand.Read(plain[:blockSize]); err != nil {
		return "", "", errors.Wrap(errors.CodeCryptoFailed, "generate random prefix", err)
	}

	binary.BigEndian.PutUint32(plain[blockSize:blockSize+4], uint32(len(msg)))
	copy(plain[blockSize+4:], msg)
	copy(plain[blockSize+4+len(msg):], e.appID)

	sealed, err := EncryptCBC(plain, e.key, e.iv)
	if err != nil {
		return "", "", err
	}

	encrypt = base64.StdEncoding.EncodeToString(sealed)
	signature = e.Signature(timestamp, nonce, encrypt)

	return encrypt, signature, nil
}
