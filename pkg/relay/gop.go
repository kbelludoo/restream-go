package relay

import (
	"sync"
	"time"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

// packet é unidade de repasse: vídeo H.264 (AU) ou áudio AAC (AU).
type packet struct {
	gen   uint64
	video bool
	pts   time.Duration
	dts   time.Duration
	au    [][]byte // vídeo
	aac   []byte   // áudio
}

func (p *packet) size() int {
	if p.video {
		n := 0
		for _, nalu := range p.au {
			n += len(nalu)
		}
		return n
	}
	return len(p.aac)
}

func cloneAU(au [][]byte) [][]byte {
	out := make([][]byte, len(au))
	for i, nalu := range au {
		c := make([]byte, len(nalu))
		copy(c, nalu)
		out[i] = c
	}
	return out
}

// gopCache guarda o GOP atual (do último keyframe até agora) para replay
// quando um target (re)conecta no meio da live.
type gopCache struct {
	mu        sync.Mutex
	pkts      []*packet
	bytes     int
	maxBytes  int
	needKey   bool
}

func newGOPCache(maxBytes int) *gopCache {
	return &gopCache{maxBytes: maxBytes, needKey: true}
}

func (g *gopCache) reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pkts = nil
	g.bytes = 0
	g.needKey = true
}

func (g *gopCache) push(p *packet) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if p.video && h264.IsRandomAccess(p.au) {
		g.pkts = nil
		g.bytes = 0
		g.needKey = false
	}
	if g.needKey {
		return
	}

	g.pkts = append(g.pkts, p)
	g.bytes += p.size()

	if g.bytes > g.maxBytes {
		g.pkts = nil
		g.bytes = 0
		g.needKey = true
	}
}

func (g *gopCache) snapshot() []*packet {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]*packet, len(g.pkts))
	copy(out, g.pkts)
	return out
}
