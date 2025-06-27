package requests

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/boostgo/convert"
	"github.com/boostgo/errorx"
	"github.com/boostgo/httpx"
	"github.com/boostgo/reflectx"
)

type RequestOption func(request *http.Request)

type Request struct {
	ctx     context.Context
	baseURL string
	client  *http.Client
	export  any

	queryVariables map[string]any
	headers        map[string]any
	cookies        map[string]any

	retryCount int
	retryWait  time.Duration

	timeout time.Duration

	basic       basicAuth
	bearerToken string

	options []RequestOption

	req      *http.Request
	resp     *http.Response
	response *Response

	mx sync.Locker
}

// R creates Request object with context
func R(ctx context.Context) *Request {
	if ctx == nil {
		ctx = context.Background()
	}

	return &Request{
		ctx: ctx,

		queryVariables: make(map[string]any),
		headers:        make(map[string]any),
		cookies:        make(map[string]any),

		retryCount: 1,
		retryWait:  time.Millisecond * 100,

		options: make([]RequestOption, 0),

		mx: &sync.Mutex{},
	}
}

func (request *Request) setBaseURL(baseURL string) *Request {
	request.baseURL = baseURL
	return request
}

// Options sets option functions which can modify created request.
func (request *Request) Options(opts ...RequestOption) *Request {
	if len(opts) == 0 {
		return request
	}

	request.options = opts
	return request
}

// Client set default http client for current Request object.
func (request *Request) Client(client *http.Client) *Request {
	if client == nil {
		return request
	}

	request.client = client
	return request
}

// RetryCount sets count of retries need.
//
// By default, retry count is 1
func (request *Request) RetryCount(count int) *Request {
	if count <= 1 {
		return request
	}

	request.retryCount = count
	return request
}

// RetryWait sets wait time between retry requests.
//
// Default wait time is 100ms
func (request *Request) RetryWait(wait time.Duration) *Request {
	if wait <= 0 {
		return request
	}

	request.retryWait = wait
	return request
}

// Timeout sets timeout for waiting for request.
//
// By default, there is no timeout
func (request *Request) Timeout(timeout time.Duration) *Request {
	if timeout <= 0 {
		return request
	}

	request.timeout = timeout
	return request
}

// BasicAuth sets username & password for basic auth mechanism
func (request *Request) BasicAuth(username, password string) *Request {
	if username == "" {
		return request
	}

	request.basic = initBasicAuth(username, password)
	return request
}

// BearerToken sets token for "Authorization" header.
//
// Prefix "Bearer " sets automatically
func (request *Request) BearerToken(token string) *Request {
	request.bearerToken = token
	return request
}

// Query add new query param to current request.
func (request *Request) Query(key string, value any) *Request {
	request.queryVariables[key] = value
	return request
}

// Queries sets query params to request.
//
// Existing keys will be rewritten
func (request *Request) Queries(queries map[string]any) *Request {
	for key, value := range queries {
		request.queryVariables[key] = value
	}
	return request
}

// Result got response bytes slice will try convert to provided export object.
//
// Export object must be pointer
func (request *Request) Result(export any) *Request {
	if !reflectx.IsPointer(export) {
		return request
	}

	request.export = export
	return request
}

// Header set one more key-value pair to headers.
//
// If key already exist it rewrites existing key value
func (request *Request) Header(key string, value any) *Request {
	request.headers[key] = value
	return request
}

// Headers sets map of key-value pairs.
//
// Existing keys will be rewritten
func (request *Request) Headers(headers map[string]any) *Request {
	for key, value := range headers {
		request.headers[key] = value
	}
	return request
}

// Authorization sets authorization token for "Authorization" header.
//
// Prefix "Bearer " will set automatically
func (request *Request) Authorization(token string) *Request {
	request.headers["Authorization"] = "Bearer " + token
	return request
}

// ContentType sets header value to "Content-Type" header.
func (request *Request) ContentType(contentType string) *Request {
	request.headers["Content-Type"] = contentType
	return request
}

// Cookie sets new cookie to request.
//
// Existing key will be rewritten
func (request *Request) Cookie(key string, value any) *Request {
	request.cookies[key] = value
	return request
}

// Cookies sets new cookies map to request.
//
// Existing keys will be rewritten
func (request *Request) Cookies(cookies map[string]any) *Request {
	for key, value := range cookies {
		request.cookies[key] = value
	}
	return request
}

// Do execute request with the provided method and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) Do(method, url string, body ...any) (*Response, error) {
	return request.retryDo(method, url, body...)
}

// GET execute request with method "GET" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) GET(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodGet, url, body...)
}

// POST execute request with method "POST" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) POST(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodPost, url, body...)
}

// PUT execute request with method "PUT" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) PUT(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodPut, url, body...)
}

// PATCH execute request with method "PATCH" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) PATCH(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodPatch, url, body...)
}

// DELETE execute request with method "DELETE" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) DELETE(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodDelete, url, body...)
}

// OPTIONS execute request with method "OPTIONS" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) OPTIONS(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodOptions, url, body...)
}

