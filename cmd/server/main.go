// Command server menjalankan HTTP/WebSocket server dashboard Remote PC.
//
// Penggunaan:
//
//	server.exe                          jalankan mode konsol
//	server.exe -config path             jalankan dengan config tertentu
//	server.exe enable                   aktifkan auto-start saat Windows boot
//	                                     + buka port di Windows Firewall (disarankan)
//	server.exe disable                  nonaktifkan auto-start & tutup port firewall
//
// Bila -config tidak diberikan, server memakai config.yaml di folder yang sama
// dengan exe (dibuatkan otomatis dengan jwt_secret acak bila belum ada).
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"remote_pc/internal/auth"
	"remote_pc/internal/config"
	"remote_pc/internal/logger"
	"remote_pc/internal/server"
	"remote_pc/internal/storage"
	"remote_pc/internal/winui"
)

// appName adalah judul dialog yang ditampilkan ke user.
const appName = "Remote PC Server"

func main() {
	configFlag := flag.String("config", "", "path file konfigurasi server (default: config.yaml di samping exe)")
	flag.Parse()

	configPath := resolveConfigPath(*configFlag)

	switch flag.Arg(0) {
	case "enable":
		enableAutostart(configPath)
		return
	case "disable":
		disableAutostart()
		return
	}

	created, err := ensureConfig(configPath)
	if err != nil {
		winui.MessageBox(appName, "Gagal menyiapkan konfigurasi:\n"+err.Error(), true)
		return
	}
	if created {
		winui.MessageBox(appName,
			"File konfigurasi baru dibuat di:\n"+configPath+
				"\n\nNilai default (host 0.0.0.0, port 7000, jwt_secret acak) sudah aman "+
				"dipakai langsung. Jalankan server lagi untuk mulai.\n\n"+
				"(Opsional) Jalankan installer server sebagai admin agar server otomatis "+
				"menyala tiap PC dinyalakan + firewall dibuka otomatis.", false)
		return
	}

	// Exe GUI-subsystem: berjalan langsung tanpa jendela console.
	if err := run(configPath); err != nil {
		winui.MessageBox(appName, "Server berhenti dengan error:\n"+err.Error(), true)
	}
}

func run(configPath string) error {
	cfg, err := config.LoadServerConfig(configPath)
	if err != nil {
		return err
	}

	logs, err := logger.New(logger.Options{
		Dir:        cfg.Storage.LogsDir,
		MainFile:   "server.log",
		Level:      cfg.Logging.Level,
		MaxSizeMB:  cfg.Logging.MaxSizeMB,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAgeDays: cfg.Logging.MaxAgeDays,
		Console:    true,
	})
	if err != nil {
		return err
	}
	defer logs.Sync()

	store, err := storage.Open(cfg.Storage.DataDir)
	if err != nil {
		return err
	}

	created, err := auth.EnsureDefaultAdmin(store.Users)
	if err != nil {
		return err
	}
	if created {
		logs.App.Warn("user admin default dibuat (admin/admin123) — SEGERA ganti password di produksi")
	}

	srv, err := server.New(cfg, store, logs)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return srv.Run(ctx)
}
