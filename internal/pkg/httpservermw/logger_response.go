package httpservermw

import (
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
	headers    http.Header
}

var _ http.ResponseWriter = (*responseWriter)(nil)

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
		body:           make([]byte, 0),
		headers:        http.Header{},
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.headers = rw.ResponseWriter.Header().Clone()
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = b
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}
