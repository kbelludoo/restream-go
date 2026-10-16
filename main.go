package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kbelludoo/restream-go/pkg/config"
	"github.com/kbelludoo/restream-go/pkg/ingest"
	"github.com/kbelludoo/restream-go/pkg/logx"
	"github.com/kbelludoo/restream-go/pkg/relay"
	"github.com/kbelludoo/restream-go/pkg/server"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("     ⚡ RESTREAM GO — FAN-OUT RTMP ULTRA-LEVE (MULTI-PLATAFORMA) ⚡            ")
	fmt.Println("     OBS → Twitch • Kick • YouTube • Bilibili  |  Sem transcoding               ")
	fmt.Println("================================================================================")

	configFile := "config.json"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("[Restream] Erro na configuração: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logx.Printf("\n[Restream] Encerrando graciosamente...")
		cancel()
	}()

	mgr := relay.NewManager(cfg)
	go mgr.Start(ctx)

	ing := ingest.NewServer(cfg.Ingest, mgr)
	go ing.Start(ctx)

	srv := server.NewServer(cfg.Settings.WebDashboardPort, mgr)
	srv.Start()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t := mgr.GetTelemetry()

			liveCount := 0
			for _, tgt := range t.Targets {
				if tgt.Status == "enviando" {
					liveCount++
				}
			}

			srcDesc := "offline"
			if t.Source.Connected {
				srcDesc = fmt.Sprintf("online (%d frames)", t.Source.Frames)
			}

			fmt.Printf("[Ticker] RAM: %.2f MB | OBS: %s | Destinos ao vivo: %d/%d | Uptime: %s\n",
				t.RamMb, srcDesc, liveCount, len(t.Targets), t.Uptime)
		}
	}
}

func init() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}
	_ = runtime.GOMAXPROCS(runtime.NumCPU())
}
