#!/bin/sh
# Build biner server Remote PC untuk Linux (tanpa Docker) — mis. untuk
# dijalankan langsung di VM/LXC Proxmox. Agent TIDAK dibangun di sini karena
# agent harus tetap berjalan di Windows (screen capture, WMI, service control).
#
# Jalankan dari mana saja:
#   sh docker/build-linux.sh
set -e

root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$root"

mkdir -p docker/bin
echo "==> build server (linux/amd64)"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o docker/bin/server-linux-amd64 ./cmd/server

echo ""
echo "Selesai: docker/bin/server-linux-amd64"
