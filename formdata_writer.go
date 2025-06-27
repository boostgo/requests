package requests

import (
	"bytes"
	"io"
	"mime/multipart"

	"github.com/boostgo/convert"
	"github.com/boostgo/errorx"
)

// FormDataWriter uses for sending form-data request body
type FormDataWriter interface {
	Add(key string, value any) error
	AddFile(name, fileName string, file []byte) error
	Set(data map[string]any) error
	Boundary() string
	ContentType() string
	Buffer() *bytes.Buffer
	Close() error
}

type formData struct {
	body   bytes.Buffer
	writer *multipart.Writer
}

// NewFormData creates FormDataWriter
//
//nolint:errcheck
func NewFormData(initial ...map[string]any) FormDataWriter {
	fd := &formData{
		body: bytes.Buffer{},
	}

	fd.writer = multipart.NewWriter(&fd.body)

	if len(initial) > 0 {
		fd.Set(initial[0])
	}

	return fd
}

func (fd *formData) Add(key string, value any) error {
	if err := fd.writer.WriteField(key, convert.String(value)); err != nil {
		return ErrFormDataWriterAdd.SetError(err)
	}

	return nil
}

func (fd *formData) AddFile(name, fileName string, file []byte) (err error) {
	defer func() {
		if err != nil {
			err = errorx.Wrap(err, ErrFormDataWriterAddFile)
		}
	}()

	fileWriter, err := fd.writer.CreateFormFile(name, fileName)
	if err != nil {
		return err
	}

	_, err = io.Copy(fileWriter, bytes.NewReader(file))
	if err != nil {
		return err
	}

	return nil
}

func (fd *formData) Set(data map[string]any) (err error) {
	defer func() {
		if err != nil {
			err = errorx.Wrap(err, ErrFormDataWriterSet)
		}
	}()

	if data == nil || len(data) == 0 {
		return nil
	}

	for key, value := range data {
		if err = fd.Add(key, value); err != nil {
			return err
		}
	}

	return nil
}

func (fd *formData) Boundary() string {
	return fd.writer.Boundary()
}

func (fd *formData) ContentType() string {
	return fd.writer.FormDataContentType()
}

func (fd *formData) Buffer() *bytes.Buffer {
	return &fd.body
}

func (fd *formData) Close() error {
	if err := fd.writer.Close(); err != nil {
		return ErrFormDataWriterClose.SetError(err)
	}

	return nil
}
