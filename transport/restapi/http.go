package restapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
)

type HTTPConfig func(*HTTP) error

// WithBuildCommitID add build commit id info
func WithBuildCommitID(hash string) HTTPConfig {
	return func(h *HTTP) error {
		h.buildCommitID = hash
		return nil
	}
}

// WithBuildTime add build time info
func WithBuildTime(t time.Time) HTTPConfig {
	return func(h *HTTP) error {
		h.buildTime = t
		return nil
	}
}

// WithStartupTime add startup time info
func WithStartupTime(t time.Time) HTTPConfig {
	return func(h *HTTP) error {
		h.startupTime = t
		return nil
	}
}

// WithObservability add observability manager
func WithObservability(o observability.Observability) HTTPConfig {
	return func(h *HTTP) error {
		if o == nil {
			return nil
		}

		h.observability = o
		return nil
	}
}

// AddHandler register the handler that implements EchoRouter.
func AddHandler(r EchoRouter) HTTPConfig {
	return func(h *HTTP) error {
		if r == nil {
			return nil
		}

		h.handlers = append(h.handlers, r)
		return nil
	}
}

// EchoRouter is contract to register echo router.
// All router added into Router will be added.
// Please keep in mind that if you register the same path into echo instance,
// the latest registered path will be used.
type EchoRouter interface {
	Router(e *echo.Echo)
}

type HTTP struct {
	buildCommitID string                      `validate:"-"`
	buildTime     time.Time                   `validate:"-"`
	startupTime   time.Time                   `validate:"required"`
	observability observability.Observability `validate:"required"`
	handlers      []EchoRouter

	echo *echo.Echo
}

var _ http.Handler = (*HTTP)(nil)

// NewHTTP implements http.Handler using echo as router.
func NewHTTP(configs ...HTTPConfig) (*HTTP, error) {
	e := echo.New()
	e.Use(
		middleware.RemoveTrailingSlash(),
		middleware.CORS(),
	)

	h := &HTTP{
		buildCommitID: "not-exist",
		buildTime:     time.Now(),
		startupTime:   time.Now(),
		observability: observability.NewNoop(),
		handlers:      make([]EchoRouter, 0),
		echo:          e,
	}

	for _, cfg := range configs {
		err := cfg(h)
		if err != nil {
			return nil, err
		}
	}

	err := validator.Validate(h)
	if err != nil {
		err = fmt.Errorf("http server config error: %w", err)
		return nil, err
	}

	e.HTTPErrorHandler = h.httpErrorHandler

	// register all handler
	for _, handler := range h.handlers {
		handler.Router(e)
	}

	return h, nil
}

func (h *HTTP) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.echo.ServeHTTP(writer, request)
}

func (h *HTTP) httpErrorHandler(err error, eCtx echo.Context) {
	ctx := eCtx.Request().Context()

	httpStatus := http.StatusUnprocessableEntity

	var errHTTP *echo.HTTPError
	if errors.As(err, &errHTTP) {
		httpStatus = errHTTP.Code
	}

	var errBinding *echo.BindingError
	if errors.As(err, &errBinding) {
		httpStatus = errBinding.Code
	}

	// if HTTP status codes not registered in IANA, then use default 500 code
	if http.StatusText(httpStatus) == "" {
		httpStatus = http.StatusInternalServerError
	}

	_err := eCtx.JSON(httpStatus, respbuilder.Error(respbuilder.ErrGeneral, err))
	if _err != nil {
		h.observability.Logger().Error(ctx, "echo.HTTPErrorHandler write json error", ylog.KV("error", _err))
	}
}
