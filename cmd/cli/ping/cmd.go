package ping

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/cli"
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpclientmw"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

const (
	instrumentationName = "github.com/yusufsyaifudin/go-project-structure/cmd/cli/ping"
)

type Opt func(*CMD) error

func WithTracer(tracer trace.TracerProvider) Opt {
	return func(cmd *CMD) error {
		cmd.tracer = tracer
		return nil
	}
}

func WithLogger(log ylog.Logger) Opt {
	return func(cmd *CMD) error {
		cmd.logger = log
		return nil
	}
}

type CMD struct {
	tracer trace.TracerProvider
	logger ylog.Logger
}

var _ cli.Command = (*CMD)(nil)

func NewCMD(opts ...Opt) (*CMD, error) {
	cmd := &CMD{
		tracer: trace.NewNoopTracerProvider(),
		logger: &ylog.Noop{},
	}

	for _, opt := range opts {
		err := opt(cmd)
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

func (c *CMD) Help() string {
	return "Return the status of server"
}

type Flag struct {
	Server string `required:"true" short:"s" long:"server" description:"URL of the server"`
}

// Run call /ping endpoint of the server.
// Example command: go run cmd/cli/main.go ping -s http://localhost:3001/
func (c *CMD) Run(args []string) int {
	ctx := context.Background()
	tracer := c.tracer.Tracer(instrumentationName)

	ctx, span := tracer.Start(ctx, "Ping CLI Run")
	defer span.End()

	var flag Flag
	_, err := flags.ParseArgs(&flag, args)
	if err != nil {
		err = fmt.Errorf("failed parsing flag: %w", err)
		log.Println(err)
		return 1
	}

	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	transport := http.DefaultTransport
	transport = otelhttp.NewTransport(transport,
		otelhttp.WithTracerProvider(c.tracer),
		otelhttp.WithPropagators(propagator),
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("Request for %s %s", r.Method, r.URL.EscapedPath())
		}),
	)
	transport = httpclientmw.NewHttpRoundTripper(
		httpclientmw.WithBaseRoundTripper(transport),
		httpclientmw.WithLogger(c.logger),
		httpclientmw.WithTracer(c.tracer),
	)

	client := http.DefaultClient
	client.Transport = transport

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(flag.Server, "/")+"/ping", nil)
	if err != nil {
		c.logger.Error(ctx, "cannot prepare request", ylog.KV("error", err))
		return 1
	}

	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error(ctx, "cannot do http client request", ylog.KV("error", err))
		return 1
	}

	defer func() {
		var _err error
		if resp != nil && resp.Body != nil {
			_err = resp.Body.Close()
		}

		if _err != nil {
			c.logger.Error(ctx, "cannot close response body", ylog.KV("error", _err))
		}
	}()

	var respBody struct {
		Data handlersystem.PingResp `json:"data"`
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&respBody)
	if err != nil {
		c.logger.Error(ctx, "cannot decode response body", ylog.KV("error", err))
		return 1
	}

	c.logger.Info(ctx, "response", ylog.KV("response", respBody))

	return 0
}

func (c *CMD) Synopsis() string {
	return "Return the status of server"
}
