package relay

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluenviron/gortmplib"

	"github.com/kbelludoo/restream-go/pkg/config"
	"github.com/kbelludoo/restream-go/pkg/logx"
)

const (
	statusDisabled     = "desativado"
	statusWaiting      = "aguardando fonte"
	statusConnecting   = "conectando"
	statusLive         = "enviando"
	statusReconnecting = "reconectando"
	statusError        = "erro"
)

type Target struct {
	cfg config.TargetConfig

	mgr  *Manager
	queue chan *packet

	forceReconnect chan struct{}

	mu          sync.Mutex
	status      string
	lastError   string
	reconnects  int
	bytesSent   int64
	lastBytes   int64
	bitrateBps  int64
	dropped     int64
	connectedAt time.Time

	client atomic.Pointer[gortmplib.Client]
}

func NewTarget(cfg config.TargetConfig, mgr *Manager, queueSize int) *Target {
	st := statusWaiting
	if !cfg.Enabled {
		st = statusDisabled
	}
	return &Target{
		cfg:            cfg,
		mgr:            mgr,
		queue:          make(chan *packet, queueSize),
		forceReconnect: make(chan struct{}, 1),
		status:         st,
	}
}

func (t *Target) Start(ctx context.Context) {
	if !t.cfg.Enabled {
		return
	}
	go t.loop(ctx)
}

func (t *Target) loop(ctx context.Context) {
	backoff := time.Second
	maxDelay := time.Duration(t.mgr.cfg.Settings.ReconnectMaxDelaySeconds) * time.Second

	for {
		if ctx.Err() != nil {
			t.setStatus(statusDisabled, "")
			return
		}

		src := t.mgr.currentSource()
		if src == nil {
			t.setStatus(statusWaiting, "")
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		t.setStatus(statusConnecting, "")
		start := time.Now()
		err := t.streamOnce(ctx, src)

		if ctx.Err() != nil {
			t.setStatus(statusDisabled, "")
			return
		}

		if time.Since(start) > 30*time.Second {
			backoff = time.Second
		}

		if err != nil {
			t.setStatus(statusReconnecting, err.Error())
			t.incReconnects()
			logx.Printf("[Relay:%s] reconectando em %s: %v", t.cfg.Name, backoff, err)
		} else {
			t.setStatus(statusReconnecting, "")
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxDelay {
			backoff = maxDelay
		}
	}
}

func (t *Target) streamOnce(ctx context.Context, src *source) error {
	u, err := url.Parse(t.cfg.URL + "#" + t.cfg.Key)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}

	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	client := &gortmplib.Client{
		URL:     u,
		Publish: true,
	}
	if err := client.Initialize(cctx); err != nil {
		return fmt.Errorf("conexão falhou: %w", err)
	}
	t.client.Store(client)
	defer func() {
		t.client.Store(nil)
		client.Close()
	}()

	writer := &gortmplib.Writer{
		Conn:   client,
		Tracks: src.tracks,
	}
	if err := writer.Initialize(); err != nil {
		return fmt.Errorf("writer init: %w", err)
	}

	t.setStatus(statusLive, "")
	t.setConnectedAt(time.Now())
	logx.Printf("[Relay:%s] conectado e enviando", t.cfg.Name)

	// replay do GOP cache para decodificação imediata no destino
	for _, p := range src.gop.snapshot() {
		if err := t.writePacket(client, writer, src, p); err != nil {
			return fmt.Errorf("replay GOP: %w", err)
		}
	}

	for {
		if t.mgr.generation() != src.gen {
			return fmt.Errorf("fonte de origem substituída")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-t.forceReconnect:
			return fmt.Errorf("reconexão forçada (fila saturada)")

		case p := <-t.queue:
			if p.gen != src.gen {
				continue
			}
			if err := t.writePacket(client, writer, src, p); err != nil {
				return fmt.Errorf("envio: %w", err)
			}
		}
	}
}

func (t *Target) writePacket(client *gortmplib.Client, writer *gortmplib.Writer, src *source, p *packet) error {
	client.NetConn().SetWriteDeadline(time.Now().Add(10 * time.Second))

	var err error
	if p.video {
		err = writer.WriteH264(src.videoTrack, p.pts, p.dts, p.au)
	} else if src.audioTrack != nil {
		err = writer.WriteMPEG4Audio(src.audioTrack, p.pts, p.aac)
	}
	if err != nil {
		return err
	}

	t.addBytes(int64(p.size()))
	return nil
}

// Enqueue entrega o pacote à fila do target; política: descarta se saturada
// e força reconexão (resync via GOP) após muitas perdas.
func (t *Target) Enqueue(p *packet) {
	if !t.cfg.Enabled {
		return
	}
	select {
	case t.queue <- p:
		t.mu.Lock()
		t.dropped = 0
		t.mu.Unlock()
	default:
		t.mu.Lock()
		t.dropped++
		over := t.dropped > int64(cap(t.queue))*2
		t.mu.Unlock()
		if over {
			t.requestReconnect()
		}
	}
}

func (t *Target) requestReconnect() {
	if c := t.client.Load(); c != nil {
		c.Close()
	}
	select {
	case t.forceReconnect <- struct{}{}:
	default:
	}
}

func (t *Target) addBytes(n int64) {
	t.mu.Lock()
	t.bytesSent += n
	t.mu.Unlock()
}

// SampleBitrate calcula o bitrate médio; chamado periodicamente pelo manager.
func (t *Target) SampleBitrate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	delta := t.bytesSent - t.lastBytes
	t.lastBytes = t.bytesSent
	// média móvel simples com o sampler do manager (2s)
	t.bitrateBps = (t.bitrateBps + delta/2) / 2
}

func (t *Target) setStatus(status, errMsg string) {
	t.mu.Lock()
	t.status = status
	if errMsg != "" {
		t.lastError = errMsg
	}
	t.mu.Unlock()
}

func (t *Target) setConnectedAt(ts time.Time) {
	t.mu.Lock()
	t.connectedAt = ts
	t.mu.Unlock()
}

func (t *Target) incReconnects() {
	t.mu.Lock()
	t.reconnects++
	t.mu.Unlock()
}

type TargetInfo struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	LastError   string `json:"lastError"`
	Reconnects  int    `json:"reconnects"`
	BitrateKbps int64  `json:"bitrateKbps"`
	SentMB      float64 `json:"sentMb"`
	Dropped     int64  `json:"dropped"`
	Uptime      string `json:"uptime"`
}

func (t *Target) Info() TargetInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	uptime := ""
	if t.status == statusLive && !t.connectedAt.IsZero() {
		uptime = time.Since(t.connectedAt).Truncate(time.Second).String()
	}

	u := t.cfg.URL
	if i := len(u); i > 0 {
		u = u + "/***"
	}

	return TargetInfo{
		Name:        t.cfg.Name,
		URL:         u,
		Status:      t.status,
		LastError:   t.lastError,
		Reconnects:  t.reconnects,
		BitrateKbps: t.bitrateBps * 8 / 1000,
		SentMB:      float64(t.bytesSent) / (1024 * 1024),
		Dropped:     t.dropped,
		Uptime:      uptime,
	}
}
