package requests

import (
	"io"
	"net/url"
	"strings"
)

type FormUrlEncodedWriter interface {
	Set(key, value string)
	Get(key string) string
	Delete(key string)
	Has(key string) bool
	Reader() io.Reader
}

type formUrlEncodedWriter struct {
	values url.Values
}

func NewFormUrlEncodedWriter() FormUrlEncodedWriter {
	return &formUrlEncodedWriter{
		values: url.Values{},
	}
}

func (w *formUrlEncodedWriter) Set(key, value string) {
	w.values.Set(key, value)
}

func (w *formUrlEncodedWriter) Get(key string) string {
	return w.values.Get(key)
}

func (w *formUrlEncodedWriter) Delete(key string) {
	w.values.Del(key)
}

func (w *formUrlEncodedWriter) Has(key string) bool {
	return w.values.Has(key)
}

func (w *formUrlEncodedWriter) Reader() io.Reader {
	return strings.NewReader(w.values.Encode())
}
