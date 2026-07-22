# Image server Remote PC untuk Linux (Docker/Proxmox). Hanya membangun
# cmd/server — agent TETAP harus jalan di Windows karena butuh WMI, screen
# capture, dan Windows service control yang tidak ada di Linux.
#
# Build dari root repo (context = root, bukan folder docker/), contoh:
#   docker build -f docker/Dockerfile -t remote-pc-server .
# atau pakai docker-compose.yml di folder ini.

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# GIT_COMMIT opsional: .git tidak ikut ke context (lihat .dockerignore), jadi
# commit hash tak bisa dibaca otomatis di dalam container. Isi manual kalau
# mau, mis: docker compose build --build-arg GIT_COMMIT=$(git rev-parse --short HEAD)
# BuildTime SELALU terisi otomatis (waktu image ini dibangun) — cukup untuk
# memastikan rebuild memang mengambil kode terbaru, cek di halaman /version.
ARG GIT_COMMIT=unknown
RUN BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X remote_pc/internal/version.GitCommit=${GIT_COMMIT} -X remote_pc/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache su-exec && adduser -D -u 1000 remotepc
WORKDIR /app
COPY --from=build /out/server /app/server
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh \
    && mkdir -p /app/config /app/data /app/logs /app/screenshots /app/uploads /app/downloads \
    && chown -R remotepc:remotepc /app
EXPOSE 9000
# Container start sebagai root agar entrypoint bisa membetulkan kepemilikan
# folder yang di-bind-mount dari host, lalu turun ke user "remotepc" sendiri
# sebelum menjalankan server (lihat entrypoint.sh).
ENTRYPOINT ["/app/entrypoint.sh"]
