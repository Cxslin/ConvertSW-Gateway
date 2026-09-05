#!/bin/bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "[*] Menghentikan ConvertSW-Gateway..."

PID=$(pgrep -f "convertsw-gateway")
if [ -n "$PID" ]; then
  echo "[-] Menghentikan backend (PID: $PID)..."
  kill $PID 2>/dev/null || true
  sleep 1
  if pgrep -f "convertsw-gateway" > /dev/null; then
    kill -9 $(pgrep -f "convertsw-gateway") 2>/dev/null || true
  fi
fi

HELPER_PID=$(pgrep -f "ConvertSW-Gateway/backend/helper/daemon.js")
if [ -n "$HELPER_PID" ]; then
  echo "[-] Menghentikan Helper daemon (PID: $HELPER_PID)..."
  kill $HELPER_PID 2>/dev/null || true
  sleep 1
  if pgrep -f "ConvertSW-Gateway/backend/helper/daemon.js" > /dev/null; then
    kill -9 $(pgrep -f "ConvertSW-Gateway/backend/helper/daemon.js") 2>/dev/null || true
  fi
fi

echo "[+] ConvertSW-Gateway berhasil dihentikan."
