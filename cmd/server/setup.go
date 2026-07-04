package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"remote_pc/internal/autostart"
	"remote_pc/internal/config"
	"remote_pc/internal/winui"
)

// createNoWindow mencegah jendela console muncul saat menjalankan program
// console (netsh) dari exe GUI-subsystem ini.
const createNoWindow = 0x08000000

// taskName adalah nama scheduled task auto-start server.
const taskName = "RemotePCServer"

// firewallRuleName adalah nama rule Windows Firewall yang dibuat/dihapus enable/disable.
const firewallRuleName = "Remote PC Server"

// exeDir mengembalikan folder tempat exe berada. Dipakai agar config dan storage
// dicari relatif terhadap exe, bukan folder kerja (yang bisa berupa System32 saat
// dijalankan oleh Task Scheduler sebagai SYSTEM).
func exeDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

// resolveConfigPath menentukan path config absolut. Bila -config diberikan, path
// itu dipakai apa adanya. Bila tidak, server mencari config.yaml di samping exe
// (lalu config/config.yaml sebagai cadangan untuk mode pengembangan).
func resolveConfigPath(flagVal string) string {
	if flagVal != "" {
		if abs, err := filepath.Abs(flagVal); err == nil {
			return abs
		}
		return flagVal
	}

	dir := exeDir()
	candidates := []string{
		filepath.Join(dir, "config.yaml"),
		filepath.Join(dir, "config", "config.yaml"),
		filepath.Join("config", "config.yaml"),
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

// ensureConfig membuat template config bila belum ada, dengan jwt_secret acak
// (bukan placeholder statis) agar aman dipakai langsung tanpa langkah manual
// tambahan. Mengembalikan true bila file baru saja dibuat.
func ensureConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	secret, err := randomSecret()
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf(configTemplate, secret)), 0o600); err != nil {
		return false, err
	}
	return true, nil
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("buat jwt_secret acak: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// configTemplate adalah isi awal config.yaml yang dibuat otomatis bila belum ada.
// %s diisi jwt_secret acak yang dibuat sekali saat template dibuat.
const configTemplate = `# Konfigurasi server Remote PC.
# host 0.0.0.0 membuat server mendengar di semua alamat jaringan PC ini, agar PC
# siswa (agent) di LAN yang sama bisa terhubung. Jangan ganti ke 127.0.0.1 kecuali
# hanya menguji di PC ini sendiri.

server:
  host: "0.0.0.0"
  port: 7000
  tls:
    enabled: false
    cert_file: ""
    key_file: ""

auth:
  # Dibuat otomatis & acak saat file ini pertama kali dibuat — aman dipakai
  # langsung, tidak perlu diganti manual.
  jwt_secret: "%s"
  jwt_expiry_hours: 24

storage:
  data_dir: "data"
  logs_dir: "logs"
  screenshots_dir: "screenshots"
  uploads_dir: "uploads"
  downloads_dir: "downloads"

heartbeat:
  interval_seconds: 5
  offline_after_seconds: 15

logging:
  level: "info"
  max_size_mb: 50
  max_backups: 5
  max_age_days: 30
`

// enableAutostart mendaftarkan server agar berjalan otomatis saat Windows boot
// (sebagai SYSTEM, sebelum ada yang login), membuka port di Windows Firewall,
// lalu menjalankannya sekarang juga. Meminta elevasi (UAC) sendiri bila belum
// admin, sehingga tidak perlu membuka terminal admin manual.
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
				"\n\nNilai default sudah aman. Jalankan installer server lagi untuk mengaktifkan auto-start.", false)
		return
	}

	cfg, err := config.LoadServerConfig(configPath)
	if err != nil {
		winui.MessageBox(appName, "Gagal membaca konfigurasi:\n"+err.Error(), true)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		winui.MessageBox(appName, "Gagal menemukan lokasi program:\n"+err.Error(), true)
		return
	}

	opts := autostart.Options{
		Trigger:     autostart.TriggerBoot,
		Description: "Server Remote PC - berjalan otomatis saat Windows menyala.",
	}
	if err := autostart.Install(taskName, exe, configPath, opts); err != nil {
		winui.MessageBox(appName, "Gagal mengaktifkan auto-start:\n"+err.Error(), true)
		return
	}

	fw := fmt.Sprintf("Port %d dibuka di Windows Firewall (semua profil).", cfg.Server.Port)
	if err := addFirewallRule(cfg.Server.Port); err != nil {
		fw = fmt.Sprintf("Catatan: gagal membuka port %d di firewall otomatis (%v).\n"+
			"Buka manual bila perlu.", cfg.Server.Port, err)
	}
	_ = autostart.Run(taskName) // langsung jalankan (best-effort)

	winui.MessageBox(appName,
		"Berhasil! Server kini berjalan tersembunyi di latar belakang dan akan "+
			"otomatis menyala setiap Windows boot.\n\n"+fw, false)
}

// disableAutostart menghapus task auto-start dan rule firewall yang dibuat enable.
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
	_ = removeFirewallRule()
	winui.MessageBox(appName, "Auto-start server dinonaktifkan & port firewall ditutup.", false)
}

// addFirewallRule membuka port TCP inbound di semua profil Windows Firewall.
// Butuh hak Administrator; wajar gagal bila dijalankan tanpa elevasi.
func addFirewallRule(port int) error {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name="+firewallRuleName,
		"dir=in",
		"action=allow",
		"protocol=TCP",
		fmt.Sprintf("localport=%d", port),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("netsh gagal: %v: %s", err, string(out))
	}
	return nil
}

// removeFirewallRule menghapus rule yang dibuat addFirewallRule.
func removeFirewallRule() error {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule",
		"name="+firewallRuleName,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("netsh gagal: %v: %s", err, string(out))
	}
	return nil
}
