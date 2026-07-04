package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"remote_pc/internal/protocol"
)

// ServeOperator meng-upgrade koneksi browser operator menjadi WebSocket lalu
// merelai stream (layar/terminal) dari agent ke browser dan meneruskan input
// browser ke agent. Autentikasi operator sudah dilakukan middleware pemanggil.
//
// Query: ?device=<id>&mode=screen|terminal
func (h *Handler) ServeOperator(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device")
	channel := r.URL.Query().Get("mode")
	if deviceID == "" || (channel != "screen" && channel != "terminal") {
		http.Error(w, "parameter device/mode tidak valid", http.StatusBadRequest)
		return
	}
	if !h.hub.IsOnline(deviceID) {
		http.Error(w, "agent tidak terhubung", http.StatusConflict)
		return
	}

	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sub := h.hub.Broker.Subscribe(deviceID, channel)

	// Mulai stream layar otomatis; terminal dimulai oleh pesan term.start browser.
	if channel == "screen" {
		h.hub.Notify(deviceID, protocol.TypeScreenStart, nil)
	}

	done := make(chan struct{})
	// Penulis: teruskan frame/output dari broker ke browser. Menutup koneksi saat
	// selesai agar loop pembaca ikut berhenti.
	go func() {
		for data := range sub {
			_ = wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
				break
			}
		}
		_ = wsConn.Close()
		close(done)
	}()

	// Pembaca: teruskan input/kontrol dari browser ke agent apa adanya.
	wsConn.SetReadLimit(4 << 20)
	for {
		_, data, err := wsConn.ReadMessage()
		if err != nil {
			break
		}
		h.hub.Send(deviceID, data)
	}

	// Teardown: hentikan langganan (menutup sub -> penulis berhenti), tunggu penulis,
	// lalu minta agent menghentikan stream.
	h.hub.Broker.Unsubscribe(deviceID, channel, sub)
	<-done
	if channel == "screen" {
		h.hub.Notify(deviceID, protocol.TypeScreenStop, nil)
	} else {
		h.hub.Notify(deviceID, protocol.TypeTermStop, nil)
	}
}
