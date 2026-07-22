# Remote PC — Pemantauan & Kendali Jarak Jauh (RMM)

Aplikasi untuk **memantau dan mengendalikan komputer Windows dari jarak jauh
lewat jaringan lokal (LAN)**. Cocok untuk **lab sekolah**: guru memantau layar
siswa, membuka file, menjalankan perintah, dan mengendalikan mouse/keyboard PC
siswa — semua dari satu dashboard web di browser.

Ditulis sepenuhnya dengan **Go** (tanpa Python/Node/PHP) dan **tanpa database** —
semua data disimpan sebagai file JSON. Ringan, satu file `.exe`, tidak perlu
menginstal apa pun tambahan.

> **Etika & izin:** aplikasi ini untuk lingkungan yang Anda kelola sendiri (lab
> sekolah/kantor) dengan sepengetahuan pemakainya. Gunakan secara bertanggung
> jawab dan sesuai kebijakan institusi Anda.

---

## Apa saja yang bisa dilakukan

- 🖥️ **Dashboard** — daftar semua PC siswa, status online/offline realtime,
  **dikelompokkan otomatis per subnet IP** agar rapi walau PC-nya banyak.
  **HP Android** siswa punya **halaman terpisah** (`/hp`) — tidak pernah
  tercampur baris dengan PC, walau tetap dikelompokkan per subnet di
  antara sesama HP.
- 🗑️ **Hapus device** — tombol hapus langsung dari daftar (dashboard PC
  maupun halaman HP), untuk PC/HP lama yang sudah tidak dipakai.
- ℹ️ **Halaman Versi** (`/version`) — cek commit & waktu build server yang
  sedang berjalan, berguna memastikan rebuild/redeploy benar-benar
  mengambil kode terbaru.
- 🔌 **Auto-discovery** — agent **cukup di-install, langsung menemukan server
  sendiri** di LAN. Tidak perlu mengetik IP/port server (masih bisa diarahkan
  manual bila mau).
- 📊 **System Info** — serial, motherboard, BIOS, CPU, RAM, GPU, adapter jaringan (WMI).
- 📸 **Screenshot** & 🎥 **Live Screen** — lihat layar PC siswa secara langsung
  (bisa fullscreen, kualitas normal/HD).
- 🖱️⌨️ **Remote Mouse & Keyboard** — kendalikan PC siswa dari browser.
- 📁 **File Explorer** — jelajah, unduh, unggah, rename, hapus, buat folder (maks 40 MB/transfer).
- 💻 **Terminal** — jalankan `cmd`/PowerShell dari jarak jauh.
- ⚙️ **Process Manager** — daftar proses + hentikan proses.
- 🔧 **Service Manager** — daftar layanan Windows + Start/Stop/Restart.
- 💬 **Kirim pesan** — tampilkan dialog pesan di layar PC siswa (satu PC, satu
  grup, atau semua sekaligus).
- ⏻ **Shutdown / Restart** jarak jauh — per-PC, **per-grup subnet, atau semua PC
  sekaligus** — dan ⚡ **Wake-on-LAN** (menyalakan PC yang mati).
- 🪟 **Windows 7** juga didukung — agent khusus (`win7/`) untuk PC lama yang belum
  bisa upgrade, fitur identik dengan agent Windows 10/11.
- 📱 **Android (HP siswa, BYOD)** — aplikasi terpisah (`android/`) untuk
  memantau HP pribadi siswa: metrik (RAM/storage/baterai/jaringan), kirim
  pesan, dan Live Screen (siswa wajib izinkan tiap sesi). *Monitor-only* —
  sengaja tanpa kontrol jarak jauh/File Explorer/Terminal, sesuai etika BYOD.

---

## Isi paket

Semua binary **sudah dibangun** dan ada di folder `bin/` — Anda **tidak perlu
meng-compile apa pun**, tinggal pakai:

