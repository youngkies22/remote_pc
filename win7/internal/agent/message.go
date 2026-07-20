package agent

import (
	"strings"

	"go.uber.org/zap"

	"remote_pc/internal/protocol"
	"remote_pc/internal/winui"
)

// showMessage menampilkan dialog pesan di layar komputer siswa (agent berjalan di
// sesi desktop pengguna via TriggerLogon, sehingga MessageBox tampil di desktop
// aktif). Dialog bersifat modal-blocking, jadi dijalankan di goroutine terpisah
// agar tidak memblokir loop pembaca WebSocket.
func (s *session) showMessage(env *protocol.Envelope) {
	var req protocol.MessageRequest
	if err := env.Decode(&req); err != nil {
		s.log.Warn("gagal decode pesan", zap.Error(err))
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Pesan dari Guru"
	}
	text := req.Text
	if strings.TrimSpace(text) == "" {
		return // jangan tampilkan dialog kosong
	}
	go func() {
		winui.MessageBox(title, text, false)
		s.log.Info("pesan ditampilkan ke pengguna", zap.String("title", title))
	}()
}
