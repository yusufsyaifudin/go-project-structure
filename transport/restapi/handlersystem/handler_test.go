package handlersystem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
)

func TestNew_NoOptions(t *testing.T) {
	h, err := handlersystem.New()
	require.NoError(t, err)
	assert.NotNil(t, h)
}

func TestNew_WithAllOptions(t *testing.T) {
	now := time.Now()
	h, err := handlersystem.New(
		handlersystem.WithBuildCommitID("abc123"),
		handlersystem.WithBuildTime(now),
		handlersystem.WithStartupTime(now),
	)
	require.NoError(t, err)
	assert.NotNil(t, h)
}

func TestSystemHandler_Router_RegistersRoutes(t *testing.T) {
	h, err := handlersystem.New()
	require.NoError(t, err)

	e := echo.New()
	h.Router(e)

	registered := make(map[string]bool)
	for _, r := range e.Routes() {
		registered[r.Method+":"+r.Path] = true
	}
	assert.True(t, registered["GET:/ping"], "GET /ping should be registered")
	assert.True(t, registered["GET:/system-info"], "GET /system-info should be registered")
}

func TestSystemHandler_Ping_Returns200(t *testing.T) {
	h, err := handlersystem.New(handlersystem.WithBuildCommitID("deadbeef"))
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.Ping(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "0", body["code"])
	assert.NotNil(t, body["data"])
}

func TestSystemHandler_Ping_ResponseContainsCommitHash(t *testing.T) {
	h, err := handlersystem.New(handlersystem.WithBuildCommitID("test-commit"))
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.Ping(c))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "test-commit", data["commit_hash"])
}

func TestSystemHandler_SystemInfo_Returns200(t *testing.T) {
	h, err := handlersystem.New()
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/system-info", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.SystemInfo(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "0", body["code"])
	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["number_go_routine"])
}
