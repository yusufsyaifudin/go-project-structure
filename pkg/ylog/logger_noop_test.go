package ylog_test

import (
	"context"
	"testing"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

func TestNewNoop(t *testing.T) {
	ctx := context.TODO()
	msg := "message log"

	logger := ylog.NewNoop()
	logger.Debug(ctx, msg)
	logger.Info(ctx, msg)
	logger.Warn(ctx, msg)
	logger.Error(ctx, msg)
	logger.Panic(ctx, msg)
	logger.Fatal(ctx, msg)
	logger.Access(ctx, msg, ylog.AccessLogData{})
}
