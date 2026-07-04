// Package agent mengimplementasikan client (agent) yang berjalan di komputer
// target: terhubung ke server via WebSocket, registrasi, mengirim heartbeat,
// dan menerima command.
package agent

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"remote_pc/internal/agent/sysinfo"
	"remote_pc/internal/config"
	"remote_pc/internal/model"
	"remote_pc/internal/protocol"
)

// Client mengelola koneksi agent ke server, termasuk reconnect otomatis.
type Client struct {
	cfg    *config.AgentConfig
	log    *zap.Logger
	static sysinfo.Static
}

// New membuat Client agent dan mengumpulkan informasi statis host sekali.
func New(cfg *config.AgentConfig, log *zap.Logger) *Client {
	static, err := sysinfo.Collect()
	if err != nil {
		log.Warn("gagal mengumpulkan info statis", zap.Error(err))
	}
	sysinfo.PrimeCPU()
	return &Client{cfg: cfg, log: log, static: static}
}

// Run menjalankan loop koneksi: connect, layani, dan reconnect saat putus,
// sampai ctx dibatalkan.
func (c *Client) Run(ctx context.Context) error {
	reconnect := time.Duration(c.cfg.Agent.ReconnectSeconds) * time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := c.connectAndServe(ctx); err != nil {
			c.log.Warn("koneksi berakhir", zap.Error(err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.log.Info("mencoba reconnect", zap.Duration("dalam", reconnect))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnect):
		}
	}
}

// connectAndServe membuka satu koneksi, registrasi, lalu melayani sampai putus.
func (c *Client) connectAndServe(parent context.Context) error {
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	wsConn, _, err := dialer.DialContext(parent, c.cfg.Agent.ServerURL, nil)
	if err != nil {
		return err
	}
	defer wsConn.Close()

	deviceID, err := c.register(wsConn)
	if err != nil {
		return err
	}
	c.log.Info("registrasi berhasil", zap.String("device_id", deviceID))

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	sess := &session{
		ws:       wsConn,
		deviceID: deviceID,
		static:   c.static,
		log:      c.log,
		send:     make(chan *protocol.Envelope, 32),
		ctx:      ctx,
	}
	defer sess.cleanup()

	go sess.writer(ctx, cancel)
	go sess.reader(ctx, cancel)
	return sess.heartbeatLoop(ctx, time.Duration(c.cfg.Agent.HeartbeatSeconds)*time.Second)
}

// register mengirim pesan registrasi dan memproses balasan server.
func (c *Client) register(wsConn *websocket.Conn) (string, error) {
	info := model.RegisterInfo{
		DeviceID:       c.cfg.Agent.DeviceID,
		Token:          c.cfg.Agent.DeviceToken,
		Hostname:       c.static.Hostname,
		Username:       c.static.Username,
		IP:             c.static.IP,
		MAC:            c.static.MAC,
		OS:             c.static.OS,
		WindowsVersion: c.static.WindowsVersion,
		Arch:           c.static.Arch,
	}
	env, err := protocol.NewEnvelope(protocol.TypeRegister, info)
	if err != nil {
		return "", err
	}
	if err := wsConn.WriteJSON(env); err != nil {
		return "", err
	}

	_ = wsConn.SetReadDeadline(time.Now().Add(15 * time.Second))
	var res protocol.Envelope
	if err := wsConn.ReadJSON(&res); err != nil {
		return "", err
	}
	_ = wsConn.SetReadDeadline(time.Time{})

	var result model.RegisterResult
	if err := res.Decode(&result); err != nil {
		return "", err
	}
	if !result.Accepted {
		return "", errors.New("registrasi ditolak server: " + result.Message)
	}
	c.persistIdentity(result)
	return result.DeviceID, nil
}

// persistIdentity menyimpan device_id/token baru ke agent.yaml bila berubah.
func (c *Client) persistIdentity(result model.RegisterResult) {
	if result.DeviceID == c.cfg.Agent.DeviceID && result.Token == c.cfg.Agent.DeviceToken {
		return
	}
	c.cfg.Agent.DeviceID = result.DeviceID
	c.cfg.Agent.DeviceToken = result.Token
	if err := c.cfg.Save(); err != nil {
		c.log.Warn("gagal menyimpan identitas agent", zap.Error(err))
	}
}
