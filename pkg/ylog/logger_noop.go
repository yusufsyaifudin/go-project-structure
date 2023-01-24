package ylog

import "context"

type noop struct{}

func NewNoop() *noop {
	return &noop{}
}

var _ Logger = (*noop)(nil)

func (n *noop) Debug(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *noop) Info(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *noop) Warn(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *noop) Error(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *noop) Panic(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *noop) Fatal(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *noop) Access(ctx context.Context, msg string, data AccessLogData) {}