| File | Untuk | Keterangan |
|------|-------|------------|
| `server-amd64.exe` | **PC guru** (server) | Windows 64-bit (Intel/AMD) |
| `server-386.exe`   | PC guru (server) | Windows 32-bit lama |
| `agent-amd64.exe`  | **PC siswa** (agent) | Windows 64-bit (Intel/AMD) |
| `agent-386.exe`    | PC siswa (agent) | Windows 32-bit lama |
| `install-server.vbs` | PC guru | Installer sekali-klik (auto-start + firewall) |
| `install-agent.vbs`  | PC siswa | Installer sekali-klik (auto-start) |

> Intel & AMD sama-sama x86-64, jadi **`agent-amd64.exe` yang sama** jalan di
> keduanya — tidak perlu binary terpisah. Pakai versi `-386` hanya untuk PC
> Windows 32-bit yang sangat lama.

Untuk PC siswa **Windows 7**, pakai `agent-win7-386.exe`/`agent-win7-amd64.exe`
di `bin/` (lihat bagian *"Alternatif: agent di PC Windows 7"* di bawah). Untuk
**HP Android** siswa, aplikasinya di-build terpisah dari folder `android/` —
tidak ada di tabel di atas karena bukan bagian dari `bin/` Windows (lihat
*"Opsional: agent Android"* di bawah).

---

## Instalasi (cukup "next-next")

Prosesnya cuma **salin file → dobel-klik → klik "Yes"**. Installer `.vbs` otomatis
minta izin Administrator (popup UAC) dan berjalan **senyap tanpa jendela hitam**.

### 1) PC guru (server)

1. Buat folder, mis. `C:\RemotePC\`.
2. Salin ke folder itu: **`server-amd64.exe`** dan **`install-server.vbs`**.
   *(Rename ke `server.exe` opsional — installer mengenali `server-amd64.exe`,
   `server-386.exe`, maupun `server.exe`.)*
3. **Dobel-klik `install-server.vbs`** → klik **"Yes"** pada popup Windows.

   Selesai. Server kini:
   - berjalan tersembunyi di latar belakang,
   - **otomatis menyala setiap PC dinyalakan**,
   - **port firewall dibuka otomatis** agar PC siswa bisa menyambung.

   Muncul dialog "Berhasil".
4. **Catat IP PC guru** (mis. `192.168.1.10`) — dipakai di config agent. Lihat IP
   LAN PC ini lewat `ipconfig`, atau dari log server. Buka dashboard di browser:
   `http://IP-PC-GURU:9000` (atau `http://127.0.0.1:9000` di PC guru sendiri).

> **Catatan first-run:** kalau `config.yaml` belum ada, `install-server.vbs`
> pertama akan **membuat config default** (dengan kunci rahasia acak yang aman)
> lalu berhenti sambil menampilkan dialog. **Dobel-klik `install-server.vbs`
> sekali lagi** untuk benar-benar mengaktifkan auto-start.

### 2) PC siswa (agent)

