#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "=========================================="
echo "  Memperbarui deepseek2API Versi Terbaru  "
echo "=========================================="

if [ -f "$DIR/stop.sh" ]; then
  echo "[1/4] Menghentikan layanan yang sedang berjalan..."
  bash "$DIR/stop.sh" || true
fi

echo "[2/4] Mengambil pembaruan terbaru dari GitHub..."
git pull origin main || true

echo "[3/4] Mengompilasi WebUI (Frontend)..."
cd "$DIR/frontend"
npm install
npm run build

echo "[4/4] Mengompilasi server Go (Backend)..."
cd "$DIR/backend"
go build -trimpath -ldflags="-s -w" -o "$DIR/bin/deepseek2api-backend" .

cd "$DIR"
echo "[+] Menjalankan kembali deepseek2API..."
bash "$DIR/start.sh" -d

echo ""
echo "=========================================="
echo " [+] Pembaruan Berhasil Diselesaikan!     "
echo " WebUI & API: http://127.0.0.1:${PORT:-7861}"
echo "=========================================="
