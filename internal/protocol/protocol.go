// Package protocol mendefinisikan format pesan WebSocket antara server dan agent.
// Seluruh pesan dibungkus dalam Envelope berformat JSON. Setiap request memiliki
// ID (UUID) sehingga response/error dapat dikorelasikan dengan request-nya.
package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType adalah jenis pesan pada field "type".
type MessageType string

const (
	// TypeRegister dikirim agent saat pertama terhubung untuk registrasi.
	TypeRegister MessageType = "register"
	// TypeRegisterResult dikirim server sebagai balasan registrasi.
	TypeRegisterResult MessageType = "register_result"
	// TypeHeartbeat dikirim agent secara periodik berisi metrik terkini.
	TypeHeartbeat MessageType = "heartbeat"
	// TypeCommand dikirim server untuk meminta agent menjalankan sesuatu.
	TypeCommand MessageType = "command"
	// TypeResponse dikirim agent sebagai hasil dari sebuah command.
	TypeResponse MessageType = "response"
	// TypeError menandakan terjadi kesalahan pada pemrosesan sebuah pesan.
	TypeError MessageType = "error"
	// TypePing/TypePong dipakai untuk menjaga koneksi (application-level keepalive).
	TypePing MessageType = "ping"
	TypePong MessageType = "pong"

	// --- Aksi command request/response (server -> agent, dibalas response/error) ---
	TypeSysInfo    MessageType = "sysinfo"    // Tahap 4: info sistem via WMI
	TypeScreenshot MessageType = "screenshot" // Tahap 5: tangkap layar sekali (JPEG)
	TypeFSDrives   MessageType = "fs.drives"  // Tahap 6: daftar drive
	TypeFSList     MessageType = "fs.list"    // Tahap 6: daftar isi folder
	TypeFSRead     MessageType = "fs.read"    // Tahap 6: unduh file
	TypeFSWrite    MessageType = "fs.write"   // Tahap 6: unggah file
	TypeFSRename   MessageType = "fs.rename"  // Tahap 6
	TypeFSDelete   MessageType = "fs.delete"  // Tahap 6
	TypeFSMkdir    MessageType = "fs.mkdir"   // Tahap 6
	TypeFSCopy     MessageType = "fs.copy"    // Tahap 6
	TypeFSMove     MessageType = "fs.move"    // Tahap 6
	TypeProcList   MessageType = "proc.list"  // Tahap 11
	TypeProcKill   MessageType = "proc.kill"  // Tahap 11
	TypeSvcList    MessageType = "svc.list"   // Tahap 12
	TypeSvcControl MessageType = "svc.control" // Tahap 12

	// --- Streaming (relay lewat operator WebSocket) ---
	TypeScreenStart   MessageType = "screen.start"   // Tahap 8: mulai stream layar
	TypeScreenStop    MessageType = "screen.stop"    // Tahap 8: hentikan stream
	TypeScreenFrame   MessageType = "screen.frame"   // Tahap 8: frame JPEG dari agent
	TypeScreenQuality MessageType = "screen.quality" // atur kualitas stream (normal/hd)
	TypeInputMouse  MessageType = "input.mouse"  // Tahap 9: event mouse
	TypeInputKey    MessageType = "input.key"    // Tahap 10: event keyboard
	TypeTermStart   MessageType = "term.start"   // Tahap 7: buka shell
	TypeTermInput   MessageType = "term.input"   // Tahap 7: input ke shell
	TypeTermOutput  MessageType = "term.output"  // Tahap 7: output dari shell
	TypeTermStop    MessageType = "term.stop"    // Tahap 7: tutup shell

	// --- Power control (server -> agent, fire-and-forget) ---
	TypePowerShutdown MessageType = "power.shutdown" // matikan komputer
	TypePowerRestart  MessageType = "power.restart"  // restart komputer
)

// Envelope adalah amplop standar untuk semua pesan WebSocket.
type Envelope struct {
	ID        string          `json:"id"`
	Type      MessageType     `json:"type"`
	DeviceID  string          `json:"device_id,omitempty"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// NewEnvelope membuat Envelope baru dengan ID acak dan timestamp saat ini.
// payload akan di-marshal ke JSON; kesalahan marshal dikembalikan ke pemanggil.
func NewEnvelope(t MessageType, payload interface{}) (*Envelope, error) {
	env := &Envelope{
		ID:        uuid.NewString(),
		Type:      t,
		Timestamp: time.Now().UnixMilli(),
	}
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		env.Payload = raw
	}
	return env, nil
}

// Reply membuat Envelope balasan yang mewarisi ID dari envelope asal sehingga
// server/agent dapat mencocokkan response dengan request.
func (e *Envelope) Reply(t MessageType, payload interface{}) (*Envelope, error) {
	reply, err := NewEnvelope(t, payload)
	if err != nil {
		return nil, err
	}
	reply.ID = e.ID
	reply.DeviceID = e.DeviceID
	return reply, nil
}

// ErrorReply membuat Envelope bertipe error dengan ID yang sama seperti request.
func (e *Envelope) ErrorReply(msg string) *Envelope {
	return &Envelope{
		ID:        e.ID,
		Type:      TypeError,
		DeviceID:  e.DeviceID,
		Timestamp: time.Now().UnixMilli(),
		Error:     msg,
	}
}

// Decode meng-unmarshal payload envelope ke struct tujuan.
func (e *Envelope) Decode(out interface{}) error {
	if len(e.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(e.Payload, out)
}
