package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"remote_pc/internal/protocol"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 << 20 // 64 MB, cukup untuk frame layar & transfer file
	sendBuffer     = 64
)

// Conn merepresentasikan satu koneksi WebSocket agent. Penulisan diserialkan
// melalui channel send + satu goroutine writePump untuk menghindari penulisan
// bersamaan pada koneksi gorilla (yang tidak aman untuk concurrent write).
type Conn struct {
	deviceID  string
	ws        *websocket.Conn
	send      chan []byte
	hub       *Hub
	log       *zap.Logger
	onMessage func(*Conn, *protocol.Envelope)
	closeOnce sync.Once
	closed    chan struct{}
}

func newConn(deviceID string, wsConn *websocket.Conn, hub *Hub, log *zap.Logger) *Conn {
	return &Conn{
		deviceID: deviceID,
		ws:       wsConn,
		send:     make(chan []byte, sendBuffer),
		hub:      hub,
		log:      log,
		closed:   make(chan struct{}),
	}
}

// DeviceID mengembalikan ID device pemilik koneksi.
func (c *Conn) DeviceID() string { return c.deviceID }

// enqueue menaruh data mentah ke antrean kirim. Bila antrean penuh (konsumen
// lambat) atau koneksi sudah ditutup, koneksi diputus dan mengembalikan false.
func (c *Conn) enqueue(data []byte) bool {
	select {
	case <-c.closed:
		return false
	case c.send <- data:
		return true
	default:
		c.close()
		return false
	}
}

// sendEnvelope mengirim sebuah envelope terserialisasi ke agent.
func (c *Conn) sendEnvelope(env *protocol.Envelope) bool {
	data, err := json.Marshal(env)
	if err != nil {
		c.log.Error("gagal marshal envelope", zap.Error(err))
		return false
	}
	return c.enqueue(data)
}

// close menutup koneksi tepat satu kali dan melepasnya dari hub.
func (c *Conn) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.ws.Close()
		if c.hub != nil {
			c.hub.remove(c)
		}
	})
}

// writePump menulis pesan dari channel send dan mengirim ping periodik.
func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-c.closed:
			return
		case msg := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		}
	}
}

// readPump membaca pesan dari agent, mem-parse envelope, dan meneruskannya ke
// onMessage. Fungsi ini blocking; keluar saat koneksi putus lalu menutup Conn.
func (c *Conn) readPump() {
	defer c.close()
	c.ws.SetReadLimit(maxMessageSize)
	_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
		var env protocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			c.log.Warn("envelope tidak valid dari agent",
				zap.String("device_id", c.deviceID), zap.Error(err))
			continue
		}
		if c.onMessage != nil {
			c.onMessage(c, &env)
		}
	}
}
