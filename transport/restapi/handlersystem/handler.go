package handlersystem

import (
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"
	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
	"github.com/yusufsyaifudin/go-project-structure/transport/restapi"
)

const (
	instrumentationName = "github.com/yusufsyaifudin/go-project-structure/transport/restapi/handlersystem"
)

type Opt func(*SystemHandler) error

func WithBuildCommitID(id string) Opt {
	return func(handler *SystemHandler) error {
		handler.buildCommitID = id
		return nil
	}
}

func WithBuildTime(t time.Time) Opt {
	return func(handler *SystemHandler) error {
		handler.buildTime = t
		return nil
	}
}

func WithStartupTime(t time.Time) Opt {
	return func(handler *SystemHandler) error {
		handler.startupTime = t
		return nil
	}
}

func WithObservability(mgr observability.Observability) Opt {
	return func(handler *SystemHandler) error {
		if mgr == nil {
			return nil
		}
		handler.observability = mgr
		return nil
	}
}

type SystemHandler struct {
	buildCommitID string
	buildTime     time.Time
	startupTime   time.Time
	observability observability.Observability
}

// Ensure SystemHandler implements restapi.EchoRouter to successfully register endpoint to Echo framework.
var _ restapi.EchoRouter = (*SystemHandler)(nil)

func New(opts ...Opt) (*SystemHandler, error) {
	systemHandler := &SystemHandler{
		startupTime: time.Now(),
	}

	for _, opt := range opts {
		err := opt(systemHandler)
		if err != nil {
			return nil, err
		}
	}

	return systemHandler, nil
}

func (s *SystemHandler) Router(e *echo.Echo) {
	e.GET("/ping", s.Ping)
	e.GET("/system-info", s.SystemInfo)
}

type PingResp struct {
	CommitHash   string    `json:"commit_hash,omitempty"`
	BuildTime    time.Time `json:"build_time,omitempty"`
	StartUpTime  time.Time `json:"startup_time,omitempty"`
	UptimeNs     int64     `json:"uptime_ns,omitempty"`
	UptimeString string    `json:"uptime_string,omitempty"`
}

func (s *SystemHandler) Ping(c echo.Context) error {
	ctx := c.Request().Context()

	// Instead of using s.observability.Tracer().Start(ctx, "Ping Handler")
	// Please use SpanFromContext from go.opentelemetry.io/otel/trace
	// This will continue the Span from previous context if exists.
	// Otherwise, it will use noopSpan{} that does nothing.
	//
	// If you still use s.observability.Tracer().Start(ctx, "Ping Handler"),
	// you will get non-consistent tracing where current Span will be pushed to OpenTelemetry agent,
	// but in fact you already disable this route via Filter in main.go server.
	span := trace.SpanFromContext(ctx)
	defer span.End()

	// Get new span and child context from TracerProvider in propagated Span.
	// Name of tracer doesn't need the current package,
	// but, for make it consistent use current package name.
	_, spanChild := span.TracerProvider().Tracer(instrumentationName).Start(ctx, "Ping Handler")
	defer spanChild.End()

	return c.JSON(http.StatusOK, respbuilder.Ok(respbuilder.Success, PingResp{
		CommitHash:   s.buildCommitID,
		BuildTime:    s.buildTime,
		StartUpTime:  s.startupTime,
		UptimeNs:     time.Since(s.startupTime).Nanoseconds(),
		UptimeString: time.Since(s.startupTime).String(),
	}))
}

type SystemInfoRespBySize struct {
	Size    uint32 `json:"size,omitempty"`
	Mallocs uint64 `json:"mallocs,omitempty"`
	Frees   uint64 `json:"frees,omitempty"`
}