1. Buat folder, mis. `C:\RemotePC\`.
2. Salin ke folder itu: **`agent-amd64.exe`** dan **`install-agent.vbs`**.
   *(Rename ke `agent.exe` **tidak perlu** — installer mengenali `agent-amd64.exe`,
   `agent-386.exe`, maupun `agent.exe` otomatis.)*
3. **Dobel-klik `install-agent.vbs`** → klik **"Yes"**.

   Saat pertama kali, installer **menanyakan IP server**:
   - **Isi IP server** (mis. `11.11.11.10`) — cara paling andal, selalu berhasil
     selama TCP 9000 terbuka. **Wajib** bila server ada di **Proxmox/Docker** atau
     **beda subnet**.
   - **Kosongkan** → mode **auto-discovery** (agent mencari server sendiri).
     Hanya berhasil bila server & PC siswa berada di **LAN/subnet yang sama** dan
     server **bukan** di dalam Docker.

   Selesai. Agent berjalan tersembunyi, **otomatis aktif setiap siswa login**.
   Dalam beberapa detik PC ini muncul di dashboard guru.

> **Deploy ke banyak PC siswa sekaligus:** buat **satu** `agent.yaml` berisi IP
> server, lalu salin **tiga file** (`agent-amd64.exe` + `agent.yaml` +
> `install-agent.vbs`) ke semua PC. Karena `agent.yaml` sudah ada, installer
> **tidak menanyakan IP lagi** — tiap PC cukup **1× dobel-klik + 1× klik Yes**.
> (Kalau server & semua siswa di LAN yang sama tanpa Docker, cukup dua file saja
> tanpa `agent.yaml` dan kosongkan IP untuk auto-discovery.)

**Cara kerja auto-discovery:** agent menyiarkan pertanyaan "di mana server?" ke
LAN (UDP broadcast), server menjawab dengan alamatnya, lalu agent menyambung.
Alamat IP server dikenali otomatis dari balasan. Kalau server pindah IP atau
belum menyala, agent otomatis mencari ulang.

> ⚠️ **Batasan:** auto-discovery hanya menjangkau **satu segmen LAN (subnet)
> yang sama**. Kalau PC siswa & PC guru dipisah router ke subnet berbeda,
> broadcast tidak menyeberang — untuk kasus itu arahkan agent secara manual
> (lihat di bawah).

**Mengarahkan agent ke IP server tertentu (opsional):** buat file **`agent.yaml`**
di samping `agent.exe` dan ganti `server_host` dari `"auto"` ke IP server:

```yaml
agent:
  server_host: "192.168.1.10"   # "auto" = cari sendiri; isi IP untuk paksa ke server tertentu
  server_port: 9000
  use_tls: false
  device_id: ""                 # dibiarkan kosong — diisi server otomatis
  device_token: ""              # dibiarkan kosong — diisi server otomatis
  reconnect_seconds: 5
  heartbeat_seconds: 5

logging:
  level: "info"
  max_size_mb: 50
  max_backups: 5
  max_age_days: 30
```

*(Template lengkap ada di `config/agent.example.yaml`.)* Setelah mengubah
`agent.yaml`, **restart agent** agar config baru terbaca (restart PC, atau
End task `agent.exe` di Task Manager lalu jalankan lagi). Pindah ke server lain
aman: server baru akan mendaftarkan PC ini sebagai device baru secara otomatis.

### 3) Alternatif: server di Linux (Docker / Proxmox)

Server (dashboard) juga bisa dijalankan di Linux — mis. VM atau LXC container
di Proxmox — lewat Docker, jadi tidak perlu PC Windows khusus untuk server.
**Agent tetap wajib di PC Windows siswa** (butuh WMI, screen capture, kontrol
service Windows); hanya bagian server yang bisa dipindah ke Linux.

```sh
docker compose -f docker/docker-compose.yml up -d --build
```

Detail lengkap (lokasi volume data, build binary Linux tanpa Docker, buka
port firewall di host Linux) ada di [`docker/README.md`](docker/README.md).

### 4) Alternatif: agent di PC Windows 7

PC siswa yang masih Windows 7 (tak bisa/belum upgrade) tetap bisa dipantau —
pakai binary `agent-win7-386.exe` / `agent-win7-amd64.exe` (dibangun dari
modul Go terpisah `win7/`, fitur identik dengan agent Windows 10/11). Rename
jadi `agent-386.exe`/`agent-amd64.exe` sebelum dipakai dengan
`install-agent.vbs` yang sama. Detail & alasan teknis (Go 1.21+ menghapus
dukungan Windows 7) ada di [`win7/README.md`](win7/README.md).

### 5) Opsional: agent Android untuk HP siswa (BYOD)

Kalau lab sekolah juga pakai HP Android siswa (milik pribadi) untuk praktik,
tersedia aplikasi Android *monitor-only* terpisah — HP siswa muncul di
dashboard yang sama, dengan metrik (RAM/storage/baterai/jaringan), kirim
pesan, dan Live Screen (siswa wajib mengetuk & mengizinkan tiap sesi;
Android otomatis menampilkan indikator berbagi layar, tak bisa
disembunyikan). **Sengaja tanpa** kontrol jarak jauh, File Explorer, atau
Terminal — di luar scope untuk perangkat pribadi siswa. Cara build & install
APK ada di [`android/README.md`](android/README.md).

---

## Cara memakai (di dashboard guru)

1. Buka `http://IP-PC-GURU:9000` di browser.
2. Login dengan akun default:
   - Username: **`admin`**
   - Password: **`admin123`** — **segera ganti untuk keamanan.**
