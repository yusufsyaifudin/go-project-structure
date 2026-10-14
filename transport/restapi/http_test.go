package restapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
)

// mockRouter satisfies restapi.EchoRouter for tests.
type mockRouter struct {
	called bool
}

func (m *mockRouter) Router(e *echo.Echo) {
	m.called = true
	e.GET("/test-route", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
}

func TestNewHTTP_DefaultsAlwaysSucceed(t *testing.T) {
	h, err := restapi.NewHTTP()
	require.NoError(t, err)
	assert.NotNil(t, h)
}

func TestNewHTTP_AllOptions(t *testing.T) {
	r := &mockRouter{}
	h, err := restapi.NewHTTP(
		restapi.WithBuildCommitID("sha123"),
		restapi.WithBuildTime(time.Now()),
		restapi.WithStartupTime(time.Now()),
		restapi.WithBaseURL("http://localhost:3001"),
		restapi.AddHandler(r),
	)
	require.NoError(t, err)
	assert.NotNil(t, h)
	assert.True(t, r.called)
}

func TestNewHTTP_AddHandler_Nil_NoError(t *testing.T) {
	h, err := restapi.NewHTTP(restapi.AddHandler(nil))
	require.NoError(t, err)
	assert.NotNil(t, h)
}

func TestHTTP_ServeHTTP_SwaggerRoute(t *testing.T) {
	h, err := restapi.NewHTTP()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	// Swagger UI HTML or redirect — not 0.
	assert.NotEqual(t, 0, rec.Code)
}

func TestHTTP_ServeHTTP_RegisteredHandler(t *testing.T) {
	r := &mockRouter{}
	h, err := restapi.NewHTTP(restapi.AddHandler(r))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test-route", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTP_httpErrorHandler_404(t *testing.T) {
	h, err := restapi.NewHTTP()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-xyzabc", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	// echo's default 404 passes through httpErrorHandler
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHTTP_httpErrorHandler_PlainError422(t *testing.T) {
	// A handler that returns a plain (non-HTTP) error → httpErrorHandler uses 422 default.
	r := &plainErrRouter{}
	h, err := restapi.NewHTTP(restapi.AddHandler(r))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/plain-err", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	// Default httpStatus in httpErrorHandler is 422, which is a valid IANA code.
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// plainErrRouter registers a route that returns a plain error (no echo.HTTPError).
type plainErrRouter struct{}

func (p *plainErrRouter) Router(e *echo.Echo) {
	e.GET("/plain-err", func(c echo.Context) error {
		return assert.AnError // plain error, not *echo.HTTPError
	})
}
