package requests

import (
	"bytes"
	"io"
)

// BytesWriter request body parser implementation.
//
// Provide any type of request body and write is as bytes.
//
// By default, content type is "application/octet-stream". But possible to provide custom content type
type BytesWriter interface {
	io.Writer
	ContentType() string
	Bytes() []byte
	SetContentType(contentType string) BytesWriter
	Reader() io.Reader
}

type bytesBuffer struct {
	buffer      *bytes.Buffer
	contentType string
}

// NewBytesWriter creates BytesWriter
func NewBytesWriter() BytesWriter {
	const (
		defaultContentType = "application/octet-stream"
	)

	return &bytesBuffer{
		buffer:      bytes.NewBuffer(make([]byte, 0)),
		contentType: defaultContentType,
	}
}

func (writer *bytesBuffer) Write(bytes []byte) (int, error) {
	n, err := writer.buffer.Write(bytes)
	if err != nil {
		return 0, ErrBytesWriterWrite.SetError(err)
	}

	return n, nil
}

func (writer *bytesBuffer) ContentType() string {
	return writer.contentType
}

func (writer *bytesBuffer) SetContentType(contentType string) BytesWriter {
	writer.contentType = contentType
	return writer
}

func (writer *bytesBuffer) Bytes() []byte {
	return writer.buffer.Bytes()
}

func (writer *bytesBuffer) Reader() io.Reader {
	return writer.buffer
}