3. Di **dashboard**, PC siswa otomatis **dikelompokkan per subnet IP** (mis.
   `192.168.1.0/24`) agar rapi. Tersedia aksi massal tanpa membuka detail:
   - **Header panel** — 💬 **Pesan Semua**, ⟳ **Restart Semua**, ⏻ **Shutdown
     Semua** (berlaku ke seluruh PC online).
   - **Tiap grup subnet** — tombol pesan/restart/shutdown khusus grup itu.
   - **Tiap baris PC** — 💬 kirim pesan, ⏻ shutdown, atau ⚡ Wake (bila offline).
4. Klik salah satu PC siswa untuk membuka **halaman detail**, lalu pilih tab
   sesuai kebutuhan:

| Tab | Fungsi | Catatan |
|-----|--------|---------|
| **Overview** | metrik & identitas realtime | tombol 💬 Pesan / ⟳ Restart / ⏻ Shutdown di kanan atas |
| **Live Screen** | lihat layar; centang **"Aktifkan kendali"** untuk remote mouse/keyboard; tombol fullscreen & pilihan kualitas normal/HD | agent harus di sesi desktop |
| **File Explorer** | jelajah, unduh, unggah, rename, hapus, buat folder | maks **40 MB** per transfer |
| **Terminal** | jalankan cmd/PowerShell realtime | — |
| **Processes** | daftar proses + Kill | kill proses terproteksi perlu admin |
| **Services** | daftar + Start/Stop/Restart | Start/Stop/Restart perlu admin |
| **System Info** | serial, motherboard, BIOS, GPU, adapter (WMI) | — |

### Menyalakan / mematikan PC dari jarak jauh

- **Shutdown / Restart** — tombol **⏻** dan **⟳** di kanan atas halaman detail
  (device harus online). Ada konfirmasi sebelum dieksekusi. Untuk banyak PC
  sekaligus, pakai tombol **Shutdown Semua / Restart Semua** di header dashboard,
  atau tombol per-grup subnet.
- **Kirim pesan** — tombol **💬** menampilkan dialog pesan di layar PC siswa.
  Bisa ke satu PC (dashboard/detail), satu grup subnet, atau semua PC online.
- **Wake-on-LAN** — tombol **⚡ Wake** di dashboard, muncul pada PC yang offline &
  punya MAC tersimpan. **Prasyarat hardware:** WOL harus diaktifkan dulu di
  **BIOS/UEFI** dan **Power Management** kartu jaringan tiap PC target (disetel
  manual sekali langsung di PC itu; tidak bisa diaktifkan dari jarak jauh).

---

## Menghentikan / uninstall

Karena berjalan tanpa jendela, agent/server **tidak bisa** ditutup dengan menutup
jendela. Untuk menonaktifkan auto-start (dan menutup port firewall di server),
jalankan dari folder exe:

```powershell
.\agent.exe disable     # di PC siswa
.\server.exe disable    # di PC guru
```

Perintah `disable` **minta izin Administrator sendiri** (popup UAC). Untuk
menghentikan proses yang sedang jalan, gunakan **Task Manager**. Menghapus total
= `disable` lalu hapus foldernya.

---

## Hal penting yang perlu diketahui

- **Live Screen / Screenshot / Remote Input** hanya berfungsi bila agent berjalan
  **di sesi desktop siswa** — makanya agent dipasang dengan pemicu *saat login*
  (installer `.vbs`), **bukan** sebagai Windows Service (Session 0, tidak bisa
  melihat layar). Monitoring, File Explorer, Terminal, Process, dan Service tetap
  jalan di semua kondisi.
- Agent **baru muncul di dashboard setelah ada siswa yang login** ke PC-nya. PC
  yang menyala tapi belum login akan tampil *offline* (dari sinilah tombol **⚡
  Wake** berguna).
