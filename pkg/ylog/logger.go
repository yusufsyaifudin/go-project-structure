package ylog

import (
	"context"
)

type LogType string

const (
	TypeAccessLog LogType = "access_log"
	TypeSys       LogType = "sys"
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

type Noop struct{}

var _ Logger = (*Noop)(nil)

func (n *Noop) Debug(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Info(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Warn(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Error(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Panic(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Fatal(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Access(ctx context.Context, msg string, data AccessLogData) {}

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

type KeyValue interface {
	Key() string
	Value() any
}

type kv struct {
	k string
	v any
}

func (k *kv) Key() string {
	return k.k
}

func (k *kv) Value() any {
	return k.v
}

var _ KeyValue = (*kv)(nil)

func KV(k string, v interface{}) KeyValue {
	return &kv{
		k: k,
		v: v,
	}
}
