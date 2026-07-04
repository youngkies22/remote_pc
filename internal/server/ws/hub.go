// Package ws mengelola koneksi WebSocket dari agent: registrasi koneksi,
// pengiriman command, dan penerimaan heartbeat/response.
package ws

import (
	"sync"

	"go.uber.org/zap"
)

// Hub adalah registry seluruh koneksi agent yang sedang aktif, aman untuk
// diakses banyak goroutine.
type Hub struct {
	mu      sync.RWMutex
	conns   map[string]*Conn // deviceID -> koneksi aktif
	log     *zap.Logger
	pending *pendingRegistry // korelasi request/response ber-UUID
	Broker  *Broker          // penyalur stream ke operator (layar/terminal)
}

// NewHub membuat Hub kosong.
func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		conns:   make(map[string]*Conn),
		log:     log,
		pending: newPendingRegistry(),
		Broker:  NewBroker(),
	}
}

// add mendaftarkan koneksi. Bila deviceID sudah punya koneksi lama, koneksi lama
// ditutup agar hanya ada satu koneksi aktif per device.
func (h *Hub) add(c *Conn) {
	h.mu.Lock()
	old, exists := h.conns[c.deviceID]
	h.conns[c.deviceID] = c
	h.mu.Unlock()
	if exists && old != c {
		old.close()
	}
}

// remove melepas koneksi hanya bila koneksi tersebut masih yang terdaftar.
func (h *Hub) remove(c *Conn) {
	h.mu.Lock()
	if cur, ok := h.conns[c.deviceID]; ok && cur == c {
		delete(h.conns, c.deviceID)
	}
	h.mu.Unlock()
}

// Send mengirim data mentah ke agent tertentu. Mengembalikan false bila agent
// tidak sedang terhubung.
func (h *Hub) Send(deviceID string, data []byte) bool {
	h.mu.RLock()
	c, ok := h.conns[deviceID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	return c.enqueue(data)
}

// IsOnline melaporkan apakah device memiliki koneksi aktif.
func (h *Hub) IsOnline(deviceID string) bool {
	h.mu.RLock()
	_, ok := h.conns[deviceID]
	h.mu.RUnlock()
	return ok
}

// OnlineIDs mengembalikan daftar deviceID yang sedang terhubung.
func (h *Hub) OnlineIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	return ids
}

// CloseAll menutup seluruh koneksi (dipakai saat shutdown server).
func (h *Hub) CloseAll() {
	h.mu.Lock()
	conns := make([]*Conn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		c.close()
	}
}
