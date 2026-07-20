package agent

import (
	"os/exec"
	"syscall"

	"go.uber.org/zap"
)

// createNoWindow mencegah jendela console muncul saat memanggil shutdown.exe.
const createNoWindow = 0x08000000

// powerControl mematikan atau merestart komputer memakai shutdown.exe. Dijalankan
// dengan /f (paksa tutup aplikasi) dan /t 0 (tanpa penundaan). Standard user pun
// umumnya boleh mematikan komputernya sendiri, sehingga tidak wajib admin.
func (s *session) powerControl(restart bool) {
	flag := "/s" // shutdown
	if restart {
		flag = "/r" // restart
	}
	cmd := exec.Command("shutdown.exe", flag, "/t", "0", "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if err := cmd.Start(); err != nil {
		s.log.Warn("gagal menjalankan shutdown", zap.Bool("restart", restart), zap.Error(err))
		return
	}
	s.log.Info("perintah power dijalankan", zap.Bool("restart", restart))
}