// HEAD execute request with method "HEAD" and returns Response object.
//
// url - if base url set concat baseURL + url.
//
// body - request body. If provide body as FormDataWriter interface - will be used form-data body. Optional
func (request *Request) HEAD(url string, body ...any) (*Response, error) {
	return request.retryDo(http.MethodHead, url, body...)
}

// initRequest creates http.Request object and if it already created will return cache
func (request *Request) initRequest(method, url string, body ...any) error {
	if request.req != nil {
		return nil
	}

	// building request path (URL)
	var err error
	var fullURL string
	if request.baseURL != "" {
		fullURL = request.baseURL + url
	} else {
		fullURL = url
	}

	// creating request
	if len(body) > 0 && body[0] != nil {
		if formDataWriter, isFormData := body[0].(FormDataWriter); isFormData {
			request.Header("Content-Type", formDataWriter.ContentType())
			if err = formDataWriter.Close(); err != nil {
				return err
			}
			request.req, err = http.NewRequest(method, fullURL, formDataWriter.Buffer())
		} else if bytesWriter, isBytes := body[0].(BytesWriter); isBytes {
			request.Header("Content-Type", bytesWriter.ContentType())
			request.req, err = http.NewRequest(method, fullURL, bytesWriter.Reader())
		} else if formEncodedWriter, isFormEncodedWriter := body[0].(FormUrlEncodedWriter); isFormEncodedWriter {
			request.Header("Content-Type", httpx.ContentTypeForm)
			request.req, err = http.NewRequest(method, fullURL, formEncodedWriter.Reader())
		} else {
			var bodyBlob []byte
			bodyBlob, err = json.Marshal(body[0])
			if err != nil {
				return err
			}

			request.req, err = http.NewRequest(method, fullURL, bytes.NewReader(bodyBlob))
		}
	} else {
		request.req, err = http.NewRequest(method, fullURL, nil)
	}
	if err != nil {
		return err
	}

	// query variables
	query := request.req.URL.Query()
	for key, value := range request.queryVariables {
		query.Set(key, convert.String(value))
	}
	request.req.URL.RawQuery = query.Encode()

	// auth
	request.initAuth()

	// headers
	for key, value := range request.headers {
		request.req.Header.Set(key, convert.String(value))
	}

	// cookies
	for key, value := range request.cookies {
		request.req.AddCookie(&http.Cookie{Name: key, Value: convert.String(value)})
	}

	// options
	for _, opt := range request.options {
		opt(request.req)
	}

	return nil
}

func (request *Request) retryDo(method, url string, body ...any) (_ *Response, err error) {
	defer func() {
		if err != nil {
			err = errorx.Wrap(err, ErrRequestRetryDo, map[string]any{
				"method": method,
				"url":    url,
				"body":   len(body) > 0 && body[0] != nil,
			})
		}
	}()

	// block sending request
	request.mx.Lock()
	defer request.mx.Unlock()

	// context is done
	select {
	case <-request.ctx.Done():
		if err = request.ctx.Err(); err != nil {
			return nil, ErrContextCanceledAndHasError
		}

		return nil, ErrContextCanceled
	default:
	}

	// context has error
	if request.ctx.Err() != nil {
		return nil, request.ctx.Err()
	}

	// set context timeout if provided
	if request.timeout > 0 {
		var cancel context.CancelFunc
		request.ctx, cancel = context.WithTimeout(context.Background(), request.timeout)
		defer cancel()
	}

	// run request with retries
	for i := 0; i < request.retryCount; i++ {
		isLast := i == request.retryCount-1

		if err = request.do(method, url, body...); err != nil {
			if isLast {
				return nil, err
			}

			time.Sleep(request.retryWait)
			continue
		}

		// break retry cycle if request was success
		break
	}

	return request.response, nil
}

func (request *Request) do(method, url string, body ...any) error {
	var err error

	// create request object, if request does again it uses cached request (created before)
	if err = request.initRequest(method, url, body...); err != nil {
		return err
	}

	// do request
	request.resp, err = request.getClient().Do(request.req)
	if err != nil {
		return err
	}
	defer func() {
		if err = request.resp.Body.Close(); err != nil {
			// todo: logger "Close response body"
		}
	}()

	// build *web.Response object
	request.response = newResponse(request, request.resp)

	// parse response body
	var respBlob []byte
	respBlob, err = io.ReadAll(request.resp.Body)
	if err != nil {
		return err
	}
	request.response.bodyBlob = respBlob

	// try to parse response body and set to export
	if request.export != nil {
		if err = request.response.Parse(request.export); err != nil {
			// todo: logger "Parse response export error"
		}
	}

	return nil
}

//nolint:gosec
func (request *Request) getClient() *http.Client {
	if request.client != nil {
		// return provided client
		return request.client
	}

	// return default client
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func (request *Request) initAuth() {
	if request.basic != (basicAuth{}) && request.basic.username != "" {
		request.req.SetBasicAuth(request.basic.username, request.basic.password)
		return
	}

	if request.bearerToken != "" {
		request.req.Header.Set("Authorization", "Bearer "+request.bearerToken)
	}
}
