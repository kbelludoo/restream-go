# ⚡ Restream Go — Ultra-Low RAM Live Engine (< 120MB RAM)

Motor de transmissão ao vivo de alta performance escrito 100% em **Go**, projetado para rodar em VPS de baixos recursos (512MB–1GB RAM).

O sistema renderiza um HUD 2D em tempo real na memória RAM e alimenta o FFmpeg diretamente pelo `stdin` (`pipe:0` rawvideo RGBA), transmitindo simultaneamente para **Kick**, **Twitch** e **YouTube** com trilha sonora de rádio Lo-Fi sincronizada.

---

## 🚀 Arquitetura & Otimizações

- **Zero Chrome / Zero Puppeteer / Zero Xvfb**: Elimina os 400MB+ de consumo de navegadores headless.
- **Zero Alocação no Loop de Render**: Utiliza buffers de frame RGBA nativos na RAM, evitando acionamentos contínuos do Garbage Collector (GC).
- **Embedded Process Orchestrator**: O binário em Go gerencia o ciclo de vida do FFmpeg internamente via `exec.Command` com auto-restart em caso de queda de rede.
- **Multi-Destino Fan-out via Tee**: Codificação única em H.264 ultrafast com saída simultânea para RTMP/RTMPS Kick, Twitch e YouTube.
- **Modo Stealth Integrado**: Anonimização total de personagens e contas de bots para proteção e privacidade.
- **Telemetria do Próprio Processo**: Coleta o consumo de memória RSS diretamente do Linux `/proc/self/statm` sem overhead.

---

## 📊 Consumo de Recursos na VPS

| Componente | Memória RAM | CPU |
| :--- | :--- | :--- |
| **Go Engine (`restream-hud`)** | **~17 MB – 36 MB RSS** | ~2% |
| **FFmpeg Encoder (libx264)** | **~90 MB – 100 MB RSS** | ~6% |
| **Cgroup Total (`stream.service`)** | **~116 MB RAM** | **~8% (1 vCPU)** |

---

## 🌐 GitHub Pages & Web Dashboard

O repositório inclui na pasta `/docs` um dashboard interativo para navegadores contendo:
- **Player Web**: Embed ao vivo da Twitch e YouTube.
- **Rádio Lo-Fi Integrado**: Player web Nightwave Plaza com visualizador de ondas.
- **Telemetria 3x3 dos Bots**: Conectado diretamente à API `status.json` do cluster.

👉 **Acesse em:** [kbelludoo.github.io/restream-go](https://kbelludoo.github.io/restream-go/)

---

## 🛠️ Como Executar

### 1. Requisitos
- Go 1.22+
- FFmpeg 6.0+

### 2. Variáveis de Ambiente
Configure suas chaves de transmissão:
```bash
export KICK_INGEST="rtmps://fa723fc1b171.global-contribute.live-video.net:443/app/SUA_CHAVE_KICK"
export TWITCH_INGEST="rtmp://live.twitch.tv/app/SUA_CHAVE_TWITCH"
export YOUTUBE_INGEST="rtmp://a.rtmp.youtube.com/live2/SUA_CHAVE_YOUTUBE"
export RADIO_URL="https://radio.plaza.one/mp3"
```

### 3. Compilar e Iniciar
```bash
# Compilar binário otimizado
go build -ldflags="-s -w" -o restream-hud ./cmd/hud

# Iniciar em modo 720p HD @ 10 FPS
./restream-hud -res 720p -fps 10
```

---

## 📄 Licença
MIT © [kbelludoo](https://github.com/kbelludoo)
