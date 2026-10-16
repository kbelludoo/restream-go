@echo off
title Restream Go - Fan-out Multi-plataforma
echo ===================================================
echo     Iniciando Restream Go...
echo ===================================================
if not exist "restream.exe" (
    echo Compilando restream.exe...
    go build -ldflags="-s -w" -o restream.exe .
)
restream.exe
pause
