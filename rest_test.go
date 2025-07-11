package requests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test Get convenience function
func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	resp, err := Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode())
	}

	var result map[string]string
	err = resp.Parse(&result)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result["status"])
	}
}

// Test Post convenience function
func TestPost(t *testing.T) {
	type RequestBody struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	body := RequestBody{
		Name:  "test",
		Value: 123,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}

		var received RequestBody
		err = json.Unmarshal(reqBody, &received)
		if err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
		}

		if received.Name != body.Name || received.Value != body.Value {
			t.Errorf("expected %+v, got %+v", body, received)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	resp, err := Post(context.Background(), body, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode())
	}
}

// Test Put convenience function
func TestPut(t *testing.T) {
	type UpdateBody struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	body := UpdateBody{
		ID:   1,
		Name: "updated",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT method, got %s", r.Method)
		}

		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}

		var received UpdateBody
		err = json.Unmarshal(reqBody, &received)
		if err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
		}

		if received.ID != body.ID || received.Name != body.Name {
			t.Errorf("expected %+v, got %+v", body, received)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := Put(context.Background(), body, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode())
	}
}

// Test Delete convenience function
func TestDelete(t *testing.T) {
	tests := []struct {
		name     string
		withBody bool
		body     any
	}{
		{
			name:     "delete without body",
			withBody: false,
		},
		{
			name:     "delete with body",
			withBody: true,
			body:     map[string]string{"reason": "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE method, got %s", r.Method)
				}

				if tt.withBody {
					reqBody, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("failed to read request body: %v", err)
					}

					if len(reqBody) == 0 {
						t.Error("expected body but got empty")
					}
				}

				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			var resp *Response
			var err error

			if tt.withBody {
				resp, err = Delete(context.Background(), server.URL, tt.body)
			} else {
				resp, err = Delete(context.Background(), server.URL)
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.StatusCode() != http.StatusNoContent {
				t.Errorf("expected status 204, got %d", resp.StatusCode())
			}
		})
	}
}

// Test convenience functions with context cancellation
func TestConvenienceFunctionsWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when context is cancelled")
	}))
	defer server.Close()

	t.Run("Get with cancelled context", func(t *testing.T) {
		_, err := Get(ctx, server.URL)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("Post with cancelled context", func(t *testing.T) {
		_, err := Post(ctx, map[string]string{"test": "data"}, server.URL)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("Put with cancelled context", func(t *testing.T) {
		_, err := Put(ctx, map[string]string{"test": "data"}, server.URL)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("Delete with cancelled context", func(t *testing.T) {
		_, err := Delete(ctx, server.URL)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})
}
