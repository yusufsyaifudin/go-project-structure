package httpservermw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type captureRequestOpt struct {
	T0               time.Time
	Request          *http.Request
	Tracer           trace.Tracer
	SpanStartOptions []trace.SpanStartOption
	Logger           *slog.Logger
}

func captureRequest(parentCtx context.Context, opt *captureRequestOpt) context.Context {
	const reqLogMsg = "capture incoming request payload"
	captReqCtx, captReqSpan := opt.Tracer.Start(parentCtx, reqLogMsg, opt.SpanStartOptions...)
	defer captReqSpan.End()

	req := opt.Request

	// Get common info regarding the source of request
	// ensure request log variable only scoped here
	var (
		reqHeader   = HttpHeaderToSimpleMap(req.Header)
		queryParams url.Values
		reqBodyBuf  = &bytes.Buffer{}
		reqBodyLen  int64
		errCum      error
	)

	if req.Body != nil {
		if reqBodyCopied, _err := io.Copy(reqBodyBuf, req.Body); _err != nil {
			errCum = errors.Join(errCum, fmt.Errorf("error copy request body: %w", _err))
			reqBodyBuf = &bytes.Buffer{}
		} else {
			reqBodyLen = reqBodyCopied
		}

		// Don't forget to close the body buffer.
		if _err := req.Body.Close(); _err != nil {
			errCum = errors.Join(errCum, fmt.Errorf("error closing request body: %w", _err))
		}

		req.Body = io.NopCloser(reqBodyBuf)
	}

	var reqBodyCaptured any
	var reqBodyDecoderErr error
	if len(reqBodyBuf.Bytes()) > 0 {
		reqBodyDecoderErr = json.Unmarshal(reqBodyBuf.Bytes(), &reqBodyCaptured)
	}

	if reqBodyDecoderErr != nil {
		errCum = errors.Join(errCum, fmt.Errorf("error unmarshal request body: %w", reqBodyDecoderErr))

		reqBodyLen = int64(reqBodyBuf.Len())
		reqBodyCaptured = reqBodyBuf.String() // Fallback with the real request body string
	}

	if _err := req.ParseForm(); _err != nil {
		errCum = errors.Join(errCum, fmt.Errorf("error parse request form: %w", _err))
	}

	// Cloned the request, for safety reason (i.e. parse form should not affect the original request)
	reqCloned := req.Clone(captReqCtx)
	reqCloned.Form = req.Form
	reqCloned.Body = io.NopCloser(reqBodyBuf)

	queryParams = reqCloned.Form

	reqURL := req.URL
	if reqURL == nil {
		reqURL = &url.URL{}
	}

	requestLatency := time.Since(opt.T0).Milliseconds()

	requestLog := AccessLog{
		Method:      opt.Request.Method,
		Host:        req.Host,
		Path:        reqURL.Path,
		StatusCode:  0,
		Header:      reqHeader,
		Body:        reqBodyCaptured,
		BodyLen:     reqBodyLen,
		QueryParams: queryParams,
		Error:       "",
		ElapsedTime: requestLatency,
	}

	if errCum != nil {
		requestLog.Error = errCum.Error()
	}

	opt.Logger.InfoContext(captReqCtx, reqLogMsg, slog.Any("request", requestLog))

	return captReqCtx
}
