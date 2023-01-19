package httpservermw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"go.uber.org/multierr"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type Opt func(*logMiddleware) error

func WithMessage(msg string) Opt {
	return func(tripper *logMiddleware) error {
		if msg == "" {
			return nil
		}

		tripper.msg = msg
		return nil
	}
}

func WithLogger(logger ylog.Logger) Opt {
	return func(tripper *logMiddleware) error {
		if logger == nil {
			tripper.logger = &ylog.Noop{}
			return nil
		}

		tripper.logger = logger
		return nil
	}
}

type logMiddleware struct {
	msg    string
	logger ylog.Logger
}

// LoggingMiddleware is a middleware that logs incoming requests
func LoggingMiddleware(next http.Handler, opts ...Opt) http.Handler {
	l := &logMiddleware{
		msg:    "request logger",
		logger: &ylog.Noop{},
	}

	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			panic(err)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t0 := time.Now()

		reqCtx := context.Background()
		if req != nil && req.Context() != nil {
			reqCtx = req.Context()
		}

		accessLog := ylog.AccessLogData{}
		defer func() {
			accessLog.ElapsedTime = time.Since(t0).Nanoseconds()

			// write log here
			l.logger.Access(reqCtx, l.msg, accessLog)
		}()

		var (
			errCum error // final cumulative error

			reqBodyBuf      = &bytes.Buffer{}
			reqBodyErr      error
			reqBodyCaptured interface{}
		)

		if req != nil && req.Body != nil {
			_, reqBodyErr = io.Copy(reqBodyBuf, req.Body)
			if reqBodyErr != nil {
				errCum = multierr.Append(errCum, fmt.Errorf("error copy request body: %w", reqBodyErr))
				reqBodyBuf = &bytes.Buffer{}
			}

			if _err := req.Body.Close(); _err != nil {
				errCum = multierr.Append(errCum, fmt.Errorf("error closing request body: %w", _err))
			}

			req.Body = io.NopCloser(reqBodyBuf)
		}

		// use json.Unmarshal instead of json.NewDecoder to make sure we can re-read the buffer
		if _err := json.Unmarshal(reqBodyBuf.Bytes(), &reqBodyCaptured); _err != nil && reqBodyBuf.Len() > 0 {
			reqBodyCaptured = reqBodyBuf.String()
		}

		// append to map only when the http.Request is not nil
		if req != nil {
			accessLog.Method = req.Method
			accessLog.Host = req.URL.Host
			accessLog.Path = req.URL.EscapedPath()
			accessLog.Request = &ylog.HTTPData{
				Header: toSimpleMap(req.Header),
				Body:   reqBodyCaptured,
			}
		}

		// Pass the request to the next handler
		respRec := httptest.NewRecorder()
		next.ServeHTTP(respRec, req)

		var (
			respBodyCaptured interface{}
			respBodyBuf      = &bytes.Buffer{}
		)

		if respRec.Result() != nil && respRec.Body != nil {
			respBodyBuf = respRec.Body
		}

		// use json.Unmarshal instead of json.NewDecoder to make sure we can re-read the buffer
		if _err := json.Unmarshal(respBodyBuf.Bytes(), &respBodyCaptured); _err != nil && respBodyBuf.Len() > 0 {
			respBodyCaptured = respBodyBuf.String()
		}

		// append to map only when the http.Response is not nil
		httpStatusCode := http.StatusInternalServerError
		if respRec.Result() != nil {
			httpStatusCode = respRec.Result().StatusCode
			accessLog.Response = &ylog.HTTPData{
				StatusCode: respRec.Result().StatusCode,
				Header:     toSimpleMap(respRec.Result().Header),
				Body:       respBodyCaptured,
			}
		}

		// write to actual
		for k, v := range respRec.Header() {
			w.Header().Set(k, strings.Join(v, " "))
		}

		w.WriteHeader(httpStatusCode)
		if _, _err := w.Write(respBodyBuf.Bytes()); _err != nil {
			errCum = multierr.Append(errCum, fmt.Errorf("failed to write to actual response writer: %w", _err))
		}

		// append error if any
		if errCum != nil {
			accessLog.Error = errCum.Error()
		}
	})
}

func toSimpleMap(h http.Header) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		out[k] = strings.Join(v, " ")
	}

	return out
}
