package restapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
)

type IRoutes interface {
	RegisterRoutes(e *echo.Echo) error
}

type HTTPConfig struct {
	BuildCommitID string       `validate:"-"`
	BuildTime     time.Time    `validate:"-"`
	StartupTime   time.Time    `validate:"required"`
	Tracer        trace.Tracer `validate:"required"`
}

type HTTP struct {
	config HTTPConfig
	echo   *echo.Echo
}

var _ http.Handler = (*HTTP)(nil)

func NewHTTP(cfg HTTPConfig) (*HTTP, error) {
	err := validator.Validate(cfg)
	if err != nil {
		err = fmt.Errorf("http server config error: %w", err)
		return nil, err
	}

	e := echo.New()
	e.Use(
		middleware.RemoveTrailingSlash(),
		middleware.CORS(),
	)

	e.HTTPErrorHandler = func(err error, eCtx echo.Context) {
		httpStatus := http.StatusUnprocessableEntity

		var errHTTP *echo.HTTPError
		if errors.As(err, &errHTTP) {
			httpStatus = errHTTP.Code
		}

		var errBinding *echo.BindingError
		if errors.As(err, &errBinding) {
			httpStatus = errBinding.Code
		}

		if httpStatus <= 0 || httpStatus >= 599 {
			httpStatus = http.StatusInternalServerError
		}

		_err := eCtx.JSON(httpStatus, respbuilder.Error(respbuilder.ErrGeneral, err))
		if _err != nil {
			_err = fmt.Errorf("echo.HTTPErrorHandler panic: %w", _err)
			panic(_err)
		}
	}

	// Prepare all handler
	handlerSystem, err := handlersystem.New(
		handlersystem.WithBuildCommitID(cfg.BuildCommitID),
		handlersystem.WithBuildTime(cfg.BuildTime),
		handlersystem.WithStartupTime(cfg.StartupTime),
		handlersystem.WithTracer(cfg.Tracer),
	)
	if err != nil {
		return nil, err
	}

	// Register all routes
	e.GET("/ping", handlerSystem.Ping)
	e.GET("/system-info", handlerSystem.SystemInfo)

	h := &HTTP{
		config: cfg,
		echo:   e,
	}

	return h, nil
}

func (h *HTTP) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.echo.ServeHTTP(writer, request)
}
