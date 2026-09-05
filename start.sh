#!/bin/bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

PORT="${PORT:-7862}"
HELPER_PORT="${HELPER_PORT:-18835}"
mkdir -p "$DIR/logs" "$DIR/data" "$DIR/bin"

# 1. Start ConvertSW Helper Daemon (Mistral, ChatGPT, DeepSeek PoW) if not running
if ! pgrep -f "ConvertSW-Gateway/backend/helper/daemon.js" > /dev/null; then
  echo "[*] Memulai ConvertSW Multi-Provider Helper Daemon (Port: $HELPER_PORT)..."
  setsid -f node "$DIR/backend/helper/daemon.js" </dev/null > "$DIR/logs/helper.log" 2>&1
  sleep 1
fi

# 2. Start ConvertSW-Gateway backend
if [ "$1" = "--daemon" ] || [ "$1" = "-d" ]; then
  if pgrep -f "convertsw-gateway" > /dev/null; then
    PID=$(pgrep -f "convertsw-gateway" | tr '\n' ' ')
    echo "[!] ConvertSW-Gateway sudah berjalan (PID: $PID)."
    exit 0
  fi
  setsid -f "$DIR/bin/convertsw-gateway" </dev/null > "$DIR/logs/output.log" 2>&1
  sleep 1
  PID=$(pgrep -f "convertsw-gateway")
  if [ -n "$PID" ]; then
    echo "[+] ConvertSW-Gateway berhasil dijalankan di background (PID: $PID)"
    echo "[+] WebUI & API: http://127.0.0.1:$PORT"
    echo "[+] Log: $DIR/logs/output.log"
  else
    echo "[-] Gagal menjalankan ConvertSW-Gateway di background. Cek log: $DIR/logs/output.log"
    exit 1
  fi
else
  echo "[*] Menjalankan ConvertSW-Gateway di port $PORT..."
  echo "[*] WebUI & API: http://127.0.0.1:$PORT"
  echo "[*] Tekan Ctrl+C untuk menghentikan."
  exec "$DIR/bin/convertsw-gateway"
fi
