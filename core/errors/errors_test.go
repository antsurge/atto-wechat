package errors

import (
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestNewAndMessage(t *testing.T) {
	err := New(CodeInvalidArgument, "bad input")

	if got := err.Error(); got != "INVALID_ARGUMENT: bad input" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestNewf(t *testing.T) {
	err := Newf(CodeSystemBusy, "errcode %d", -1)

	if err.Message != "errcode -1" {
		t.Fatalf("message = %q", err.Message)
	}
}

func TestWrapNilCauseIsNil(t *testing.T) {
	if err := Wrap(CodeRequestFailed, "failed", nil); err != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", err)
	}
}

func TestWrapUnwrap(t *testing.T) {
	cause := stderrors.New("dial tcp: refused")
	err := Wrap(CodeRequestFailed, "call failed", cause)

	if !stderrors.Is(err, cause) {
		t.Fatal("Unwrap does not expose the cause")
	}
	if got := err.Error(); !strings.Contains(got, "dial tcp: refused") {
		t.Fatalf("Error() = %q", got)
	}
}

func TestHTTPStatusInMessage(t *testing.T) {
	err := Newf(CodeNotFound, "missing")
	err.HTTPStatus = 404

	if got := err.Error(); !strings.Contains(got, "http 404") {
		t.Fatalf("Error() = %q", got)
	}
}

func TestIsAndCodeOf(t *testing.T) {
	err := New(CodeTokenExpired, "expired")

	if !Is(err, CodeTokenExpired) {
		t.Fatal("Is = false for matching code")
	}
	if Is(err, CodeTokenInvalid) {
		t.Fatal("Is = true for a different code")
	}
	if CodeOf(err) != CodeTokenExpired {
		t.Fatalf("CodeOf = %q", CodeOf(err))
	}

	if CodeOf(stderrors.New("plain")) != CodeUnknown {
		t.Fatal("CodeOf(plain) != CodeUnknown")
	}
}

func TestAsThroughWrapping(t *testing.T) {
	inner := New(CodeTimeout, "timeout")

	// A contextual wrapper from another package is the case As has to see
	// through; a *Error cause is the outermost SDK error and As returns it.
	outer := fmt.Errorf("call: %w", inner)

	got, ok := As(outer)
	if !ok {
		t.Fatal("As failed to unwrap")
	}
	if got.Code != CodeTimeout {
		t.Fatalf("code = %q, want TIMEOUT", got.Code)
	}
}

func TestClassifiers(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		retry    bool
		token    bool
		auth     bool
		argument bool
	}{
		{"system busy", CodeSystemBusy, true, false, false, false},
		{"rate limited", CodeRateLimited, true, false, false, false},
		{"timeout", CodeTimeout, false, false, false, false},
		{"expired token", CodeTokenExpired, false, true, false, false},
		{"invalid token", CodeTokenInvalid, false, true, false, false},
		{"unauthorized", CodeUnauthorized, false, false, true, false},
		{"forbidden", CodeForbidden, false, false, true, false},
		{"invalid argument", CodeInvalidArgument, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "x")

			if IsRetryable(err) != tt.retry {
				t.Errorf("IsRetryable = %v, want %v", IsRetryable(err), tt.retry)
			}
			if IsTokenExpired(err) != tt.token {
				t.Errorf("IsTokenExpired = %v, want %v", IsTokenExpired(err), tt.token)
			}
			if IsAuthError(err) != tt.auth {
				t.Errorf("IsAuthError = %v, want %v", IsAuthError(err), tt.auth)
			}
			if IsParamError(err) != tt.argument {
				t.Errorf("IsParamError = %v, want %v", IsParamError(err), tt.argument)
			}
		})
	}
}

func TestClassifiersIgnorePlainErrors(t *testing.T) {
	err := stderrors.New("plain")

	if IsRetryable(err) || IsTokenExpired(err) || IsAuthError(err) || IsParamError(err) {
		t.Fatal("a plain error was classified as an SDK error")
	}
}

func TestLogValue(t *testing.T) {
	err := Wrap(CodeDecodeFailed, "decode", stderrors.New("unexpected end"))

	val := err.LogValue()
	if val.Kind() != slog.KindGroup {
		t.Fatalf("kind = %s, want group", val.Kind())
	}

	var sawCode bool
	for _, attr := range val.Group() {
		if attr.Key == "code" && attr.Value.String() == "DECODE_FAILED" {
			sawCode = true
		}
	}
	if !sawCode {
		t.Fatal("log value does not expose the code")
	}
}
