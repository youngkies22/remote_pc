package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

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
  port: 9000
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
