package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Test JSON request/response handling
func TestJSONContent(t *testing.T) {
	type TestData struct {
		Name   string `json:"name"`
		Value  int    `json:"value"`
		Active bool   `json:"active"`
	}

	t.Run("send and receive JSON", func(t *testing.T) {
		requestData := TestData{
			Name:   "test",
			Value:  42,
			Active: true,
		}

		responseData := TestData{
			Name:   "response",
			Value:  100,
			Active: false,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") {
				t.Errorf("expected JSON content type, got %s", contentType)
			}

			// Parse request body
			var received TestData
			err := json.NewDecoder(r.Body).Decode(&received)
			if err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}

			if received != requestData {
				t.Errorf("expected %+v, got %+v", requestData, received)
			}

			// Send response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseData)
		}))
		defer server.Close()

		var result TestData
		resp, err := R(context.Background()).
			Result(&result).
			POST(server.URL, requestData)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != responseData {
			t.Errorf("expected response %+v, got %+v", responseData, result)
		}

		if resp.ContentType() != "application/json" {
			t.Errorf("expected content type application/json, got %s", resp.ContentType())
		}
	})
}

// Test FormData writer
func TestFormDataWriter(t *testing.T) {
	t.Run("create and use FormData", func(t *testing.T) {
		// Create FormData
		formData := NewFormData()
		err := formData.Add("name", "John Doe")
		if err != nil {
			t.Fatalf("failed to add field: %v", err)
		}

		err = formData.Add("age", 30)
		if err != nil {
			t.Fatalf("failed to add field: %v", err)
		}

		err = formData.AddFile("avatar", "avatar.txt", []byte("file content"))
		if err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Parse multipart form
			err := r.ParseMultipartForm(10 << 20) // 10 MB
			if err != nil {
				t.Errorf("failed to parse multipart form: %v", err)
			}

			// Check fields
			if name := r.FormValue("name"); name != "John Doe" {
				t.Errorf("expected name 'John Doe', got '%s'", name)
			}

			if age := r.FormValue("age"); age != "30" {
				t.Errorf("expected age '30', got '%s'", age)
			}

			// Check file
			file, header, err := r.FormFile("avatar")
			if err != nil {
				t.Errorf("failed to get file: %v", err)
			}
			defer file.Close()

			if header.Filename != "avatar.txt" {
				t.Errorf("expected filename 'avatar.txt', got '%s'", header.Filename)
			}

			content, err := io.ReadAll(file)
			if err != nil {
				t.Errorf("failed to read file: %v", err)
			}

			if string(content) != "file content" {
				t.Errorf("expected file content 'file content', got '%s'", string(content))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err = R(context.Background()).POST(server.URL, formData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("FormData with initial data", func(t *testing.T) {
		initialData := map[string]any{
			"field1": "value1",
			"field2": 123,
			"field3": true,
		}

		formData := NewFormData(initialData)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			if err != nil {
				t.Errorf("failed to parse multipart form: %v", err)
			}

			if v := r.FormValue("field1"); v != "value1" {
				t.Errorf("expected field1='value1', got '%s'", v)
			}

			if v := r.FormValue("field2"); v != "123" {
				t.Errorf("expected field2='123', got '%s'", v)
			}

			if v := r.FormValue("field3"); v != "true" {
				t.Errorf("expected field3='true', got '%s'", v)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := R(context.Background()).POST(server.URL, formData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("FormData content type", func(t *testing.T) {
		formData := NewFormData()
		formData.Add("test", "value")

		boundary := formData.Boundary()
		expectedContentType := formData.ContentType()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")

			// Parse media type
			mediaType, params, err := mime.ParseMediaType(contentType)
			if err != nil {
				t.Errorf("failed to parse content type: %v", err)
			}

			if mediaType != "multipart/form-data" {
				t.Errorf("expected media type 'multipart/form-data', got '%s'", mediaType)
			}

			if params["boundary"] != boundary {
				t.Errorf("expected boundary '%s', got '%s'", boundary, params["boundary"])
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := R(context.Background()).POST(server.URL, formData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify ContentType method
		if !strings.Contains(expectedContentType, "multipart/form-data") {
			t.Errorf("ContentType() should contain 'multipart/form-data', got '%s'", expectedContentType)
		}
	})
}

// Test BytesWriter
func TestBytesWriter(t *testing.T) {
	t.Run("write bytes with default content type", func(t *testing.T) {
		writer := NewBytesWriter()
		testData := []byte("Hello, World!")

		n, err := writer.Write(testData)
		if err != nil {
			t.Fatalf("failed to write bytes: %v", err)
		}

		if n != len(testData) {
			t.Errorf("expected to write %d bytes, wrote %d", len(testData), n)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/octet-stream" {
				t.Errorf("expected content type 'application/octet-stream', got '%s'", contentType)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("failed to read body: %v", err)
			}

			if !bytes.Equal(body, testData) {
				t.Errorf("expected body '%s', got '%s'", string(testData), string(body))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err = R(context.Background()).POST(server.URL, writer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("write bytes with custom content type", func(t *testing.T) {
		writer := NewBytesWriter()
		writer.SetContentType("text/plain")

		testData := []byte("Plain text content")
		writer.Write(testData)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if contentType != "text/plain" {
				t.Errorf("expected content type 'text/plain', got '%s'", contentType)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("failed to read body: %v", err)
			}

			if string(body) != string(testData) {
				t.Errorf("expected body '%s', got '%s'", string(testData), string(body))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := R(context.Background()).POST(server.URL, writer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("BytesWriter methods", func(t *testing.T) {
		writer := NewBytesWriter()
		testData := []byte("test data")

		// Test Write
		n, err := writer.Write(testData)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(testData) {
			t.Errorf("expected to write %d bytes, wrote %d", len(testData), n)
		}

		// Test Bytes
		if !bytes.Equal(writer.Bytes(), testData) {
			t.Errorf("Bytes() returned %v, expected %v", writer.Bytes(), testData)
		}

		// Test ContentType
		if writer.ContentType() != "application/octet-stream" {
			t.Errorf("default content type should be 'application/octet-stream', got '%s'", writer.ContentType())
		}

		// Test SetContentType
		writer.SetContentType("custom/type")
		if writer.ContentType() != "custom/type" {
			t.Errorf("content type should be 'custom/type', got '%s'", writer.ContentType())
		}

		// Test Reader
		reader := writer.Reader()
		if reader == nil {
			t.Error("Reader() returned nil")
		}

		readData, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read from Reader: %v", err)
		}

		if !bytes.Equal(readData, testData) {
			t.Errorf("Reader data mismatch: got %v, expected %v", readData, testData)
		}
	})
}

// Test FormUrlEncodedWriter
func TestFormUrlEncodedWriter(t *testing.T) {
	t.Run("form URL encoded data", func(t *testing.T) {
		writer := NewFormUrlEncodedWriter()
		writer.Set("name", "John Doe")
		writer.Set("email", "john@example.com")
		writer.Set("age", "30")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/x-www-form-urlencoded" {
				t.Errorf("expected content type 'application/x-www-form-urlencoded', got '%s'", contentType)
			}

			err := r.ParseForm()
			if err != nil {
				t.Errorf("failed to parse form: %v", err)
			}

			if name := r.FormValue("name"); name != "John Doe" {
				t.Errorf("expected name 'John Doe', got '%s'", name)
			}

			if email := r.FormValue("email"); email != "john@example.com" {
				t.Errorf("expected email 'john@example.com', got '%s'", email)
			}

			if age := r.FormValue("age"); age != "30" {
				t.Errorf("expected age '30', got '%s'", age)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := R(context.Background()).POST(server.URL, writer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("FormUrlEncodedWriter methods", func(t *testing.T) {
		writer := NewFormUrlEncodedWriter()

		// Test Set and Get
		writer.Set("key1", "value1")
		if writer.Get("key1") != "value1" {
			t.Errorf("expected Get('key1') to return 'value1', got '%s'", writer.Get("key1"))
		}

		// Test Has
		if !writer.Has("key1") {
			t.Error("Has('key1') should return true")
		}
		if writer.Has("nonexistent") {
			t.Error("Has('nonexistent') should return false")
		}

		// Test Delete
		writer.Delete("key1")
		if writer.Has("key1") {
			t.Error("key1 should be deleted")
		}

		// Test Reader with special characters
		writer.Set("special", "value with spaces & symbols")
		reader := writer.Reader()
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read from Reader: %v", err)
		}

		values, err := url.ParseQuery(string(data))
		if err != nil {
			t.Fatalf("failed to parse query: %v", err)
		}

		if values.Get("special") != "value with spaces & symbols" {
			t.Errorf("special characters not properly encoded/decoded")
		}
	})
}

// Test content type priority
func TestContentTypePriority(t *testing.T) {
	tests := []struct {
		name                string
		setupRequest        func(*Request) *Request
		body                any
		expectedContentType string
	}{
		{
			name: "JSON body sets content type",
			setupRequest: func(r *Request) *Request {
				return r
			},
			body:                map[string]string{"key": "value"},
			expectedContentType: "", // JSON doesn't set content type header by default
		},
		{
			name: "FormData overrides content type",
			setupRequest: func(r *Request) *Request {
				return r.ContentType("application/json")
			},
			body: func() FormDataWriter {
				fd := NewFormData()
				fd.Add("test", "value")
				return fd
			}(),
			expectedContentType: "multipart/form-data",
		},
		{
			name: "BytesWriter sets its content type",
			setupRequest: func(r *Request) *Request {
				return r
			},
			body: func() BytesWriter {
				bw := NewBytesWriter()
				bw.SetContentType("custom/binary")
				bw.Write([]byte("data"))
				return bw
			}(),
			expectedContentType: "custom/binary",
		},
		{
			name: "FormUrlEncoded sets content type",
			setupRequest: func(r *Request) *Request {
				return r
			},
			body: func() FormUrlEncodedWriter {
				fw := NewFormUrlEncodedWriter()
				fw.Set("key", "value")
				return fw
			}(),
			expectedContentType: "application/x-www-form-urlencoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contentType := r.Header.Get("Content-Type")

				if tt.expectedContentType != "" {
					mediaType, _, _ := mime.ParseMediaType(contentType)
					expectedType, _, _ := mime.ParseMediaType(tt.expectedContentType)

					if mediaType != expectedType {
						t.Errorf("expected content type '%s', got '%s'", tt.expectedContentType, contentType)
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			req := R(context.Background())
			req = tt.setupRequest(req)

			_, err := req.POST(server.URL, tt.body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
