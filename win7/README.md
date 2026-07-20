# Agent untuk Windows 7 (client saja)

Folder ini berisi **agent (client) versi khusus Windows 7**, sebagai **modul Go
terpisah** dari proyek utama. Server **tidak** ada di sini — server tetap dibangun
dari proyek utama (Go 1.25) dan tidak perlu diubah.

## Kenapa harus terpisah?

Sejak **Go 1.21, Google menghapus dukungan Windows 7/8/Server 2012**. Biner apa pun
yang dibuat dengan Go >= 1.21 (termasuk agent utama yang pakai Go 1.25) **crash
langsung saat start di Windows 7** — bukan salah kode, tapi runtime Go-nya.

Versi Go **terakhir** yang mendukung Windows 7 adalah **Go 1.20.14**. Karena
beberapa dependency proyek utama menuntut Go modern, folder ini punya `go.mod`
sendiri (`go 1.20`) dengan dependency yang **diturunkan** ke versi yang aman di
Go 1.20:

| Dependency | Proyek utama | Versi Win7 (di sini) | Alasan turun |
|---|---|---|---|
| toolchain Go | 1.25 | **1.20.14** | 1.21+ buang Win7 |
| `gopsutil` | v4.26.6 | **v3.23.12** | v4 pakai paket `slices` (Go 1.21) |
| `golang.org/x/sys` | v0.46.0 | **v0.24.0** | v0.31+ butuh Go 1.23/1.24/1.25 |
| `tklauser/go-sysconf` | v0.3.16 | **v0.3.12** | versi baru menyeret x/sys naik |
| `tklauser/numcpus` | v0.11.0 | **v0.6.1** | idem |

Kode agent-nya **disalin dari sumber yang sama** dengan proyek utama, jadi fitur
identik (SysInfo, Screenshot, File Explorer, Terminal, Live Screen, Mouse/Keyboard,
Processes, Services, Power, Pesan, Auto-discovery, Autostart). Semua Win32 API yang
dipakai memang tersedia di Windows 7 — penghalang satu-satunya cuma versi Go.

## Cara build

Dari folder `win7/`:

```powershell
powershell -ExecutionPolicy Bypass -File build-win7.ps1
```

Hasilnya di folder `bin/` (satu tingkat di atas):

- **`agent-win7-386.exe`** — 32-bit, **jalan di Windows 7 32-bit MAUPUN 64-bit**.
  Ini yang dipakai untuk sebagian besar kasus.
- `agent-win7-amd64.exe` — 64-bit, opsional (khusus Win7 64-bit, sedikit lebih
  optimal). Kalau ragu, cukup pakai yang `386`.

Skrip memakai `GOTOOLCHAIN=go1.20.14`, jadi `go` otomatis mengunduh & memakai
toolchain 1.20.14 walau Go yang terpasang versi baru — tidak perlu install Go 1.20
manual.

## Cara deploy ke PC siswa Windows 7

Sama seperti agent biasa, TAPI installer `.vbs` mengenali nama file
`agent.exe` / `agent-amd64.exe` / `agent-386.exe` — **bukan** `agent-win7-*.exe`.
Jadi **rename dulu**:

1. `agent-win7-386.exe` → ganti nama jadi **`agent-386.exe`** (atau `agent.exe`).
2. Copy bertiga ke PC siswa Win7: `agent-386.exe` + `agent.yaml` (isi `server_host`
   IP server yang benar) + `bin/install-agent.vbs`.
3. Di PC siswa: dobel-klik `install-agent.vbs`, klik **Yes** di popup UAC.

Karena `386` jalan di 32-bit & 64-bit, satu file ini cukup untuk semua PC Win7.

## Batasan penting (WAJIB diperhatikan)

- **Runtime lama = tanpa patch keamanan terbaru.** Go 1.20 & dependency di sini
  sudah tidak menerima update keamanan. Ini **trade-off yang tak terhindarkan**
  demi jalan di Win7. Karena itu: **PC Windows 10/11 tetap pakai agent utama**
  (proyek induk), **hanya PC Windows 7 yang pakai build ini**.
- **Belum diuji di mesin Windows 7 nyata.** Build sudah terverifikasi (compile
  bersih, PE i386, subsystem GUI, `go vet` bersih), tapi "benar jalan di Win7"
  harus dites langsung di PC Windows 7 asli.
- **Ini salinan point-in-time.** Kalau nanti ada perubahan/penambahan fitur di
  agent utama, kode di `win7/` **tidak ikut berubah otomatis** — harus disalin
  ulang dari sumbernya lalu di-build lagi dengan skrip ini.
- Batasan fungsional lain (Session 0, kill proses terproteksi butuh admin, file
  transfer maks 40 MB, auto-discovery hanya satu subnet) **sama** dengan agent
  utama.
