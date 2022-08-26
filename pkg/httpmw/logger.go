package httpmw

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/yusufsyaifudin/ylog"
	"go.uber.org/multierr"
)

// ContextData propagate some data
type ContextData struct {
	RequestID     string `tracer:"trace_id"`
	CorrelationID string `tracer:"correlation_id"`
}

type LoggerOpt struct {
	SkipPath map[string]struct{}
}

func Logger(opt LoggerOpt) echo.MiddlewareFunc {
	var toSimpleMap = func(h http.Header) map[string]string {
		out := map[string]string{}
		for k, v := range h {
			out[k] = strings.Join(v, " ")
		}

		return out
	}

	fn := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			t0 := time.Now()

			var (
				err        error
				reqBody    []byte
				reqBodyErr error
				reqBodyObj interface{}
			)

			req := c.Request()
			resp := c.Response()

			requestID := strings.TrimSpace(req.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = uuid.NewString()
			}

			propagateData := ContextData{
				RequestID:     requestID,
				CorrelationID: uuid.NewString(),
			}

			tracer, err := ylog.NewTracer(propagateData, ylog.WithTag("tracer"))
			if err != nil {
				err = fmt.Errorf("cannot inject propagation data: %w", err)
				return err
			}

			// add context
			ctx := ylog.Inject(c.Request().Context(), tracer)
			c.SetRequest(c.Request().WithContext(ctx))

			if req.Body != nil {
				reqBody, reqBodyErr = io.ReadAll(req.Body)
				if reqBodyErr != nil {
					err = multierr.Append(err, fmt.Errorf("error read request body: %w", reqBodyErr))
					reqBody = []byte("")
				}

				c.Request().Body = io.NopCloser(bytes.NewReader(reqBody))
			}

			if _err := json.Unmarshal(reqBody, &reqBodyObj); _err == nil {
				reqBody = []byte("")
			}

			// capture response body
			resBody := &bytes.Buffer{}
			mw := io.MultiWriter(resp.Writer, resBody)
			writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: resp.Writer}
			resp.Writer = writer
			resp.Header().Set("X-Request-ID", propagateData.RequestID)
			resp.Header().Set("X-Correlation-ID", propagateData.CorrelationID)

			if _err := next(c); _err != nil {
				err = multierr.Append(err, _err)
				//c.Error(err) // doesn't need call this because if we call this, it will duplicate response
			}

			// if response is larger than 1Mb then don't print all message
			if resBody.Len() >= 1_000_000 {
				resBody.Truncate(0)
				resBody.WriteString("message larger than 1MB to log")
			}

			respBody := resBody.Bytes()
			var respObj interface{}
			if _err := json.Unmarshal(respBody, &respObj); _err == nil {
				respBody = []byte("")
			}

			errStr := ""
			if err != nil {
				errStr = err.Error()
			}

			type HTTPData struct {
				Header     map[string]string `json:"header,omitempty"`
				DataObject interface{}       `json:"data_object,omitempty"`
				DataString string            `json:"data_string,omitempty"`
			}

			type AccessLogData struct {
				Path        string   `json:"path,omitempty"`
				Request     HTTPData `json:"request,omitempty"`
				Response    HTTPData `json:"response,omitempty"`
				Error       string   `json:"error,omitempty"`
				ElapsedTime int64    `json:"elapsed_time,omitempty"`
			}

			// log outgoing request
			logData := AccessLogData{
				Path: c.Path(),
				Request: HTTPData{
					Header:     toSimpleMap(req.Header),
					DataObject: reqBody,
					DataString: string(reqBody),
				},
				Response: HTTPData{
					Header:     toSimpleMap(resp.Header()),
					DataObject: respObj,
					DataString: string(respBody),
				},
				Error:       errStr,
				ElapsedTime: time.Since(t0).Milliseconds(),
			}

			// skip path that does not need to be logged
			if opt.SkipPath != nil {
				_, exist := opt.SkipPath[c.Path()]
				if !exist {
					ylog.Info(ctx, "incoming request", ylog.KV("data", logData))
				}
			}

			return err
		}
	}

	return fn
}

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

var _ io.Writer = (*bodyDumpResponseWriter)(nil)
var _ http.ResponseWriter = (*bodyDumpResponseWriter)(nil)

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
