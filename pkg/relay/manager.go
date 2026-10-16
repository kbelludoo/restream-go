package relay

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"

	"github.com/kbelludoo/restream-go/pkg/config"
	"github.com/kbelludoo/restream-go/pkg/logx"
)

var errNoH264 = errors.New("stream sem track de vídeo H.264; configure o OBS para enviar H.264")

type source struct {
	gen        uint64
	tracks     []*gortmplib.Track
	videoTrack *gortmplib.Track
	audioTrack *gortmplib.Track
	gop        *gopCache

	frames atomic.Int64
}

type Manager struct {
	cfg *config.Config

	targets []*Target

	mu      sync.RWMutex
	src     *source
	gen     atomic.Uint64
	started time.Time

	logsMu sync.Mutex
	logs   []string
}

func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		cfg:     cfg,
		started: time.Now(),
	}
	for _, tc := range cfg.Targets {
		t := NewTarget(tc, m, cfg.Settings.TargetQueueSize)
		m.targets = append(m.targets, t)
	}
	return m
}

func (m *Manager) Start(ctx context.Context) {
	for _, t := range m.targets {
		t.Start(ctx)
	}

	// sampler de bitrate dos targets (2s)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, t := range m.targets {
					t.SampleBitrate()
				}
			}
		}
	}()
}

// AttachPublisher registra o stream vindO do OBS e notifica todos os targets.
func (m *Manager) AttachPublisher(reader *gortmplib.Reader) (*source, error) {
	tracks := reader.Tracks()

	var videoTrack, audioTrack *gortmplib.Track
	for _, tr := range tracks {
		switch tr.Codec.(type) {
		case *codecs.H264:
			if videoTrack == nil {
				videoTrack = tr
			}
		case *codecs.MPEG4Audio:
			if audioTrack == nil {
				audioTrack = tr
			}
		}
	}

	if videoTrack == nil {
		return nil, errNoH264
	}

	gen := m.gen.Add(1)
	src := &source{
		gen:        gen,
		tracks:     tracks,
		videoTrack: videoTrack,
		audioTrack: audioTrack,
		gop:        newGOPCache(m.cfg.Settings.GOPCacheMaxBytes),
	}

	m.mu.Lock()
	old := m.src
	m.src = src
	m.mu.Unlock()

	if old != nil {
		m.logf("fonte substituída (nova sessão do OBS)")
	}
	m.logf("OBS conectado: %d track(s), repasse iniciando", len(tracks))

	// força reconexão dos targets para pegar os novos tracks
	for _, t := range m.targets {
		t.requestReconnect()
	}

	return src, nil
}

func (m *Manager) DetachPublisher(src *source) {
	m.mu.Lock()
	if m.src == src {
		m.src = nil
	}
	m.mu.Unlock()

	m.logf("OBS desconectado; aguardando nova conexão")
	for _, t := range m.targets {
		t.requestReconnect()
	}
}

func (m *Manager) currentSource() *source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.src
}

func (m *Manager) generation() uint64 {
	return m.gen.Load()
}

// HandleVideo/HandleAudio são chamados pelo loop de leitura do ingest.
func (m *Manager) HandleVideo(src *source, pts, dts time.Duration, au [][]byte) {
	p := &packet{
		gen:   src.gen,
		video: true,
		pts:   pts,
		dts:   dts,
		au:    cloneAU(au),
	}
	src.frames.Add(1)
	src.gop.push(p)
	for _, t := range m.targets {
		t.Enqueue(p)
	}
}

func (m *Manager) HandleAudio(src *source, pts time.Duration, aac []byte) {
	buf := make([]byte, len(aac))
	copy(buf, aac)
	p := &packet{
		gen:   src.gen,
		video: false,
		pts:   pts,
		dts:   pts,
		aac:   buf,
	}
	src.gop.push(p)
	for _, t := range m.targets {
		t.Enqueue(p)
	}
}

func (m *Manager) logf(format string, args ...any) {
	logx.Printf(format, args...)
}

type SourceInfo struct {
	Connected bool   `json:"connected"`
	Frames    int64  `json:"frames"`
	Tracks    int    `json:"tracks"`
	Codec     string `json:"codec"`
}

type Telemetry struct {
	RamMb    float64      `json:"ramMb"`
	Uptime   string       `json:"uptime"`
	Source   SourceInfo   `json:"source"`
	Targets  []TargetInfo `json:"targets"`
	Logs     []string     `json:"logs"`
}

func (m *Manager) GetTelemetry() Telemetry {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	tel := Telemetry{
		RamMb:  float64(ms.HeapAlloc) / (1024 * 1024),
		Uptime: time.Since(m.started).Truncate(time.Second).String(),
		Logs:   logx.Tail(),
	}

	if src := m.currentSource(); src != nil {
		tel.Source = SourceInfo{
			Connected: true,
			Frames:    src.frames.Load(),
			Tracks:    len(src.tracks),
			Codec:     "H.264 + AAC",
		}
	}

	for _, t := range m.targets {
		tel.Targets = append(tel.Targets, t.Info())
	}

	return tel
}
