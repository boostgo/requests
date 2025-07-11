package requests

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/boostgo/reflectx"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test Response methods
func TestResponseMethods(t *testing.T) {
	type TestData struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	t.Run("test all response methods", func(t *testing.T) {
		testData := TestData{
			Message: "test message",
			Count:   42,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(testData)
		}))
		defer server.Close()

		resp, err := R(context.Background()).GET(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Test Raw()
		if resp.Raw() == nil {
			t.Error("Raw() returned nil")
		}

		// Test Status()
		if resp.Status() != "201 Created" {
			t.Errorf("expected status '201 Created', got '%s'", resp.Status())
		}

		// Test StatusCode()
		if resp.StatusCode() != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, resp.StatusCode())
		}

		// Test ContentType()
		if resp.ContentType() != "application/json" {
			t.Errorf("expected content type 'application/json', got '%s'", resp.ContentType())
		}

		// Test BodyRaw()
		bodyRaw := resp.BodyRaw()
		if len(bodyRaw) == 0 {
			t.Error("BodyRaw() returned empty slice")
		}

		// Test Parse()
		var result TestData
		err = resp.Parse(&result)
		if err != nil {
			t.Errorf("Parse() failed: %v", err)
		}

		if result.Message != testData.Message || result.Count != testData.Count {
			t.Errorf("expected %+v, got %+v", testData, result)
		}

		// Test IsFailure()
		if resp.IsFailure() {
			t.Error("IsFailure() should return false for 201 status")
		}
	})
}

// Test Parse with different scenarios
func TestResponseParse(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		target       any
		expectError  bool
		errorCheck   func(error) bool
	}{
		{
			name:         "parse to struct",
			responseBody: `{"name":"John","age":30}`,
			target: &struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			expectError: false,
		},
		{
			name:         "parse to map",
			responseBody: `{"key1":"value1","key2":"value2"}`,
			target:       &map[string]string{},
			expectError:  false,
		},
		{
			name:         "parse to slice",
			responseBody: `[1,2,3,4,5]`,
			target:       &[]int{},
			expectError:  false,
		},
		{
			name:         "parse invalid JSON",
			responseBody: `{invalid json}`,
			target: &struct {
				Name string `json:"name"`
			}{},
			expectError: true,
		},
		{
			name:         "parse to non-pointer",
			responseBody: `{"name":"test"}`,
			target: struct {
				Name string `json:"name"`
			}{},
			expectError: true,
			errorCheck: func(err error) bool {
				return errors.Is(err, reflectx.ErrCheckExport)
			},
		},
		{
			name:         "parse empty response",
			responseBody: `{}`, // Use empty JSON object instead of empty string
			target:       &struct{}{},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			resp, err := R(context.Background()).GET(server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = resp.Parse(tt.target)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Errorf("error check failed: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Test Context method
func TestResponseContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalCtx := context.WithValue(context.Background(), "test-key", "test-value")
	resp, err := R(originalCtx).GET(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newCtx := resp.Context(context.Background())
	if newCtx == nil {
		t.Error("Context() returned nil")
	}
}

// Test IsFailure with different status codes
func TestResponseIsFailure(t *testing.T) {
	tests := []struct {
		statusCode int
		isFailure  bool
	}{
		{http.StatusOK, false},
		{http.StatusCreated, false},
		{http.StatusAccepted, false},
		{http.StatusNoContent, false},
		{http.StatusMovedPermanently, false},
		{http.StatusFound, false},
		{http.StatusBadRequest, true},
		{http.StatusUnauthorized, true},
		{http.StatusForbidden, true},
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			resp, err := R(context.Background()).GET(server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.IsFailure() != tt.isFailure {
				t.Errorf("expected IsFailure=%v for status %d, got %v",
					tt.isFailure, tt.statusCode, resp.IsFailure())
			}
		})
	}
}
