# Remote PC Server — Linux / Docker

Folder ini berisi cara menjalankan **server** (dashboard guru) di Linux, mis.
di VM atau LXC container Proxmox. **Agent tetap wajib jalan di Windows** —
agent butuh screen capture, WMI, dan kontrol service Windows yang tidak ada
di Linux. Hanya server yang dipindah; agent tetap disambungkan ke IP/port
server ini seperti biasa (lihat `agent.yaml` di README utama).

## Opsi 1 — Docker (disarankan)

Dari root repo:

```sh
docker compose -f docker/docker-compose.yml up -d --build
```

- Dashboard: `http://IP-HOST:9000`
- Data persisten (config, database JSON, log, upload/download, screenshot)
  disimpan di `docker/data/` di host lewat volume — aman terhadap
  `docker compose down` / rebuild image.
- First run otomatis membuat `docker/data/config/config.yaml` (host
  `0.0.0.0`, port `9000`, `jwt_secret` acak). Container akan langsung siap
  pakai; tidak perlu langkah tambahan.
- Restart otomatis kalau container/Docker/host reboot ditangani oleh
  `restart: unless-stopped` di `docker-compose.yml` — tidak perlu setup
  auto-start terpisah seperti di Windows.
- Ganti port, batasi resource, atau tambah TLS lewat env/volume config
  seperti biasa di `docker-compose.yml`.

Lihat log: `docker logs -f remote-pc-server`. Stop: `docker compose -f docker/docker-compose.yml down`.

**Cek deployment berhasil update:** buka `http://IP-HOST:9000/version` — bandingkan
"Waktu Build" dengan waktu Anda menjalankan `--build`. Kalau mau commit hash
Git juga ikut tampil (opsional, defaultnya "unknown" karena `.git` sengaja
tidak ikut ke context Docker):

```sh
GIT_COMMIT=$(git rev-parse --short HEAD) docker compose -f docker/docker-compose.yml up -d --build
```

## Opsi 2 — Binary Linux langsung (tanpa Docker)

```sh
sh docker/build-linux.sh
./docker/bin/server-linux-amd64
```

First run membuat `config.yaml` di sebelah binary lalu berhenti; jalankan
lagi untuk mulai melayani. Untuk auto-start saat boot, buat unit systemd
sendiri (perintah `enable`/`disable` bawaan hanya berfungsi di Windows).

## Firewall / jaringan

Pastikan port **9000** (TCP) terbuka di host Proxmox/VM agar agent Windows
di LAN yang sama bisa terhubung — perintah `enable` Windows yang otomatis
membuka Windows Firewall tidak berlaku di sini; buka port lewat `ufw`,
`firewalld`, atau aturan firewall Proxmox sesuai distro yang dipakai.
