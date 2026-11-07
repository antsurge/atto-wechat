package config

import (
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	c := New()

	if c.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q", c.UserAgent)
	}
	if c.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("HTTPTimeout = %v", c.HTTPTimeout)
	}
	if c.RetryAttempts != DefaultRetryAttempts {
		t.Errorf("RetryAttempts = %d", c.RetryAttempts)
	}
	if c.Logger == nil {
		t.Error("Logger = nil")
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("defaults do not validate: %v", err)
	}
}

func TestOptions(t *testing.T) {
	hc := &http.Client{Timeout: time.Second}

	c := New(
		WithBaseURL("https://example.com/"),
		WithHTTPTimeout(time.Second),
		WithUserAgent("test-agent"),
		WithRetry(3, 10*time.Millisecond),
		WithRetryMaxBackoff(50*time.Millisecond),
		WithLogLevel(slog.LevelError),
		WithHTTPClient(hc),
		nil,
	)

	if c.BaseURL != "https://example.com/" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.RetryAttempts != 3 || c.RetryBackoff != 10*time.Millisecond {
		t.Errorf("retry = %d/%v", c.RetryAttempts, c.RetryBackoff)
	}
	if c.RetryMaxBackoff != 50*time.Millisecond {
		t.Errorf("RetryMaxBackoff = %v", c.RetryMaxBackoff)
	}
	if c.LogLevel != slog.LevelError {
		t.Errorf("LogLevel = %v", c.LogLevel)
	}
	if c.HTTPClient != hc {
		t.Error("HTTPClient not injected")
	}
}

func TestNilLoggerAndClientIgnored(t *testing.T) {
	c := New(WithLogger(nil), WithHTTPClient(nil))

	if c.Logger == nil {
		t.Fatal("nil logger replaced the default")
	}
	if c.HTTPClient != nil {
		t.Fatal("nil client was stored")
	}
}

func TestValidateRejectsBadConfig(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CoreConfig)
	}{
		{"empty base url", func(c *CoreConfig) { c.BaseURL = "" }},
		{"zero timeout", func(c *CoreConfig) { c.HTTPTimeout = 0 }},
	 {"negative retries", func(c *CoreConfig) { c.RetryAttempts = -1 }},
		{"negative backoff", func(c *CoreConfig) { c.RetryBackoff = -time.Second }},
		{"negative max backoff", func(c *CoreConfig) { c.RetryMaxBackoff = -time.Second }},
		{"backoff exceeds max", func(c *CoreConfig) {
			c.RetryBackoff = 2 * time.Second
			c.RetryMaxBackoff = time.Second
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			tt.mutate(c)

			if err := c.Validate(); err == nil {
				t.Fatal("Validate accepted an invalid config")
			}
		})
	}
}

func TestValidateNil(t *testing.T) {
	var c *CoreConfig

	if err := c.Validate(); err == nil {
		t.Fatal("Validate(nil) returned nil error")
	}
}

func TestCloneIsIndependent(t *testing.T) {
	c := New()
	cp := c.Clone()

	cp.BaseURL = "https://other"

	if c.BaseURL != DefaultBaseURL {
		t.Fatal("Clone shares state with the original")
	}

	if (*CoreConfig)(nil).Clone() != nil {
		t.Fatal("Clone(nil) != nil")
	}
}
