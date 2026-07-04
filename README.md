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

- 🖥️ **Dashboard** — daftar semua PC siswa, status online/offline realtime.
- 📊 **System Info** — serial, motherboard, BIOS, CPU, RAM, GPU, adapter jaringan (WMI).
- 📸 **Screenshot** & 🎥 **Live Screen** — lihat layar PC siswa secara langsung
  (bisa fullscreen, kualitas normal/HD).
- 🖱️⌨️ **Remote Mouse & Keyboard** — kendalikan PC siswa dari browser.
- 📁 **File Explorer** — jelajah, unduh, unggah, rename, hapus, buat folder (maks 40 MB/transfer).
- 💻 **Terminal** — jalankan `cmd`/PowerShell dari jarak jauh.
- ⚙️ **Process Manager** — daftar proses + hentikan proses.
- 🔧 **Service Manager** — daftar layanan Windows + Start/Stop/Restart.
- ⏻ **Shutdown / Restart** jarak jauh, dan ⚡ **Wake-on-LAN** (menyalakan PC yang mati).

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

---

## Instalasi (cukup "next-next")

Prosesnya cuma **salin file → dobel-klik → klik "Yes"**. Installer `.vbs` otomatis
minta izin Administrator (popup UAC) dan berjalan **senyap tanpa jendela hitam**.

### 1) PC guru (server)

1. Buat folder, mis. `C:\RemotePC\`.
2. Salin ke folder itu: **`server-amd64.exe`** dan **`install-server.vbs`**.
3. **Ganti nama** `server-amd64.exe` → **`server.exe`** *(installer mencari nama `server.exe`)*.
4. **Dobel-klik `install-server.vbs`** → klik **"Yes"** pada popup Windows.

   Selesai. Server kini:
   - berjalan tersembunyi di latar belakang,
   - **otomatis menyala setiap PC dinyalakan**,
   - **port firewall dibuka otomatis** agar PC siswa bisa menyambung.

   Muncul dialog "Berhasil".
5. **Catat IP PC guru** (mis. `192.168.1.10`) — dipakai di config agent. Lihat IP
   LAN PC ini lewat `ipconfig`, atau dari log server. Buka dashboard di browser:
   `http://IP-PC-GURU:7000` (atau `http://127.0.0.1:7000` di PC guru sendiri).

> **Catatan first-run:** kalau `config.yaml` belum ada, `install-server.vbs`
> pertama akan **membuat config default** (dengan kunci rahasia acak yang aman)
> lalu berhenti sambil menampilkan dialog. **Dobel-klik `install-server.vbs`
> sekali lagi** untuk benar-benar mengaktifkan auto-start.

### 2) PC siswa (agent)

1. Buat folder, mis. `C:\RemotePC\`.
2. Salin ke folder itu: **`agent-amd64.exe`** dan **`install-agent.vbs`**.
3. **Ganti nama** `agent-amd64.exe` → **`agent.exe`**.
4. Buat file **`agent.yaml`** di folder yang sama, isi IP PC guru:

   ```yaml
   agent:
     server_host: "192.168.1.10"   # GANTI dengan IP PC guru (server)
     server_port: 7000
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

   *(Template lengkap ada di `config/agent.example.yaml`.)*
5. **Dobel-klik `install-agent.vbs`** → klik **"Yes"**.

   Selesai. Agent kini berjalan tersembunyi dan **otomatis aktif setiap siswa
   login**. Dalam beberapa detik PC ini muncul di dashboard guru.

> **Deploy ke banyak PC siswa sekaligus:** siapkan **satu** `agent.yaml` berisi
> IP server yang benar, lalu salin **tiga file yang sama** (`agent.exe` +
> `agent.yaml` + `install-agent.vbs`) ke semua PC siswa. Tiap PC cuma perlu
> **1× dobel-klik + 1× klik Yes** — tak perlu edit config satu per satu.

---

## Cara memakai (di dashboard guru)

1. Buka `http://IP-PC-GURU:7000` di browser.
2. Login dengan akun default:
   - Username: **`admin`**
   - Password: **`admin123`** — **segera ganti untuk keamanan.**
3. Klik salah satu PC siswa di dashboard untuk membuka **halaman detail**, lalu
   pilih tab sesuai kebutuhan:

| Tab | Fungsi | Catatan |
|-----|--------|---------|
| **Overview** | metrik & identitas realtime | tombol ⏻ Shutdown / ⟳ Restart di kanan atas |
| **Live Screen** | lihat layar; centang **"Aktifkan kendali"** untuk remote mouse/keyboard; tombol fullscreen & pilihan kualitas normal/HD | agent harus di sesi desktop |
| **File Explorer** | jelajah, unduh, unggah, rename, hapus, buat folder | maks **40 MB** per transfer |
| **Terminal** | jalankan cmd/PowerShell realtime | — |
| **Processes** | daftar proses + Kill | kill proses terproteksi perlu admin |
| **Services** | daftar + Start/Stop/Restart | Start/Stop/Restart perlu admin |
| **System Info** | serial, motherboard, BIOS, GPU, adapter (WMI) | — |

### Menyalakan / mematikan PC dari jarak jauh

- **Shutdown / Restart** — tombol **⏻** dan **⟳** di kanan atas halaman detail
  (device harus online). Ada konfirmasi sebelum dieksekusi.
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
- Jaringan: server bind `0.0.0.0:7000`. Pastikan PC guru & siswa **di LAN yang
  sama** dan port **7000** tidak diblokir router.

---

## (Lanjutan) Menjalankan manual tanpa auto-start

Kalau tidak mau memasang auto-start, cukup **dobel-klik** exe-nya:

- `server.exe` di PC guru — first-run membuat `config.yaml` lalu berhenti; jalankan
  lagi untuk mulai melayani.
- `agent.exe` di PC siswa — first-run membuat `agent.yaml` lalu berhenti; isi IP
  server, jalankan lagi untuk menyambung.

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
internal/wol               Wake-on-LAN (magic packet)
internal/winui             dialog Windows (MessageBox) + elevasi UAC
web/                       frontend (HTML + CSS + Vanilla JS), di-embed
```

**Arsitektur singkat:** server memegang WebSocket `Hub` (RPC ber-UUID) + `Broker`
(stream ke browser operator) dan relay `/ws/operator`. Agent menerima perintah,
merutekannya ke paket `sysinfo/fsops/procs/winservices/screen/input/terminal`.
Frontend per-tab: `web/static/js/tab-*.js`.
