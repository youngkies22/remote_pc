// Command agent adalah client Remote PC yang berjalan di komputer target.
// Dapat dijalankan sebagai aplikasi konsol, auto-start saat login, maupun
// Windows Service.
//
// Penggunaan:
//
//	agent.exe                           jalankan mode konsol
//	agent.exe -config path              jalankan dengan config tertentu
//	agent.exe enable                    aktifkan auto-start saat login (disarankan)
//	agent.exe disable                   nonaktifkan auto-start
//	agent.exe install                   pasang sebagai Windows Service (lanjutan)
//	agent.exe uninstall                 hapus service
//	agent.exe start | stop              kontrol service
//
// Bila -config tidak diberikan, agent memakai agent.yaml di folder yang sama
// dengan exe. Auto-start (enable) berjalan di sesi desktop user sehingga fitur
// screenshot, live screen, dan remote input tetap berfungsi (berbeda dengan
// Windows Service yang terisolasi di Session 0).
package main

import (
	"flag"

	"remote_pc/internal/agent/winsvc"
	"remote_pc/internal/winui"
)

const (
	serviceName    = "RemotePCAgent"
	serviceDisplay = "Remote PC Agent"
	serviceDesc    = "Agent monitoring & remote management Remote PC."
)

func main() {
	// Default kosong: bila tidak diisi, resolveConfigPath mencari agent.yaml di
	// samping exe (bukan relatif folder kerja) — inilah yang memperbaiki masalah
	// agent di bin/ selalu tersambung ke localhost:7000.
	configFlag := flag.String("config", "", "path file konfigurasi agent (default: agent.yaml di samping exe)")
	flag.Parse()

	configPath := resolveConfigPath(*configFlag)

	// Bila diluncurkan oleh Service Control Manager, jalankan sebagai service.
	if isSvc, err := winsvc.IsService(); err == nil && isSvc {
		runAsService(configPath)
		return
	}

	switch flag.Arg(0) {
	case "enable":
		enableAutostart(configPath)
	case "disable":
		disableAutostart()
	case "install":
		serviceResult(installService(configPath), "Service terpasang.")
	case "uninstall":
		serviceResult(winsvc.Uninstall(serviceName), "Service dihapus.")
	case "start":
		serviceResult(winsvc.Control(serviceName, true), "Service dijalankan.")
	case "stop":
		serviceResult(winsvc.Control(serviceName, false), "Service dihentikan.")
	default:
		runConsole(configPath)
	}
}

// serviceResult menampilkan hasil operasi Windows Service (lanjutan) lewat dialog.
func serviceResult(err error, okMsg string) {
	if err != nil {
		winui.MessageBox(appName, "Operasi service gagal:\n"+err.Error(), true)
		return
	}
	winui.MessageBox(appName, okMsg, false)
}
