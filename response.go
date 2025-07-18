package requests

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/boostgo/httpx"
	"github.com/boostgo/reflectx"
)

type Response struct {
	request  *Request
	raw      *http.Response
	bodyBlob []byte
}

func newResponse(request *Request, resp *http.Response) *Response {
	return &Response{
		request: request,
		raw:     resp,
	}
}

func (response *Response) Raw() *http.Response {
	return response.raw
}

func (response *Response) Status() string {
	return response.raw.Status
}

func (response *Response) StatusCode() int {
	return response.raw.StatusCode
}

func (response *Response) BodyRaw() []byte {
	return response.bodyBlob
}

func (response *Response) Parse(export any) error {
	if response.bodyBlob == nil {
		return nil
	}

	if err := reflectx.CheckExport(export); err != nil {
		return err
	}

	if err := json.Unmarshal(response.bodyBlob, export); err != nil {
		return newParseResponseBodyError(response.request.req.RequestURI, response.raw.StatusCode, response.bodyBlob)
	}

	return nil
}

func (response *Response) Context(ctx context.Context) context.Context {
	return ctx
}

func (response *Response) ContentType() string {
	return response.raw.Header.Get("Content-Type")
}

func (response *Response) IsFailure() bool {
	return httpx.IsFailureCode(response.StatusCode())
}
