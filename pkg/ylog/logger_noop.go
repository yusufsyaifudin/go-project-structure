package ylog

import "context"

type Noop struct{}

func NewNoop() *Noop {
	return &Noop{}
}

var _ Logger = (*Noop)(nil)

func (n *Noop) Debug(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Info(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Warn(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Error(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Panic(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Fatal(ctx context.Context, msg string, fields ...KeyValue) {}

func (n *Noop) Access(ctx context.Context, msg string, data AccessLogData) {}
