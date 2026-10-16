#!/usr/bin/env bash
# ==============================================================================
# Stream Ultra-Leve com Renderizador Go Nativo (cmd/hud) — ZERO CHROME (< 50MB RAM)
# Frames 1920x1080@10fps em Go (fogleman/gg) -> pipe rawvideo rgba -> FFmpeg
# Destinos: Kick + Twitch + YouTube + Audio Radio Lo-Fi
# ==============================================================================

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

FPS=10
KICK_INGEST="rtmps://fa723fc1b171.global-contribute.live-video.net:443/app/sk_us-west-2_kwGOE5X6GOkk_lHTZEDKeLJhh0RoCC5tcmOrUWiFLbS"
TWITCH_INGEST="rtmp://live.twitch.tv/app/live_21107325_YimOnx0kdCUXwTxLJA0Gx6aXgHiMXE"
YOUTUBE_INGEST="rtmp://a.rtmp.youtube.com/live2/5fst-9b20-k652-g8mg-10p0"
RADIO_URL="https://radio.plaza.one/mp3"

HUD_BIN="$DIR/hud/restream-hud"

echo "========================================================================"
echo " ⚡ INICIANDO STREAM NATIVO EM GO (< 50MB RAM TOTAL) ⚡"
echo " Zero Chrome / Zero Xvfb — Renderização 2D Canvas em Go"
echo " Destinos: Kick + Twitch + YouTube + Audio Lo-Fi"
echo "========================================================================"

cleanup() {
    echo ""
    echo "[!] Encerrando processos..."
    kill $(jobs -p) 2>/dev/null || true
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

chmod +x "$HUD_BIN"

echo "[1/1] Iniciando Motor Nativo Go com FFmpeg Embutido..."
exec "$HUD_BIN" -fps $FPS
