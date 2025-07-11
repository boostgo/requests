package requests

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test Basic Authentication
func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name               string
		username           string
		password           string
		expectedAuthHeader string
	}{
		{
			name:               "valid credentials",
			username:           "user",
			password:           "pass",
			expectedAuthHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name:               "username with special characters",
			username:           "user@example.com",
			password:           "p@ssw0rd!",
			expectedAuthHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:p@ssw0rd!")),
		},
		{
			name:               "empty password",
			username:           "user",
			password:           "",
			expectedAuthHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.expectedAuthHeader {
					t.Errorf("expected Authorization header '%s', got '%s'", tt.expectedAuthHeader, authHeader)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			_, err := R(context.Background()).
				BasicAuth(tt.username, tt.password).
				GET(server.URL)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Test Bearer Token Authentication
func TestBearerToken(t *testing.T) {
	tests := []struct {
		name               string
		token              string
		expectedAuthHeader string
	}{
		{
			name:               "simple token",
			token:              "abc123",
			expectedAuthHeader: "Bearer abc123",
		},
		{
			name:               "JWT token",
			token:              "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectedAuthHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		},
		{
			name:               "token with special characters",
			token:              "token-with-special_chars.123",
			expectedAuthHeader: "Bearer token-with-special_chars.123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.expectedAuthHeader {
					t.Errorf("expected Authorization header '%s', got '%s'", tt.expectedAuthHeader, authHeader)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			_, err := R(context.Background()).
				BearerToken(tt.token).
				GET(server.URL)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Test Authorization header (from Request)
func TestAuthorizationHeader(t *testing.T) {
	token := "test-token-123"
	expectedHeader := "Bearer " + token

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != expectedHeader {
			t.Errorf("expected Authorization header '%s', got '%s'", expectedHeader, authHeader)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := R(context.Background()).
		Authorization(token).
		GET(server.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test authentication priority (when multiple auth methods are set)
func TestAuthenticationPriority(t *testing.T) {
	tests := []struct {
		name               string
		setupRequest       func(*Request) *Request
		expectedAuthHeader string
	}{
		{
			name: "basic auth overrides bearer token",
			setupRequest: func(r *Request) *Request {
				return r.
					BearerToken("bearer-token").
					BasicAuth("user", "pass")
			},
			expectedAuthHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name: "bearer token when no basic auth username",
			setupRequest: func(r *Request) *Request {
				return r.
					BasicAuth("", "pass").
					BearerToken("bearer-token")
			},
			expectedAuthHeader: "Bearer bearer-token",
		},
		{
			name: "authorization header overrides bearer token",
			setupRequest: func(r *Request) *Request {
				return r.
					BearerToken("bearer-token").
					Authorization("custom-token")
			},
			expectedAuthHeader: "Bearer custom-token",
		},
		{
			name: "custom authorization header",
			setupRequest: func(r *Request) *Request {
				return r.
					Header("Authorization", "Custom custom-scheme-token")
			},
			expectedAuthHeader: "Custom custom-scheme-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.expectedAuthHeader {
					t.Errorf("expected Authorization header '%s', got '%s'", tt.expectedAuthHeader, authHeader)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			req := R(context.Background())
			req = tt.setupRequest(req)

			_, err := req.GET(server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Test Client-level authentication
func TestClientAuthentication(t *testing.T) {
	t.Run("client basic auth", func(t *testing.T) {
		username := "client-user"
		password := "client-pass"
		expectedHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != expectedHeader {
				t.Errorf("expected Authorization header '%s', got '%s'", expectedHeader, authHeader)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := New().BasicAuth(username, password)
		_, err := client.R(context.Background()).GET(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("client bearer token", func(t *testing.T) {
		token := "client-bearer-token"
		expectedHeader := "Bearer " + token

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != expectedHeader {
				t.Errorf("expected Authorization header '%s', got '%s'", expectedHeader, authHeader)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := New().BearerToken(token)
		_, err := client.R(context.Background()).GET(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("client authorization header", func(t *testing.T) {
		token := "client-auth-token"
		expectedHeader := "Bearer " + token

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != expectedHeader {
				t.Errorf("expected Authorization header '%s', got '%s'", expectedHeader, authHeader)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := New().Authorization(token)
		_, err := client.R(context.Background()).GET(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// Test request-level auth overrides client-level auth
func TestRequestAuthOverridesClientAuth(t *testing.T) {
	clientToken := "client-token"
	requestToken := "request-token"
	expectedHeader := "Bearer " + requestToken

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != expectedHeader {
			t.Errorf("expected Authorization header '%s', got '%s'", expectedHeader, authHeader)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New().BearerToken(clientToken)
	_, err := client.R(context.Background()).
		BearerToken(requestToken).
		GET(server.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test edge cases
func TestAuthenticationEdgeCases(t *testing.T) {
	t.Run("empty username in basic auth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			// When username is empty, basic auth should not be set
			if strings.HasPrefix(authHeader, "Basic") {
				t.Errorf("Basic auth should not be set with empty username, got '%s'", authHeader)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := R(context.Background()).
			BasicAuth("", "password").
			GET(server.URL)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty bearer token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			// Empty bearer token should still set the header
			if authHeader != "" && authHeader != "Bearer " {
				t.Errorf("unexpected Authorization header '%s'", authHeader)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := R(context.Background()).
			BearerToken("").
			GET(server.URL)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
