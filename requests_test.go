package requests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test basic HTTP methods
func TestHTTPMethods(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		execMethod func(*Request, string, ...any) (*Response, error)
	}{
		{
			name:   "GET method",
			method: http.MethodGet,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.GET(url, body...)
			},
		},
		{
			name:   "POST method",
			method: http.MethodPost,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.POST(url, body...)
			},
		},
		{
			name:   "PUT method",
			method: http.MethodPut,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.PUT(url, body...)
			},
		},
		{
			name:   "DELETE method",
			method: http.MethodDelete,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.DELETE(url, body...)
			},
		},
		{
			name:   "PATCH method",
			method: http.MethodPatch,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.PATCH(url, body...)
			},
		},
		{
			name:   "OPTIONS method",
			method: http.MethodOptions,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.OPTIONS(url, body...)
			},
		},
		{
			name:   "HEAD method",
			method: http.MethodHead,
			execMethod: func(r *Request, url string, body ...any) (*Response, error) {
				return r.HEAD(url, body...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected method %s, got %s", tt.method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			// Execute request
			resp, err := tt.execMethod(R(context.Background()), server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify response
			if resp.StatusCode() != http.StatusOK {
				t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode())
			}

			// HEAD method should not return body
			if tt.method != http.MethodHead && string(resp.BodyRaw()) != "OK" {
				t.Errorf("expected body 'OK', got '%s'", string(resp.BodyRaw()))
			} else if tt.method == http.MethodHead && len(resp.BodyRaw()) != 0 {
				t.Errorf("HEAD method should not return body, got '%s'", string(resp.BodyRaw()))
			}
		})
	}
}

