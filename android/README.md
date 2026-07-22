# RemotePC Agent — Android (Tahap A1–A4)

Aplikasi Android *monitor-only* untuk HP siswa (BYOD), menyambung ke server
RemotePC Go yang sama dipakai agent Windows. Native Kotlin, modul Gradle
terpisah — tidak memengaruhi build server/agent Windows.

## Status

### A1 — Fondasi (terverifikasi di HP fisik nyata)
- Layar setup: nama perangkat (nickname), IP/host server, port.
- Foreground service (`AgentService`) menjaga koneksi WebSocket ke
  `ws://<host>:<port>/ws/agent`, registrasi, dan heartbeat berkala (5 detik) —
  protokolnya identik dengan `internal/agent` (Go), server tidak perlu tahu
  bedanya.
- Reconnect otomatis tiap 5 detik bila putus.
- Notifikasi permanen "RemotePC — terhubung ke guru" dengan tombol
  **Putuskan** (siswa bisa hentikan kapan saja — ini HP pribadi siswa, bukan
  device sekolah yang di-provisioning).
- Field `os`/`username`/`windows_version` diisi kreatif: `os="Android"`,
  `username` = merk+model HP, `windows_version` dipakai ulang untuk versi
  Android — supaya dashboard yang sudah ada tampil apa adanya tanpa ubah UI.

### A2 — Metrik
- RAM, storage, uptime (native, akurat).
- Baterai: field baru `battery_percent` di `internal/model/device.go`
  (opsional/omitempty — agent Windows tak terpengaruh).
- Jenis jaringan: field baru `network_type` ("wifi"/"cellular"/"none").
- **CPU percent selalu 0** — Android melarang aplikasi biasa membaca
  `/proc/stat` sistem sejak Android 8 tanpa root, beda dari Windows yang bebas
  lewat WMI. Ini keterbatasan permanen, bukan "belum sempat".

### A3 — Pesan dari guru
- Agent menerima `TypeMessage` (sama seperti Windows) dan menampilkannya
  sebagai **notifikasi heads-up** (channel `remotepc_alert`, importance HIGH).
  Windows pakai dialog modal blocking; Android tidak bisa memunculkan dialog
  dari background service secara andal (dibatasi OS), jadi notifikasi menonjol
  adalah padanan yang paling reliable.
- Tombol 💬 Pesan di dashboard & device detail kini aktif untuk device Android.

### A4 — Live Screen (opsional, siswa wajib izinkan tiap sesi)
- Guru buka tab "Live Screen" → server otomatis kirim `screen.start` ke agent
  (persis alur Windows, lihat `internal/server/ws/operator.go`).
- Kalau belum ada izin MediaProjection aktif, agent memposting notifikasi
  heads-up **"Guru ingin melihat layar Anda"**. Siswa **wajib mengetuk**
  notifikasi itu → membuka `ScreenCaptureActivity` (activity tak berujud) →
  memicu dialog sistem Android "Mulai merekam/menyiarkan layar?" → siswa pilih
  "Mulai sekarang".
- Setelah diizinkan: `MediaProjection` + `VirtualDisplay` + `ImageReader`
  menangkap layar tiap 300ms, di-encode JPEG (kualitas 60 normal / 92 HD,
  mengikuti `screen.quality` dari guru — sama seperti Windows), dikirim
  sebagai `screen.frame` lewat koneksi `/ws/agent` yang sama.
  Server merelainya ke browser guru lewat `Broker` yang sudah ada — **tidak
  ada perubahan server untuk fitur ini**.
- `screen.stop` (guru menutup tab) → capture dihentikan **dan token
  MediaProjection dilepas** — sesi berikutnya wajib izin ulang. Ini sengaja
  dibuat ketat (bukan reuse token) supaya perilakunya konsisten & sesuai janji
  ke siswa: "wajib izinkan tiap sesi", bukan sekali untuk selamanya.
- Selama capture aktif, Android **otomatis** menampilkan indikator sistem
  "sedang membagikan layar" — tidak bisa disembunyikan, dan itu memang
  disengaja (transparansi ke siswa).
- **Tidak ada** kontrol mouse/keyboard jarak jauh (di luar scope BYOD, tidak
  direncanakan).

### Halaman detail device (`device.html`) disesuaikan per OS
Untuk device Android: tab **File Explorer/Terminal/Processes/Services/
System Info** dan tombol **Restart/Shutdown** disembunyikan (agent Android
tidak & tidak akan mendukungnya — di luar scope monitor-only). Tab **Overview**
dan **Live Screen** tetap tampil karena keduanya berfungsi penuh.

## Build

Butuh Android SDK (compileSdk 35) — sudah terpasang di mesin ini via Android
Studio. Build dari folder ini:

```
JAVA_HOME="/c/Program Files/Android/Android Studio/jbr" ./gradlew assembleDebug
```

APK debug ada di `app/build/outputs/apk/debug/app-debug.apk`.

## Install & uji ke HP

1. Aktifkan **Opsi Pengembang → Debugging USB** di HP siswa (Settings →
   About phone → tap "Nomor build" 7x → Opsi Pengembang muncul).
2. Colok HP via USB, izinkan "Trust this computer" di HP.
3. Cek terdeteksi: `adb devices`
4. Install: `adb install -r app/build/outputs/apk/debug/app-debug.apk`
5. Buka app "RemotePC Siswa", isi IP server & port (default `9000`), tekan
   Hubungkan.
6. Cek dashboard guru — device muncul **online** dengan `os: Android`.
7. Uji Pesan: dari dashboard/device detail klik 💬, tulis pesan → notifikasi
   muncul di HP.
8. Uji Live Screen: buka tab "Live Screen" di device detail → di HP muncul
   notifikasi "Guru ingin melihat layar Anda" → ketuk → izinkan di dialog
   sistem → layar HP mulai tampil di browser guru.

Catatan: server harus `ws://` (bukan `wss://`) untuk LAN sekolah biasa —
`network_security_config.xml` sudah mengizinkan cleartext traffic khusus
untuk app ini. Perubahan `device.js`/`dashboard.js` baru terlihat setelah
`server-*.exe` di-rebuild ulang (`build.ps1`) karena frontend di-`go:embed`.
