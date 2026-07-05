#!/bin/sh
# Dijalankan sebagai root saat container start: folder yang di-bind-mount dari
# host (docker/data/...) sering kali dibuat otomatis oleh Docker milik root,
# sementara server berjalan sebagai user non-root "remotepc" demi keamanan.
# Betulkan kepemilikannya di sini, baru turunkan privilege sebelum exec server.
set -e
chown -R remotepc:remotepc /app/config /app/data /app/logs /app/screenshots /app/uploads /app/downloads
exec su-exec remotepc /app/server -config /app/config/config.yaml
