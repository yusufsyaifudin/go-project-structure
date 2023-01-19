package handlersystem

import (
	"net/http"
	"runtime"
	"time"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"

	"github.com/labstack/echo/v4"
	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
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

type PingResp struct {
	CommitHash   string    `json:"commit_hash,omitempty"`
	BuildTime    time.Time `json:"build_time,omitempty"`
	StartUpTime  time.Time `json:"startup_time,omitempty"`
	UptimeNs     int64     `json:"uptime_ns,omitempty"`
	UptimeString string    `json:"uptime_string,omitempty"`
}

func (s *SystemHandler) Ping(c echo.Context) error {
	ctx := c.Request().Context()
	_, span := s.observability.Tracer().Start(ctx, "Ping Handler")
	defer span.End()

	return c.JSON(http.StatusOK, respbuilder.Ok(respbuilder.Success, PingResp{
		CommitHash:   s.buildCommitID,
		BuildTime:    s.buildTime,
		StartUpTime:  s.startupTime,
		UptimeNs:     time.Since(s.startupTime).Nanoseconds(),
		UptimeString: time.Since(s.startupTime).String(),
	}))
}

type SystemInfoRespBySize [61]struct {
	Size    uint32 `json:"size,omitempty"`
	Mallocs uint64 `json:"mallocs,omitempty"`
	Frees   uint64 `json:"frees,omitempty"`
}

type SystemInfoResp struct {
	Alloc         uint64               `json:"alloc,omitempty"`
	TotalAlloc    uint64               `json:"total_alloc,omitempty"`
	Sys           uint64               `json:"sys,omitempty"`
	Lookups       uint64               `json:"lookups,omitempty"`
	Mallocs       uint64               `json:"mallocs,omitempty"`
	Frees         uint64               `json:"frees,omitempty"`
	HeapAlloc     uint64               `json:"heap_alloc,omitempty"`
	HeapSys       uint64               `json:"heap_sys,omitempty"`
	HeapIdle      uint64               `json:"heap_idle,omitempty"`
	HeapInuse     uint64               `json:"heap_inuse,omitempty"`
	HeapReleased  uint64               `json:"heap_released,omitempty"`
	HeapObjects   uint64               `json:"heap_objects,omitempty"`
	StackInuse    uint64               `json:"stack_inuse,omitempty"`
	StackSys      uint64               `json:"stack_sys,omitempty"`
	MSpanInuse    uint64               `json:"m_span_inuse,omitempty"`
	MSpanSys      uint64               `json:"m_span_sys,omitempty"`
	MCacheInuse   uint64               `json:"m_cache_inuse,omitempty"`
	MCacheSys     uint64               `json:"m_cache_sys,omitempty"`
	BuckHashSys   uint64               `json:"buck_hash_sys,omitempty"`
	GCSys         uint64               `json:"gc_sys,omitempty"`
	OtherSys      uint64               `json:"other_sys,omitempty"`
	NextGC        uint64               `json:"next_gc,omitempty"`
	LastGC        uint64               `json:"last_gc,omitempty"`
	PauseTotalNs  uint64               `json:"pause_total_ns,omitempty"`
	PauseNs       [256]uint64          `json:"pause_ns,omitempty"`
	PauseEnd      [256]uint64          `json:"pause_end,omitempty"`
	NumGC         uint32               `json:"num_gc,omitempty"`
	NumForcedGC   uint32               `json:"num_forced_gc,omitempty"`
	GCCPUFraction float64              `json:"gccpu_fraction,omitempty"`
	EnableGC      bool                 `json:"enable_gc,omitempty"`
	DebugGC       bool                 `json:"debug_gc,omitempty"`
	BySize        SystemInfoRespBySize `json:"by_size,omitempty"`
}

func (s *SystemHandler) SystemInfo(c echo.Context) error {
	var bToMb = func(b uint64) uint64 {
		return b / 1024 / 1024
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	resp := SystemInfoResp{
		Alloc:         bToMb(m.Alloc),
		TotalAlloc:    bToMb(m.TotalAlloc),
		Sys:           bToMb(m.Sys),
		Lookups:       m.Lookups,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		HeapAlloc:     0,
		HeapSys:       0,
		HeapIdle:      0,
		HeapInuse:     0,
		HeapReleased:  0,
		HeapObjects:   0,
		StackInuse:    0,
		StackSys:      0,
		MSpanInuse:    0,
		MSpanSys:      0,
		MCacheInuse:   0,
		MCacheSys:     0,
		BuckHashSys:   0,
		GCSys:         0,
		OtherSys:      0,
		NextGC:        0,
		LastGC:        0,
		PauseTotalNs:  0,
		PauseNs:       [256]uint64{},
		PauseEnd:      [256]uint64{},
		NumGC:         m.NumGC,
		NumForcedGC:   m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
		EnableGC:      m.EnableGC,
		DebugGC:       m.DebugGC,
		BySize:        SystemInfoRespBySize{},
	}

	return c.JSON(http.StatusOK, respbuilder.Ok(respbuilder.Success, resp))
}