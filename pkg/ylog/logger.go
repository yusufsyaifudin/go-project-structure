package ylog

import (
	"context"
)

type logType string

const (
	typeAccessLog logType = "access_log"
	typeSys       logType = "sys"
)

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...KeyValue)
	Info(ctx context.Context, msg string, fields ...KeyValue)
	Warn(ctx context.Context, msg string, fields ...KeyValue)
	Error(ctx context.Context, msg string, fields ...KeyValue)
	Panic(ctx context.Context, msg string, fields ...KeyValue)
	Fatal(ctx context.Context, msg string, fields ...KeyValue)
	Access(ctx context.Context, msg string, data AccessLogData)
}

type KeyValue interface {
	Key() string
	Value() any
}

type HTTPData struct {
	StatusCode int               `json:"statusCode,omitempty"`
	Header     map[string]string `json:"header,omitempty"`
	Body       interface{}       `json:"body,omitempty"`
}

type AccessLogData struct {
	Method      string    `json:"method,omitempty"`
	Host        string    `json:"host,omitempty"`
	Path        string    `json:"path,omitempty"`
	Request     *HTTPData `json:"request,omitempty"`
	Response    *HTTPData `json:"response,omitempty"`
	Error       string    `json:"error,omitempty"`
	ElapsedTime int64     `json:"elapsedTime,omitempty"`
}
