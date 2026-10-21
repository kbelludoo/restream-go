# ⚡ Restream Go — Ultra-Low RAM Live Engine (< 120MB RAM)

Motor de transmissão ao vivo de alta performance escrito 100% em **Go**, projetado para rodar em VPS de baixos recursos (512MB–1GB RAM).

O sistema renderiza um HUD 2D em tempo real na memória RAM e alimenta o FFmpeg diretamente pelo `stdin` (`pipe:0` rawvideo RGBA), transmitindo simultaneamente para **Kick**, **Twitch** e **YouTube** com trilha sonora de rádio Lo-Fi sincronizada.

---

## 🚀 Arquitetura & Otimizações

- **Zero Chrome / Zero Puppeteer / Zero Xvfb**: Elimina os 400MB+ de consumo de navegadores headless.
- **Frame RGBA Reutilizado**: Mantém um único frame completo e atualiza as regiões animadas no próprio buffer, sem copiar a tela inteira a cada quadro.
- **Recorte sem Buffers Temporários**: O letreiro usa uma janela sobre os pixels do frame, evitando máscaras e imagens do tamanho da tela a cada quadro. Os tamanhos de texto compartilham o mesmo arquivo de fonte já carregado.
- **Filas Curtas no FFmpeg**: A entrada de vídeo usa uma fila de 2 frames e os filtros usam 1 thread. Quando a saída atrasa, o pipe segura o produtor; a fila de áudio mantém 32 pacotes. Veja as [opções do FFmpeg](https://ffmpeg.org/ffmpeg.html).
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

As cifras da VPS acima são referências anteriores às otimizações de memória. O consumo total atualizado deve ser medido na VPS, incluindo o processo FFmpeg.

### Comparação local das otimizações

Medição em Windows/amd64, Go 1.27, com 720p a 10 FPS e saída rawvideo para `NUL`: média de 25 amostras de Working Set, uma por segundo, após 10 segundos de aquecimento. Apenas o processo Go, sem FFmpeg:

| Métrica | Antes | Depois |
| :--- | ---: | ---: |
| RAM média do processo Go | 114,34 MiB | 71,36 MiB |
| Bytes alocados por quadro, 720p | 8.383.507 | ~74.000 |
| Bytes alocados por quadro, 1080p | 18.824.511 | ~132.000 |

A média de RAM caiu cerca de 38% nesse ambiente. Os bytes alocados por quadro são memória temporária, não o total residente. As fontes do Windows e do Linux diferem, portanto os valores absolutos não são diretamente comparáveis à VPS.

Para repetir o benchmark e validar os frames e o fan-out local:

```bash
go test ./...
go test ./cmd/hud -run '^$' -bench BenchmarkHUDFrame -benchmem -benchtime=100x
```

O teste de fan-out usa FFmpeg e FFprobe quando disponíveis e verifica vídeo H.264 e áudio AAC em três arquivos locais, em 720p e 1080p.

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
