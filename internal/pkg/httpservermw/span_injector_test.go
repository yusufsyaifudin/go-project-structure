package httpservermw_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
)

func TestSpanInjectorMiddleware_FilterFalse(t *testing.T) {
	// When filter returns false, the middleware injects a NeverSample span and
	// passes the request to the next handler.
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	filter := func(*http.Request) bool { return false }
	h := httpservermw.SpanInjectorMiddleware(next, filter)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSpanInjectorMiddleware_FilterTrue(t *testing.T) {
	// When filter returns true, no synthetic span is added; request passes through.
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	filter := func(*http.Request) bool { return true }
	h := httpservermw.SpanInjectorMiddleware(next, filter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSpanInjectorMiddleware_NilFilter(t *testing.T) {
	// nil filter is treated as "include all" — no span injection.
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	h := httpservermw.SpanInjectorMiddleware(next, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}
