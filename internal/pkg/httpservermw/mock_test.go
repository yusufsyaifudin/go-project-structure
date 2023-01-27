package httpservermw_test

import (
	"io"
	"net/http"
)

type mockHandler struct {
	responseCode   int
	responseHeader map[string]string
	responseBody   string
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	for k, v := range m.responseHeader {
		w.Header().Set(k, v)
	}

	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))

}

type buf struct {
	err error
}

var _ io.Reader = (*buf)(nil)

func newBuf(err error) io.Reader {
	return &buf{
		err: err,
	}
}

func (b *buf) Read(p []byte) (n int, err error) {
	if b.err != nil {
		return 0, b.err
	}

	return len(p), nil
}

type closer struct {
	buf io.Reader
	err error
}

var _ io.ReadCloser = (*closer)(nil)

func newCloser(buf io.Reader, err error) *closer {
	return &closer{
		buf: buf,
		err: err,
	}
}

func (c *closer) Read(p []byte) (n int, err error) {
	return c.buf.Read(p)
}

func (c *closer) Close() error {
	return c.err
}

type mockRespWriter struct {
	header http.Header
	write  func(b []byte) (int, error)
	code   int
}

var _ http.ResponseWriter = (*mockRespWriter)(nil)

func (m *mockRespWriter) Header() http.Header {
	return m.header
}

func (m *mockRespWriter) Write(i []byte) (int, error) {
	return m.write(i)
}

func (m *mockRespWriter) WriteHeader(_ int) {}
