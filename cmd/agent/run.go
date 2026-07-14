package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"

	"go.uber.org/zap"

	"remote_pc/internal/agent"
	"remote_pc/internal/agent/winsvc"
	"remote_pc/internal/config"
	"remote_pc/internal/logger"
	"remote_pc/internal/winui"
)

// newRun membangun fungsi kerja agent yang dipakai baik oleh mode konsol maupun
// service. Log selalu ke file (bukan console) karena exe dikompilasi sebagai
// aplikasi GUI-subsystem tanpa jendela.
func newRun(configPath string) winsvc.RunFunc {
	return func(ctx context.Context) error {
		cfg, err := config.LoadAgentConfig(configPath)
		if err != nil {
			return err
		}
		// Log ditulis di folder yang sama dengan exe agar konsisten walau folder
		// kerja berbeda (mis. Task Scheduler menjalankan dari System32).
		logs, err := logger.New(logger.Options{
			Dir:        filepath.Join(exeDir(), "logs"),
			MainFile:   "client.log",
			Level:      cfg.Logging.Level,
			MaxSizeMB:  cfg.Logging.MaxSizeMB,
			MaxBackups: cfg.Logging.MaxBackups,
			MaxAgeDays: cfg.Logging.MaxAgeDays,
			Console:    false,
		})
		if err != nil {
			return err
		}
		defer logs.Sync()

		logs.App.Info("agent dimulai", zap.String("server", cfg.Agent.ServerURL))
		client := agent.New(cfg, logs.App)
		return client.Run(ctx)
	}
}

// runConsole menjalankan agent di latar belakang tanpa jendela. Bila config belum
// ada, dibuatkan otomatis dengan default "auto" (auto-discovery) lalu LANGSUNG
// jalan — tidak perlu diisi IP server dulu.
func runConsole(configPath string) {
	if _, err := ensureConfig(configPath); err != nil {
		winui.MessageBox(appName, "Gagal menyiapkan konfigurasi:\n"+err.Error(), true)
		return
	}

	// Berjalan langsung di proses ini tanpa jendela (exe GUI-subsystem). Tidak
	// ada terminal yang bisa ditutup siswa; hanya bisa dihentikan lewat Task
	// Manager atau perintah disable.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := newRun(configPath)(ctx); err != nil && ctx.Err() == nil {
		winui.MessageBox(appName, "Agent berhenti dengan error:\n"+err.Error(), true)
	}
}

// runAsService menjalankan agent di bawah SCM. Direktori kerja dipindah ke folder
// eksekutabel agar path relatif (logs) tetap konsisten.
func runAsService(configPath string) {
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}
	_ = winsvc.Run(serviceName, newRun(absConfig(configPath)))
}

// installService memasang service dengan path konfigurasi absolut agar dapat
// ditemukan saat berjalan dari direktori System32.
func installService(configPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return winsvc.Install(serviceName, serviceDisplay, serviceDesc, exe,
		"-config", absConfig(configPath))
}

func absConfig(configPath string) string {
	if abs, err := filepath.Abs(configPath); err == nil {
		return abs
	}
	return configPath
}
