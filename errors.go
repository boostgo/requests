package requests

import "github.com/boostgo/errorx"

var (
	ErrParseResponseBody           = errorx.New("response.parse_body")
	ErrExportResponseMustBePointer = errorx.New("response.export_must_be_pointer")
	ErrContextCanceledAndHasError  = errorx.New("context.canceled_and_has_error")
	ErrContextCanceled             = errorx.New("context.canceled")

	ErrBytesWriterWrite      = errorx.New("bytes_writer.write")
	ErrFormDataWriterAdd     = errorx.New("formdata_writer.add")
	ErrFormDataWriterAddFile = errorx.New("formdata_writer.add_file")
	ErrFormDataWriterSet     = errorx.New("formdata_writer.set")
	ErrFormDataWriterClose   = errorx.New("formdata_writer.close")

	ErrRequestRetryDo = errorx.New("request.retry_do")
)

type responseBodyContext struct {
	URL  string `json:"url"`
	Code int    `json:"code"`
	Blob []byte `json:"blob"`
}

func newParseResponseBodyError(url string, code int, blob []byte) *errorx.Error {
	return ErrParseResponseBody.SetData(responseBodyContext{
		URL:  url,
		Code: code,
		Blob: blob,
	})
}
