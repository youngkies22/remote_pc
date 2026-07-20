package agent

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"remote_pc/internal/agent/sysinfo"
	"remote_pc/internal/agent/terminal"
	"remote_pc/internal/protocol"
)

// session mewakili satu koneksi agent yang sudah teregistrasi. Penulisan ke
// socket diserialkan melalui channel send + goroutine writer tunggal.
type session struct {
	ws       *websocket.Conn
	deviceID string
	static   sysinfo.Static
	log      *zap.Logger
	send     chan *protocol.Envelope
	ctx      context.Context // siklus hidup koneksi (untuk stream & terminal)

	mu            sync.Mutex
	screenStop    context.CancelFunc // pembatal loop stream layar (nil bila tidak stream)
	screenQuality string             // "normal" (default) atau "hd", diatur browser via TypeScreenQuality
	term          *terminal.Session  // sesi terminal aktif (nil bila tidak ada)
}

// cleanup menghentikan stream layar dan menutup terminal saat koneksi berakhir.
func (s *session) cleanup() {
	s.stopScreen()
	s.mu.Lock()
	if s.term != nil {
		s.term.Close()
		s.term = nil
	}
	s.mu.Unlock()
}

// writer menulis envelope dari channel send ke socket. Keluar saat ctx selesai
// atau saat terjadi kegagalan tulis (lalu membatalkan sesi).
func (s *session) writer(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case env := <-s.send:
			_ = s.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.ws.WriteJSON(env); err != nil {
				s.log.Warn("gagal menulis ke server", zap.Error(err))
				return
			}
		}
	}
}

// reader membaca pesan dari server dan meneruskannya ke dispatch. Keluar saat
// koneksi putus (lalu membatalkan sesi).
func (s *session) reader(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	for {
		var env protocol.Envelope
		if err := s.ws.ReadJSON(&env); err != nil {
			return
		}
		s.dispatch(ctx, &env)
	}
}

// enqueue mengantre envelope untuk dikirim tanpa memblokir bila sesi berakhir.
func (s *session) enqueue(ctx context.Context, env *protocol.Envelope) bool {
	select {
	case s.send <- env:
		return true
	case <-ctx.Done():
		return false
	}
}
