package restapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
)

type HTTPConfig struct {
	BuildCommitID string    `validate:"-"`
	BuildTime     time.Time `validate:"-"`
	StartupTime   time.Time `validate:"required"`
}

type HTTP struct {
	Config     HTTPConfig
	EchoServer http.Handler
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

	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, respbuilder.Ok(respbuilder.Success, map[string]interface{}{
			"ok":          true,
			"commit_hash": cfg.BuildCommitID,
			"build_time":  cfg.BuildTime,
			"start_up":    cfg.StartupTime,
			"uptime_ns":   time.Since(cfg.StartupTime).Nanoseconds(),
			"uptime_str":  time.Since(cfg.StartupTime).String(),
		}))
	})

	return &HTTP{
		Config:     cfg,
		EchoServer: e,
	}, nil
}

func (h *HTTP) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.EchoServer.ServeHTTP(writer, request)
}
