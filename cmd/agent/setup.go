package main

import (
	"os"
	"path/filepath"

	"remote_pc/internal/autostart"
	"remote_pc/internal/winui"
)

// taskName adalah nama scheduled task auto-start (terpisah dari service).
const taskName = "RemotePCAgent"

// appName adalah judul dialog yang ditampilkan ke user.
const appName = "Remote PC Agent"

// configTemplate adalah isi awal agent.yaml yang dibuat otomatis bila belum ada.
// User cukup mengganti server_host dan server_port.
const configTemplate = `# Konfigurasi agent Remote PC.
# Isi server_host dengan IP komputer SERVER (PC guru) dan server_port dengan port
# server (default 7000). IP server tercetak di layar server saat dinyalakan.
#
# device_id & device_token BIARKAN KOSONG - server mengisinya otomatis saat agent
# pertama kali registrasi, lalu tersimpan kembali ke file ini.

agent:
  server_host: "192.168.1.10"   # GANTI dengan IP PC server (guru)
  server_port: 7000
  use_tls: false
  device_id: ""
  device_token: ""
  reconnect_seconds: 5
  heartbeat_seconds: 5

logging:
  level: "info"
  max_size_mb: 50
  max_backups: 5
  max_age_days: 30
`

// exeDir mengembalikan folder tempat exe berada. Dipakai agar config dan logs
// dicari relatif terhadap exe, bukan folder kerja (yang bisa berupa System32 saat
// dijalankan oleh Task Scheduler).
func exeDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

// resolveConfigPath menentukan path config absolut. Bila -config diberikan, path
// itu dipakai apa adanya. Bila tidak, agent mencari agent.yaml di samping exe
// (lalu config/agent.yaml sebagai cadangan untuk mode pengembangan).
func resolveConfigPath(flagVal string) string {
	if flagVal != "" {
		if abs, err := filepath.Abs(flagVal); err == nil {
			return abs
		}
		return flagVal
	}

	dir := exeDir()
	candidates := []string{
		filepath.Join(dir, "agent.yaml"),
		filepath.Join(dir, "config", "agent.yaml"),
		filepath.Join("config", "agent.yaml"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			if abs, e := filepath.Abs(c); e == nil {
				return abs
			}
			return c
		}
	}
	// Belum ada file mana pun: pakai lokasi utama di samping exe (akan dibuat
	// sebagai template saat pertama kali dijalankan).
	return candidates[0]
}

// ensureConfig membuat template config bila belum ada. Mengembalikan true bila
// file baru saja dibuat.
func ensureConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(configTemplate), 0o600); err != nil {
		return false, err
	}
	return true, nil
}

// enableAutostart mendaftarkan agent agar berjalan otomatis saat login, lalu
// menjalankannya sekarang juga. Bila belum admin, minta elevasi (UAC) sendiri
// sehingga tidak perlu membuka terminal admin manual.
func enableAutostart(configPath string) {
	if !winui.IsAdmin() {
		if err := winui.RunSelfElevated("enable"); err != nil {
			winui.MessageBox(appName, "Persetujuan Administrator dibutuhkan untuk mengaktifkan "+
				"auto-start, tetapi dibatalkan atau gagal:\n"+err.Error(), true)
		}
		return
	}

	created, err := ensureConfig(configPath)
	if err != nil {
		winui.MessageBox(appName, "Gagal menyiapkan konfigurasi:\n"+err.Error(), true)
		return
	}
	if created {
		winui.MessageBox(appName,
			"Konfigurasi baru dibuat di:\n"+configPath+
				"\n\nIsi dulu server_host (IP server) dan server_port, lalu jalankan installer lagi.", false)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		winui.MessageBox(appName, "Gagal menemukan lokasi program:\n"+err.Error(), true)
		return
	}
	opts := autostart.Options{
		Trigger:     autostart.TriggerLogon,
		Description: "Agent Remote PC - berjalan otomatis saat user login Windows.",
	}
	if err := autostart.Install(taskName, exe, configPath, opts); err != nil {
		winui.MessageBox(appName, "Gagal mengaktifkan auto-start:\n"+err.Error(), true)
		return
	}
	_ = autostart.Run(taskName) // langsung jalankan (best-effort)
	winui.MessageBox(appName,
		"Berhasil! Agent kini berjalan tersembunyi di latar belakang dan akan "+
			"otomatis aktif setiap Windows login.\n\nTidak ada jendela yang perlu dibiarkan terbuka.", false)
}

// disableAutostart menghapus auto-start agent. Meminta elevasi bila perlu.
func disableAutostart() {
	if !winui.IsAdmin() {
		if err := winui.RunSelfElevated("disable"); err != nil {
			winui.MessageBox(appName, "Persetujuan Administrator dibutuhkan:\n"+err.Error(), true)
		}
		return
	}
	if err := autostart.Uninstall(taskName); err != nil {
		winui.MessageBox(appName, "Gagal menonaktifkan auto-start:\n"+err.Error(), true)
		return
	}
	winui.MessageBox(appName, "Auto-start agent dinonaktifkan.", false)
}
