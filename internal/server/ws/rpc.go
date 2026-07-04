package ws

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"

	"remote_pc/internal/protocol"
)

// ErrAgentOffline dikembalikan bila device yang dituju tidak sedang terhubung.
var ErrAgentOffline = errors.New("agent tidak terhubung")

// pendingRegistry menyimpan channel penunggu response untuk tiap request ber-UUID.
type pendingRegistry struct {
	mu   sync.Mutex
	byID map[string]chan *protocol.Envelope
}

func newPendingRegistry() *pendingRegistry {
	return &pendingRegistry{byID: make(map[string]chan *protocol.Envelope)}
}

func (p *pendingRegistry) add(id string) chan *protocol.Envelope {
	ch := make(chan *protocol.Envelope, 1)
	p.mu.Lock()
	p.byID[id] = ch
	p.mu.Unlock()
	return ch
}

func (p *pendingRegistry) remove(id string) {
	p.mu.Lock()
	delete(p.byID, id)
	p.mu.Unlock()
}

func (p *pendingRegistry) deliver(env *protocol.Envelope) bool {
	p.mu.Lock()
	ch, ok := p.byID[env.ID]
	if ok {
		delete(p.byID, env.ID)
	}
	p.mu.Unlock()
	if !ok {
		return false
	}
	ch <- env
	return true
}

// Request mengirim command ke agent dan menunggu balasannya (response/error),
// dikorelasikan lewat UUID. Mengembalikan error bila agent offline, timeout
// (ctx), atau agent membalas error.
func (h *Hub) Request(ctx context.Context, deviceID string, t protocol.MessageType, payload interface{}) (*protocol.Envelope, error) {
	env, err := protocol.NewEnvelope(t, payload)
	if err != nil {
		return nil, err
	}
	env.ID = uuid.NewString()
	env.DeviceID = deviceID

	data, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}

	ch := h.pending.add(env.ID)
	defer h.pending.remove(env.ID)

	if !h.Send(deviceID, data) {
		return nil, ErrAgentOffline
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-ch:
		if reply.Type == protocol.TypeError {
			return nil, errors.New(reply.Error)
		}
		return reply, nil
	}
}

// deliverResponse merutekan response/error dari agent ke pemanggil Request yang menunggu.
func (h *Hub) deliverResponse(env *protocol.Envelope) bool {
	return h.pending.deliver(env)
}

// Notify mengirim perintah ke agent tanpa menunggu balasan (fire-and-forget),
// dipakai untuk kontrol stream (start/stop) dan event input realtime.
func (h *Hub) Notify(deviceID string, t protocol.MessageType, payload interface{}) bool {
	env, err := protocol.NewEnvelope(t, payload)
	if err != nil {
		return false
	}
	env.DeviceID = deviceID
	data, err := json.Marshal(env)
	if err != nil {
		return false
	}
	return h.Send(deviceID, data)
}
