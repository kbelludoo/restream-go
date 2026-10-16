#!/usr/bin/env bash
# ==============================================================================
# Stream com renderizador Go nativo (cmd/hud) — SEM Chrome, Xvfb e navegador
# Frames 1920x1080@30fps em Go -> pipe rawvideo -> FFmpeg -> Kick/Twitch/YouTube
# ==============================================================================

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

FPS=30
KICK_INGEST="rtmps://fa723fc1b171.global-contribute.live-video.net:443/app/sk_us-west-2_kwGOE5X6GOkk_lHTZEDKeLJhh0RoCC5tcmOrUWiFLbS"
TWITCH_INGEST="rtmp://live.twitch.tv/app/live_21107325_YimOnx0kdCUXwTxLJA0Gx6aXgHiMXE"
YOUTUBE_INGEST="rtmp://a.rtmp.youtube.com/live2/5fst-9b20-k652-g8mg-10p0"
RADIO_URL="https://radio.plaza.one/mp3"

HUD_BIN="$DIR/hud/restream-hud"
HUD_CFG="$DIR/hud/hud.json"

echo "========================================================================"
echo " ⚡ INICIANDO STREAM — RENDERIZADOR GO NATIVO ⚡"
echo " Sem Chrome / Xvfb / navegador — frames desenhados em Go"
echo " Destinos: Kick + Twitch + YouTube"
echo "========================================================================"

cleanup() {
    echo ""
    echo "[!] Encerrando processos..."
    kill $(jobs -p) 2>/dev/null || true
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

if [ ! -x "$HUD_BIN" ]; then
    echo "[Erro] $HUD_BIN não encontrado"
    exit 1
fi

if ! pgrep -f "restream-linux" > /dev/null; then
    if [ -f "./restream-linux" ]; then
        echo "[1/2] Iniciando restream-linux (ingest OBS)..."
        ./restream-linux > restream.log 2>&1 &
        sleep 1
    fi
else
    echo "[1/2] restream-linux já está em execução."
fi

echo "[2/2] Transmitindo HUD Go via FFmpeg..."
"$HUD_BIN" -config "$HUD_CFG" -o - | ffmpeg -hide_banner -loglevel info \
    -thread_queue_size 1024 -f rawvideo -pix_fmt rgb24 -video_size 1920x1080 -framerate $FPS -i pipe:0 \
    -thread_queue_size 1024 -reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 5 -i "$RADIO_URL" \
    -c:v libx264 -preset veryfast -tune zerolatency \
    -b:v 3000k -maxrate 3000k -bufsize 6000k \
    -pix_fmt yuv420p -g 60 -keyint_min 60 -sc_threshold 0 \
    -c:a aac -b:a 128k -ar 44100 \
    -af "aresample=async=1000,volume=0.8" \
    -flags +global_header \
    -f tee -map 0:v -map 1:a \
    "[f=flv:onfail=ignore:flvflags=no_duration_filesize]${KICK_INGEST}|[f=flv:onfail=ignore:flvflags=no_duration_filesize]${TWITCH_INGEST}|[f=flv:onfail=ignore:flvflags=no_duration_filesize]${YOUTUBE_INGEST}"