- **`enable`/install cukup dijalankan SEKALI seumur hidup per PC** — bukan tiap
  kali PC nyala. Scheduled Task-nya permanen; PC boleh mati-nyala berkali-kali,
  aplikasi otomatis aktif sendiri.
- Butuh **Administrator (UAC)** saat install karena mendaftarkan auto-start &
  membuka firewall adalah perubahan sistem Windows — bukan keterbatasan program.
- Jaringan: server bind `0.0.0.0:9000`. Pastikan PC guru & siswa **di LAN yang
  sama** dan port **9000** tidak diblokir router. Auto-discovery memakai port
  **UDP 9000** (WebSocket memakai **TCP 9000**); installer server membuka
  keduanya di firewall otomatis.
- **Kalau server sudah pernah di-install sebelum fitur auto-discovery ada,**
  jalankan **`server.exe enable` sekali lagi** (atau dobel-klik
  `install-server.vbs`) agar port **UDP** 9000 ikut dibuka di firewall — tanpa
  itu, siaran pencarian dari agent akan terblokir.

---

## (Lanjutan) Menjalankan manual tanpa auto-start

Kalau tidak mau memasang auto-start, cukup **dobel-klik** exe-nya:

- `server.exe` di PC guru — first-run membuat `config.yaml` lalu berhenti; jalankan
  lagi untuk mulai melayani.
- `agent.exe` di PC siswa — first-run membuat `agent.yaml` (default `server_host:
  "auto"`) lalu **langsung jalan** mencari server di LAN; tidak perlu mengisi IP.

Config dicari **di samping exe** (bukan folder kerja), jadi exe bisa dijalankan
dari mana saja. Keduanya dikompilasi GUI-subsystem sehingga **tidak pernah**
memunculkan jendela terminal; umpan balik lewat **dialog Windows**, log ke `logs/`.

---

## (Untuk developer) Build dari source

Butuh **Go** terpasang. Dari root project:

```powershell
powershell -ExecutionPolicy Bypass -File build.ps1
```

Menghasilkan `bin/server-amd64.exe`, `bin/server-386.exe`, `bin/agent-amd64.exe`,
`bin/agent-386.exe` (dikompilasi `-H=windowsgui`, tanpa cgo).

Menjalankan langsung dari source:

```powershell
go run ./cmd/server      # server
go run ./cmd/agent       # agent
```

> Frontend (HTML/CSS/JS di `web/`) di-`go:embed` ke dalam `server-*.exe` —
> perubahan tampilan perlu **rebuild server**.

### Struktur

```
cmd/server, cmd/agent      entrypoint
internal/config            konfigurasi YAML
internal/logger            zap + rotasi log
internal/model             struct bersama (Device, User, Metrics, ...)
internal/protocol          amplop pesan WebSocket (UUID + tipe)
internal/storage           baca/tulis file JSON (mutex + atomic write)
internal/auth              bcrypt + JWT + middleware + seed admin
internal/server            http, routes, REST API, WebSocket hub + broker
internal/agent             client, heartbeat, sysinfo, screen, input, terminal, dll
internal/autostart         Scheduled Task (auto-start) via schtasks
internal/discovery         auto-discovery server di LAN (UDP broadcast)
internal/wol               Wake-on-LAN (magic packet)
internal/winui             dialog Windows (MessageBox) + elevasi UAC (stub di Linux)
web/                       frontend (HTML + CSS + Vanilla JS), di-embed
docker/                    Dockerfile, docker-compose, build binary Linux (server saja — lihat docker/README.md)
win7/                      modul Go 1.20 terpisah — agent utk Windows 7 (lihat win7/README.md)
android/                   modul Kotlin/Gradle terpisah — agent utk HP Android/BYOD (lihat android/README.md)
```

**Arsitektur singkat:** server memegang WebSocket `Hub` (RPC ber-UUID) + `Broker`
(stream ke browser operator) dan relay `/ws/operator`. Agent menerima perintah,
merutekannya ke paket `sysinfo/fsops/procs/winservices/screen/input/terminal`.
Frontend per-tab: `web/static/js/tab-*.js`.