type SystemInfoResp struct {
	NumberGoRoutine int `json:"number_go_routine,omitempty"`

	// This is from runtime.MemStats
	Alloc         uint64                 `json:"alloc,omitempty"`
	TotalAlloc    uint64                 `json:"total_alloc,omitempty"`
	Sys           uint64                 `json:"sys,omitempty"`
	Lookups       uint64                 `json:"lookups,omitempty"`
	Mallocs       uint64                 `json:"mallocs,omitempty"`
	Frees         uint64                 `json:"frees,omitempty"`
	HeapAlloc     uint64                 `json:"heap_alloc,omitempty"`
	HeapSys       uint64                 `json:"heap_sys,omitempty"`
	HeapIdle      uint64                 `json:"heap_idle,omitempty"`
	HeapInuse     uint64                 `json:"heap_inuse,omitempty"`
	HeapReleased  uint64                 `json:"heap_released,omitempty"`
	HeapObjects   uint64                 `json:"heap_objects,omitempty"`
	StackInuse    uint64                 `json:"stack_inuse,omitempty"`
	StackSys      uint64                 `json:"stack_sys,omitempty"`
	MSpanInuse    uint64                 `json:"m_span_inuse,omitempty"`
	MSpanSys      uint64                 `json:"m_span_sys,omitempty"`
	MCacheInuse   uint64                 `json:"m_cache_inuse,omitempty"`
	MCacheSys     uint64                 `json:"m_cache_sys,omitempty"`
	BuckHashSys   uint64                 `json:"buck_hash_sys,omitempty"`
	GCSys         uint64                 `json:"gc_sys,omitempty"`
	OtherSys      uint64                 `json:"other_sys,omitempty"`
	NextGC        uint64                 `json:"next_gc,omitempty"`
	LastGC        uint64                 `json:"last_gc,omitempty"`
	PauseTotalNs  uint64                 `json:"pause_total_ns,omitempty"`
	PauseNs       [256]uint64            `json:"pause_ns,omitempty"`
	PauseEnd      [256]uint64            `json:"pause_end,omitempty"`
	NumGC         uint32                 `json:"num_gc,omitempty"`
	NumForcedGC   uint32                 `json:"num_forced_gc,omitempty"`
	GCCPUFraction float64                `json:"gccpu_fraction,omitempty"`
	EnableGC      bool                   `json:"enable_gc,omitempty"`
	DebugGC       bool                   `json:"debug_gc,omitempty"`
	BySize        []SystemInfoRespBySize `json:"by_size,omitempty"`
}

func (s *SystemHandler) SystemInfo(c echo.Context) error {
	ctx := c.Request().Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()

	_, spanChild := span.TracerProvider().Tracer(instrumentationName).Start(ctx, "System Info Handler")
	defer spanChild.End()

	var bToMb = func(b uint64) uint64 {
		return b / 1024 / 1024
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	bySize := make([]SystemInfoRespBySize, 0)
	for _, size := range m.BySize {
		bySize = append(bySize, SystemInfoRespBySize{
			Size:    size.Size,
			Mallocs: size.Mallocs,
			Frees:   size.Frees,
		})
	}

	resp := SystemInfoResp{
		NumberGoRoutine: runtime.NumGoroutine(),

		Alloc:         bToMb(m.Alloc),
		TotalAlloc:    bToMb(m.TotalAlloc),
		Sys:           bToMb(m.Sys),
		Lookups:       m.Lookups,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		HeapAlloc:     bToMb(m.HeapAlloc),
		HeapSys:       bToMb(m.HeapSys),
		HeapIdle:      bToMb(m.HeapIdle),
		HeapInuse:     bToMb(m.HeapInuse),
		HeapReleased:  bToMb(m.HeapReleased),
		HeapObjects:   m.HeapObjects,
		StackInuse:    bToMb(m.StackInuse),
		StackSys:      bToMb(m.StackSys),
		MSpanInuse:    bToMb(m.MSpanInuse),
		MSpanSys:      bToMb(m.MSpanSys),
		MCacheInuse:   bToMb(m.MCacheInuse),
		MCacheSys:     bToMb(m.MCacheSys),
		BuckHashSys:   bToMb(m.BuckHashSys),
		GCSys:         bToMb(m.GCSys),
		OtherSys:      bToMb(m.OtherSys),
		NextGC:        m.NextGC,
		LastGC:        m.LastGC,
		PauseTotalNs:  m.PauseTotalNs,
		PauseNs:       m.PauseNs,
		PauseEnd:      m.PauseEnd,
		NumGC:         m.NumGC,
		NumForcedGC:   m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
		EnableGC:      m.EnableGC,
		DebugGC:       m.DebugGC,
		BySize:        bySize,
	}

	return c.JSON(http.StatusOK, respbuilder.Ok(respbuilder.Success, resp))
}
