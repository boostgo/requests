package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test newParseResponseBodyError
func TestNewParseResponseBodyError(t *testing.T) {
	url := "https://example.com/api"
	code := 400
	blob := []byte("bad request")

	err := newParseResponseBodyError(url, code, blob)

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// Check if the error contains the expected data
	data, ok := err.Data().(responseBodyContext)
	if !ok {
		t.Fatal("error data is not of type responseBodyContext")
	}

	if data.URL != url {
		t.Errorf("expected URL %s, got %s", url, data.URL)
	}

	if data.Code != code {
		t.Errorf("expected code %d, got %d", code, data.Code)
	}

	if string(data.Blob) != string(blob) {
		t.Errorf("expected blob %s, got %s", string(blob), string(data.Blob))
	}
}

// Test various error scenarios
func TestRequestErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupTest   func() (*Response, error)
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "invalid URL",
			setupTest: func() (*Response, error) {
				return R(context.Background()).GET("://invalid-url")
			},
			expectError: true,
		},
		{
			name: "network error - connection refused",
			setupTest: func() (*Response, error) {
				// Use a port that's unlikely to be in use
				return R(context.Background()).GET("http://localhost:65534")
			},
			expectError: true,
		},
		{
			name: "parse error - non-pointer result",
			setupTest: func() (*Response, error) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(`{"name":"test"}`))
				}))
				defer server.Close()

				var result struct{ Name string }
				return R(context.Background()).
					Result(result). // Non-pointer
					GET(server.URL)
			},
			expectError: false, // Request succeeds, but parsing would fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.setupTest()

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if err != nil && tt.errorCheck != nil && !tt.errorCheck(err) {
				t.Errorf("error check failed: %v", err)
			}
		})
	}
}

// Test context error scenarios
func TestContextErrors(t *testing.T) {
	t.Run("context with existing error", func(t *testing.T) {
		// Create a custom context that returns an error
		ctx := &errorContext{
			Context: context.Background(),
			err:     context.DeadlineExceeded,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called when context has error")
		}))
		defer server.Close()

		_, err := R(ctx).GET(server.URL)
		if err == nil {
			t.Error("expected error when context has error")
		}
	})
}

// Custom context for testing
type errorContext struct {
	context.Context
	err error
}

func (c *errorContext) Err() error {
	return c.err
}

// Test error wrapping
func TestErrorWrapping(t *testing.T) {
	t.Run("retry do error wrapping", func(t *testing.T) {
		// Force an error by using invalid URL
		_, err := R(context.Background()).GET("://invalid")

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		// Check if error is wrapped with ErrRequestRetryDo
		// The actual error checking would depend on the errorx implementation
	})
}
