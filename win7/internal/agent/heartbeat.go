package agent

import (
	"context"
	"time"

	"remote_pc/internal/agent/sysinfo"
	"remote_pc/internal/model"
	"remote_pc/internal/protocol"
)

// heartbeatLoop mengirim heartbeat pertama segera, lalu berkala sesuai interval,
// sampai ctx dibatalkan. Mengembalikan penyebab berhentinya loop.
func (s *session) heartbeatLoop(ctx context.Context, interval time.Duration) error {
	if err := s.sendHeartbeat(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.sendHeartbeat(ctx); err != nil {
				return err
			}
		}
	}
}

// sendHeartbeat mengumpulkan metrik terkini dan mengantrekannya untuk dikirim.
func (s *session) sendHeartbeat(ctx context.Context) error {
	hb := model.Heartbeat{
		Hostname: s.static.Hostname,
		Username: s.static.Username,
		IP:       s.static.IP,
		MAC:      s.static.MAC,
		Metrics:  sysinfo.CollectMetrics(ctx),
	}
	env, err := protocol.NewEnvelope(protocol.TypeHeartbeat, hb)
	if err != nil {
		return err
	}
	env.DeviceID = s.deviceID
	if !s.enqueue(ctx, env) {
		return ctx.Err()
	}
	return nil
}
