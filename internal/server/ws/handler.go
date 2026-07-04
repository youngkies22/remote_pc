package ws

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"remote_pc/internal/model"
	"remote_pc/internal/protocol"
	"remote_pc/internal/storage"
)

// Handler menangani endpoint WebSocket agent: upgrade, registrasi, dan heartbeat.
type Handler struct {
	hub      *Hub
	devices  *storage.DeviceRepo
	log      *zap.Logger
	upgrader websocket.Upgrader
}

// NewHandler membuat Handler WebSocket agent.
func NewHandler(hub *Hub, devices *storage.DeviceRepo, log *zap.Logger) *Handler {
	return &Handler{
		hub:     hub,
		devices: devices,
		log:     log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// Agent bukan browser sehingga origin tidak relevan; keamanan dijaga
			// oleh device token pada handshake registrasi.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// ServeAgent meng-upgrade koneksi HTTP menjadi WebSocket lalu menjalankan siklus
// hidup koneksi agent (registrasi -> loop pesan -> tandai offline saat putus).
func (h *Handler) ServeAgent(w http.ResponseWriter, r *http.Request) {
	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("gagal upgrade websocket", zap.Error(err))
		return
	}

	dev, ok := h.register(wsConn, r)
	if !ok {
		_ = wsConn.Close()
		return
	}

	conn := newConn(dev.ID, wsConn, h.hub, h.log)
	conn.onMessage = h.handleMessage
	h.hub.add(conn)
	h.log.Info("agent terhubung",
		zap.String("device_id", dev.ID), zap.String("hostname", dev.Hostname))

	go conn.writePump()
	conn.readPump() // blocking sampai koneksi putus

	h.markOffline(dev.ID)
	h.log.Info("agent terputus", zap.String("device_id", dev.ID))
}

// register memproses pesan pertama (registrasi). Untuk device lama, token
// diverifikasi; untuk device baru, ID dan token baru diterbitkan.
func (h *Handler) register(wsConn *websocket.Conn, r *http.Request) (model.Device, bool) {
	_ = wsConn.SetReadDeadline(time.Now().Add(15 * time.Second))
	var env protocol.Envelope
	if err := wsConn.ReadJSON(&env); err != nil {
		h.log.Warn("gagal baca pesan registrasi", zap.Error(err))
		return model.Device{}, false
	}
	_ = wsConn.SetReadDeadline(time.Time{})

	if env.Type != protocol.TypeRegister {
		h.log.Warn("pesan pertama bukan register", zap.String("type", string(env.Type)))
		return model.Device{}, false
	}
	var info model.RegisterInfo
	if err := env.Decode(&info); err != nil {
		h.log.Warn("payload registrasi tidak valid", zap.Error(err))
		return model.Device{}, false
	}

	dev, err := h.resolveDevice(info, clientIP(r))
	if err != nil {
		h.log.Warn("registrasi ditolak", zap.Error(err))
		_ = wsConn.WriteJSON(mustReply(&env, protocol.TypeRegisterResult,
			model.RegisterResult{Accepted: false, Message: err.Error()}))
		return model.Device{}, false
	}

	reply := mustReply(&env, protocol.TypeRegisterResult, model.RegisterResult{
		DeviceID: dev.ID, Token: dev.Token, Accepted: true,
	})
	if err := wsConn.WriteJSON(reply); err != nil {
		h.log.Warn("gagal kirim hasil registrasi", zap.Error(err))
		return model.Device{}, false
	}
	return dev, true
}

// resolveDevice membuat device baru atau memverifikasi & memperbarui device lama.
func (h *Handler) resolveDevice(info model.RegisterInfo, ip string) (model.Device, error) {
	now := time.Now()
	if info.DeviceID != "" {
		existing, found := h.devices.Get(info.DeviceID)
		if found {
			if existing.Token != info.Token {
				return model.Device{}, errTokenMismatch
			}
			applyRegisterInfo(&existing, info, ip, now)
			if err := h.devices.Save(existing); err != nil {
				return model.Device{}, err
			}
			return existing, nil
		}
	}
	dev := model.Device{
		ID:        uuid.NewString(),
		Token:     newToken(),
		FirstSeen: now,
	}
	applyRegisterInfo(&dev, info, ip, now)
	if err := h.devices.Save(dev); err != nil {
		return model.Device{}, err
	}
	return dev, nil
}

// handleMessage memproses pesan setelah registrasi (heartbeat, response, stream).
func (h *Handler) handleMessage(c *Conn, env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypeHeartbeat:
		h.handleHeartbeat(c.DeviceID(), env)
	case protocol.TypeResponse, protocol.TypeError:
		h.hub.deliverResponse(env)
	case protocol.TypeScreenFrame:
		h.hub.Broker.Publish(c.DeviceID(), "screen", mustJSON(env))
	case protocol.TypeTermOutput:
		h.hub.Broker.Publish(c.DeviceID(), "terminal", mustJSON(env))
	case protocol.TypePong:
		// keepalive
	default:
		h.log.Debug("tipe pesan belum didukung",
			zap.String("type", string(env.Type)), zap.String("device_id", c.DeviceID()))
	}
}

// handleHeartbeat memperbarui metrik dan status device di storage.
func (h *Handler) handleHeartbeat(deviceID string, env *protocol.Envelope) {
	var hb model.Heartbeat
	if err := env.Decode(&hb); err != nil {
		h.log.Warn("payload heartbeat tidak valid", zap.Error(err))
		return
	}
	dev, ok := h.devices.Get(deviceID)
	if !ok {
		return
	}
	dev.Metrics = hb.Metrics
	dev.Status = model.StatusOnline
	dev.LastSeen = time.Now()
	if hb.Hostname != "" {
		dev.Hostname = hb.Hostname
	}
	if hb.Username != "" {
		dev.Username = hb.Username
	}
	if hb.IP != "" {
		dev.IP = hb.IP
	}
	if hb.MAC != "" {
		dev.MAC = hb.MAC
	}
	if err := h.devices.Save(dev); err != nil {
		h.log.Error("gagal simpan heartbeat", zap.Error(err))
	}
}

func (h *Handler) markOffline(deviceID string) {
	dev, ok := h.devices.Get(deviceID)
	if !ok {
		return
	}
	dev.Status = model.StatusOffline
	if err := h.devices.Save(dev); err != nil {
		h.log.Error("gagal tandai offline", zap.Error(err))
	}
}

func applyRegisterInfo(dev *model.Device, info model.RegisterInfo, ip string, now time.Time) {
	dev.Hostname = info.Hostname
	dev.Username = info.Username
	dev.OS = info.OS
	dev.WindowsVersion = info.WindowsVersion
	dev.Arch = info.Arch
	dev.MAC = info.MAC
	dev.IP = firstNonEmpty(info.IP, ip)
	dev.Status = model.StatusOnline
	dev.LastSeen = now
	if dev.FirstSeen.IsZero() {
		dev.FirstSeen = now
	}
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.NewString()
	}
	return hex.EncodeToString(b)
}

func mustReply(env *protocol.Envelope, t protocol.MessageType, payload interface{}) *protocol.Envelope {
	reply, err := env.Reply(t, payload)
	if err != nil {
		return env.ErrorReply(err.Error())
	}
	return reply
}
