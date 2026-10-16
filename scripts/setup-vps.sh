#!/usr/bin/env bash
# ==============================================================================
# Setup completo para VPS Linux (Ubuntu / Debian)
# Transmissão ultra-leve de página web sem OBS (Headless Chromium + Xvfb + FFmpeg)
# Consumo médio: ~180MB a 250MB de RAM total
# ==============================================================================
set -e

echo "=== [1/4] Atualizando pacotes e instalando dependências mínimas ==="
sudo apt-get update -y
sudo apt-get install -y --no-install-recommends \
    xvfb \
    ffmpeg \
    chromium-browser \
    fonts-liberation \
    fonts-noto-color-emoji \
    xdotool \
    pulseaudio

# Em distros recentes onde chromium-browser é snap ou pacote alternativo:
if ! command -v chromium-browser &> /dev/null && ! command -v chromium &> /dev/null; then
    echo "Instalando chromium alternativo..."
    sudo apt-get install -y chromium || true
fi

echo "=== [2/4] Criando diretórios e permissões ==="
mkdir -p "$HOME/restream"
mkdir -p "$HOME/restream/scripts"
chmod +x "$HOME/restream/scripts/"*.sh 2>/dev/null || true

echo "=== [3/4] Instalação concluída com sucesso! ==="
echo "Execute: bash ~/restream/scripts/start-stream.sh"