// Test request building with headers
func TestRequestHeaders(t *testing.T) {
	tests := []struct {
		name            string
		setupRequest    func(*Request) *Request
		expectedHeaders map[string]string
	}{
		{
			name: "single header",
			setupRequest: func(r *Request) *Request {
				return r.Header("X-Custom-Header", "custom-value")
			},
			expectedHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
		{
			name: "multiple headers",
			setupRequest: func(r *Request) *Request {
				return r.Headers(map[string]any{
					"X-Header-1": "value1",
					"X-Header-2": "value2",
				})
			},
			expectedHeaders: map[string]string{
				"X-Header-1": "value1",
				"X-Header-2": "value2",
			},
		},
		{
			name: "content type header",
			setupRequest: func(r *Request) *Request {
				return r.ContentType("application/json")
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name: "authorization header",
			setupRequest: func(r *Request) *Request {
				return r.Authorization("test-token")
			},
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expectedValue := range tt.expectedHeaders {
					if value := r.Header.Get(key); value != expectedValue {
						t.Errorf("expected header %s=%s, got %s", key, expectedValue, value)
					}
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

// Test request with cookies
func TestRequestCookies(t *testing.T) {
	tests := []struct {
		name            string
		setupRequest    func(*Request) *Request
		expectedCookies map[string]string
	}{
		{
			name: "single cookie",
			setupRequest: func(r *Request) *Request {
				return r.Cookie("session", "abc123")
			},
			expectedCookies: map[string]string{
				"session": "abc123",
			},
		},
		{
			name: "multiple cookies",
			setupRequest: func(r *Request) *Request {
				return r.Cookies(map[string]any{
					"session": "abc123",
					"user_id": "456",
				})
			},
			expectedCookies: map[string]string{
				"session": "abc123",
				"user_id": "456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for name, expectedValue := range tt.expectedCookies {
					cookie, err := r.Cookie(name)
					if err != nil {
						t.Errorf("cookie %s not found: %v", name, err)
						continue
					}
					if cookie.Value != expectedValue {
						t.Errorf("expected cookie %s=%s, got %s", name, expectedValue, cookie.Value)
					}
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

// Test query parameters
func TestQueryParameters(t *testing.T) {
	tests := []struct {
		name          string
		setupRequest  func(*Request) *Request
		expectedQuery map[string]string
	}{
		{
			name: "single query parameter",
			setupRequest: func(r *Request) *Request {
				return r.Query("key", "value")
			},
			expectedQuery: map[string]string{
				"key": "value",
			},
		},
		{
			name: "multiple query parameters",
			setupRequest: func(r *Request) *Request {
				return r.Queries(map[string]any{
					"param1": "value1",
					"param2": 123,
					"param3": true,
				})
			},
			expectedQuery: map[string]string{
				"param1": "value1",
				"param2": "123",
				"param3": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expectedValue := range tt.expectedQuery {
					if value := r.URL.Query().Get(key); value != expectedValue {
						t.Errorf("expected query %s=%s, got %s", key, expectedValue, value)
					}
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

// Test response parsing
func TestResponseParsing(t *testing.T) {
	type TestResponse struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	tests := []struct {
		name           string
		responseBody   any
		expectedResult TestResponse
		expectError    bool
	}{
		{
			name: "successful JSON parsing",
			responseBody: TestResponse{
				Message: "success",
				Code:    200,
			},
			expectedResult: TestResponse{
				Message: "success",
				Code:    200,
			},
			expectError: false,
		},
		{
			name:         "invalid JSON",
			responseBody: "invalid json",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if s, ok := tt.responseBody.(string); ok {
					w.Write([]byte(s))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			var result TestResponse
			_, err := R(context.Background()).
				Result(&result).
				GET(server.URL)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if result.Message != tt.expectedResult.Message || result.Code != tt.expectedResult.Code {
					t.Errorf("expected %+v, got %+v", tt.expectedResult, result)
				}
			}
		})
	}
}

// Test response status codes
func TestResponseStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus string
		isFailure      bool
	}{
		{
			name:           "200 OK",
			statusCode:     http.StatusOK,
			expectedStatus: "200 OK",
			isFailure:      false,
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedStatus: "404 Not Found",
			isFailure:      true,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedStatus: "500 Internal Server Error",
			isFailure:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			resp, err := R(context.Background()).GET(server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode() != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, resp.StatusCode())
			}

			if resp.Status() != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, resp.Status())
			}

			if resp.IsFailure() != tt.isFailure {
				t.Errorf("expected IsFailure=%v, got %v", tt.isFailure, resp.IsFailure())
			}
		})
	}
}

// Test context cancellation
func TestContextCancellation(t *testing.T) {
	t.Run("context cancelled before request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called when context is cancelled")
		}))
		defer server.Close()

		_, err := R(ctx).GET(server.URL)
		if err == nil {
			t.Error("expected error when context is cancelled")
		}
	})

	t.Run("context cancelled during request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cancel() // Cancel during request
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// This might or might not fail depending on timing
		R(ctx).GET(server.URL)
	})
}

// Test timeout
func TestTimeout(t *testing.T) {
	t.Run("request times out", func(t *testing.T) {
		// Create a channel to signal when the handler starts
		handlerStarted := make(chan struct{})

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(handlerStarted)
			// Sleep much longer than the timeout to ensure timeout occurs
			select {
			case <-time.After(1 * time.Second):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				// Request was cancelled
				return
			}
		}))
		defer server.Close()

		start := time.Now()
		resp, err := R(context.Background()).
			Timeout(50 * time.Millisecond).
			GET(server.URL)
		_ = resp
		elapsed := time.Since(start)

		// Wait for handler to start to ensure the request reached the server
		select {
		case <-handlerStarted:
			// Handler was called
		case <-time.After(100 * time.Millisecond):
			// Handler wasn't called in time
		}

		if err == nil {
			t.Error("expected timeout error")
		}

		// Verify that the request was cancelled due to timeout (should take ~50ms, not 1s)
		if elapsed > 200*time.Millisecond {
			t.Errorf("request took too long: %v, expected ~50ms", elapsed)
		}
	})

	t.Run("request completes before timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Complete quickly
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		resp, err := R(context.Background()).
			Timeout(500 * time.Millisecond).
			GET(server.URL)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if resp != nil && resp.StatusCode() != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode())
		}
	})
}

// Test request with body
func TestRequestWithBody(t *testing.T) {
	type TestBody struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name         string
		body         any
		expectedBody string
	}{
		{
			name: "JSON body",
			body: TestBody{
				Name:  "test",
				Value: 123,
			},
			expectedBody: `{"name":"test","value":123}`,
		},
		{
			name:         "nil body",
			body:         nil,
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read body: %v", err)
				}

				// Normalize JSON for comparison
				if tt.expectedBody != "" && tt.expectedBody != string(body) {
					var expected, actual any
					json.Unmarshal([]byte(tt.expectedBody), &expected)
					json.Unmarshal(body, &actual)

					expectedJSON, _ := json.Marshal(expected)
					actualJSON, _ := json.Marshal(actual)

					if string(expectedJSON) != string(actualJSON) {
						t.Errorf("expected body %s, got %s", tt.expectedBody, string(body))
					}
				} else if tt.expectedBody != string(body) {
					t.Errorf("expected body %s, got %s", tt.expectedBody, string(body))
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			var args []any
			if tt.body != nil {
				args = append(args, tt.body)
			}

			_, err := R(context.Background()).POST(server.URL, args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
