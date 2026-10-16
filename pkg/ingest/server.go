package ingest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"

	"github.com/kbelludoo/restream-go/pkg/config"
	"github.com/kbelludoo/restream-go/pkg/logx"
	"github.com/kbelludoo/restream-go/pkg/relay"
)

type Server struct {
	cfg config.IngestConfig
	mgr *relay.Manager
}

func NewServer(cfg config.IngestConfig, mgr *relay.Manager) *Server {
	return &Server{cfg: cfg, mgr: mgr}
}

func (s *Server) Start(ctx context.Context) {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logx.Printf("[Ingest] erro ao escutar em %s: %v", addr, err)
		return
	}

	logx.Printf("[Ingest] RTMP server online em rtmp://localhost%s/%s/<streamKey>", addr, s.cfg.App)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logx.Printf("[Ingest] erro no accept: %v", err)
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	remote := conn.RemoteAddr().String()
	logx.Printf("[Ingest] conexão de %s", remote)
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(15 * time.Second))

	sc := &gortmplib.ServerConn{RW: conn}
	if err := sc.Initialize(); err != nil {
		logx.Printf("[Ingest] handshake falhou (%s): %v", remote, err)
		return
	}

	if err := sc.AcceptConn(); err != nil {
		logx.Printf("[Ingest] accept falhou (%s): %v", remote, err)
		return
	}

	if !sc.Publish {
		logx.Printf("[Ingest] rejeitado (%s): conexão não-publicadora", remote)
		_ = sc.RejectAction()
		return
	}

	if !s.validateKey(sc.URL) {
		logx.Printf("[Ingest] rejeitado (%s): stream key inválida", remote)
		_ = sc.RejectAction()
		return
	}

	reader := &gortmplib.Reader{Conn: sc}
	if err := reader.Initialize(); err != nil {
		logx.Printf("[Ingest] reader init falhou (%s): %v", remote, err)
		return
	}

	src, err := s.mgr.AttachPublisher(reader)
	if err != nil {
		logx.Printf("[Ingest] rejeitado (%s): %v", remote, err)
		_ = sc.RejectAction()
		return
	}
	defer s.mgr.DetachPublisher(src)

	for _, tr := range reader.Tracks() {
		logx.Printf("[Ingest] track detectado: %T", tr.Codec)
	}

	for _, tr := range reader.Tracks() {
		track := tr
		switch track.Codec.(type) {
		case *codecs.H264:
			reader.OnDataH264(track, func(pts, dts time.Duration, au [][]byte) {
				_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
				s.mgr.HandleVideo(src, pts, dts, au)
			})
		case *codecs.MPEG4Audio:
			reader.OnDataMPEG4Audio(track, func(pts time.Duration, au []byte) {
				_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
				s.mgr.HandleAudio(src, pts, au)
			})
		}
	}

	for {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
		if err := reader.Read(); err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			logx.Printf("[Ingest] stream encerrado (%s): %v", remote, err)
			return
		}
	}
}

func (s *Server) validateKey(u *url.URL) bool {
	if s.cfg.StreamKey == "" {
		return true
	}
	if u == nil {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return false
	}
	return parts[len(parts)-1] == s.cfg.StreamKey
}
