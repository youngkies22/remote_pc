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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 1000 remotepc
WORKDIR /app
COPY --from=build /out/server /app/server
RUN mkdir -p /app/config /app/data /app/logs /app/screenshots /app/uploads /app/downloads \
    && chown -R remotepc:remotepc /app
USER remotepc
EXPOSE 7000
ENTRYPOINT ["/app/server", "-config", "/app/config/config.yaml"]
