package errors

// Code represents an SDK error code.
//
// Codes are stable, machine-readable strings rather than WeChat's numeric
// errcode values. The two answer different questions: the numeric errcode says
// what WeChat replied, while a Code says what the SDK decided the failure
// means, which is what callers branch on. A response that carries a business
// errcode is not an error at all (see core/response), so it never reaches this
// type.
type Code string

const (
	// General errors.
	CodeUnknown         Code = "UNKNOWN"
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	CodeInvalidResponse Code = "INVALID_RESPONSE"

	// HTTP errors.
	CodeRequestFailed Code = "REQUEST_FAILED"
	CodeUnauthorized  Code = "UNAUTHORIZED"
	CodeForbidden     Code = "FORBIDDEN"
	CodeNotFound      Code = "NOT_FOUND"
	CodeTimeout       Code = "TIMEOUT"
	CodeCancelled     Code = "CANCELLED"

	// Remote state that is transient: the same request may succeed if it is
	// replayed later.
	CodeSystemBusy  Code = "SYSTEM_BUSY"
	CodeRateLimited Code = "RATE_LIMITED"

	// Authentication errors.
	CodeTokenExpired Code = "TOKEN_EXPIRED"
	CodeTokenInvalid Code = "TOKEN_INVALID"

	// Serialization errors.
	CodeEncodeFailed Code = "ENCODE_FAILED"
	CodeDecodeFailed Code = "DECODE_FAILED"

	// Cryptography errors: signature mismatch, bad key, tampered payload.
	CodeCryptoFailed Code = "CRYPTO_FAILED"
)
