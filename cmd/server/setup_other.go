//go:build !windows

package main

import "remote_pc/internal/winui"

// enableAutostart tidak didukung di luar Windows: gunakan restart policy
// container (mis. "restart: unless-stopped" di Docker Compose) atau unit
// systemd sendiri bila menjalankan binary langsung tanpa container.
func enableAutostart(configPath string) {
	winui.MessageBox(appName,
		"Perintah 'enable' hanya berlaku di Windows (Task Scheduler + Firewall).\n"+
			"Di Linux/Docker, jadwalkan restart otomatis lewat restart policy "+
			"container (mis. restart: unless-stopped) atau systemd.", true)
}

// disableAutostart tidak didukung di luar Windows — lihat enableAutostart.
func disableAutostart() {
	winui.MessageBox(appName,
		"Perintah 'disable' hanya berlaku di Windows. Di Linux/Docker, hentikan "+
			"container/service lewat tool orkestrasi yang dipakai.", true)
}
