package agent

import (
	"context"
	"time"

	"go.uber.org/zap"

	"remote_pc/internal/agent/screen"
	"remote_pc/internal/agent/terminal"
	"remote_pc/internal/protocol"
)

// screenFPS menentukan target frame rate stream layar (~10 FPS).
const screenInterval = 100 * time.Millisecond

// screenQualityNormal/HD adalah preset kualitas JPEG untuk stream Live Screen.
// Resolusi selalu native (tidak diturunkan) — yang berubah hanya kompresi.
const (
	screenQualityNormal = "normal"
	screenQualityHD     = "hd"
)

// encoderForQuality memetakan preset kualitas ke encoder JPEG yang sesuai.
func encoderForQuality(quality string) screen.Encoder {
	if quality == screenQualityHD {
		return screen.JPEGEncoder{Quality: 92}
	}
	return screen.JPEGEncoder{Quality: 60}
}

// setScreenQuality mengubah kualitas stream yang sedang berjalan; berlaku pada
// frame berikutnya tanpa perlu reconnect.
func (s *session) setScreenQuality(quality string) {
	if quality != screenQualityHD {
		quality = screenQualityNormal
	}
	s.mu.Lock()
	s.screenQuality = quality
	s.mu.Unlock()
}

func (s *session) getScreenQuality() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.screenQuality == "" {
		return screenQualityNormal
	}
	return s.screenQuality
}

// startScreen memulai loop pengiriman frame layar bila belum berjalan.
func (s *session) startScreen() {
	s.mu.Lock()
	if s.screenStop != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.screenStop = cancel
	s.mu.Unlock()
	go s.screenLoop(ctx)
}

// stopScreen menghentikan loop stream layar bila sedang berjalan.
func (s *session) stopScreen() {
	s.mu.Lock()
	if s.screenStop != nil {
		s.screenStop()
		s.screenStop = nil
	}
	s.mu.Unlock()
}

// screenLoop menangkap dan mengirim frame layar sampai ctx dibatalkan.
func (s *session) screenLoop(ctx context.Context) {
	ticker := time.NewTicker(screenInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			enc := encoderForQuality(s.getScreenQuality())
			shot, err := screen.CaptureWith(enc)
			if err != nil {
				continue
			}
			env, err := protocol.NewEnvelope(protocol.TypeScreenFrame, shot)
			if err != nil {
				continue
			}
			env.DeviceID = s.deviceID
			if !s.enqueue(ctx, env) {
				return
			}
		}
	}
}

// startTerminal membuka sesi shell dan mengalirkan output-nya ke server.
func (s *session) startTerminal(shell string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.term != nil {
		return
	}
	term, err := terminal.Start(shell, func(out string) {
		env, err := protocol.NewEnvelope(protocol.TypeTermOutput, protocol.TermData{Data: out})
		if err != nil {
			return
		}
		env.DeviceID = s.deviceID
		s.enqueue(s.ctx, env)
	})
	if err != nil {
		s.log.Warn("gagal membuka terminal", zap.Error(err))
		s.enqueue(s.ctx, mustEnvelope(protocol.TypeTermOutput,
			protocol.TermData{Data: "[gagal membuka shell: " + err.Error() + "]\r\n"}, s.deviceID))
		return
	}
	s.term = term
}

// termInput meneruskan input ke shell yang sedang berjalan.
func (s *session) termInput(data string) {
	s.mu.Lock()
	term := s.term
	s.mu.Unlock()
	if term != nil {
		_ = term.Write(data)
	}
}

// stopTerminal menutup sesi shell yang berjalan.
func (s *session) stopTerminal() {
	s.mu.Lock()
	term := s.term
	s.term = nil
	s.mu.Unlock()
	if term != nil {
		term.Close()
	}
}

func mustEnvelope(t protocol.MessageType, payload interface{}, deviceID string) *protocol.Envelope {
	env, err := protocol.NewEnvelope(t, payload)
	if err != nil {
		return &protocol.Envelope{Type: t, DeviceID: deviceID}
	}
	env.DeviceID = deviceID
	return env
}
