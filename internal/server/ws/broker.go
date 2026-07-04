package ws

import "sync"

// Broker menyalurkan aliran data (frame layar, output terminal) dari agent ke
// operator (browser) yang berlangganan sebuah kanal per-device.
type Broker struct {
	mu   sync.RWMutex
	subs map[string]map[chan []byte]struct{}
}

// NewBroker membuat Broker kosong.
func NewBroker() *Broker {
	return &Broker{subs: make(map[string]map[chan []byte]struct{})}
}

func brokerKey(deviceID, channel string) string {
	return deviceID + "|" + channel
}

// Subscribe mendaftarkan langganan dan mengembalikan channel penerima data.
func (b *Broker) Subscribe(deviceID, channel string) chan []byte {
	key := brokerKey(deviceID, channel)
	ch := make(chan []byte, 8)
	b.mu.Lock()
	if b.subs[key] == nil {
		b.subs[key] = make(map[chan []byte]struct{})
	}
	b.subs[key][ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe melepas langganan dan menutup channel-nya.
func (b *Broker) Unsubscribe(deviceID, channel string, ch chan []byte) {
	key := brokerKey(deviceID, channel)
	b.mu.Lock()
	if set, ok := b.subs[key]; ok {
		if _, exists := set[ch]; exists {
			delete(set, ch)
			close(ch)
		}
		if len(set) == 0 {
			delete(b.subs, key)
		}
	}
	b.mu.Unlock()
}

// Publish mengirim data ke semua pelanggan kanal. Pengiriman non-blocking:
// data di-drop bila buffer pelanggan penuh (mencegah agent tertahan konsumen lambat).
func (b *Broker) Publish(deviceID, channel string, data []byte) {
	key := brokerKey(deviceID, channel)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[key] {
		select {
		case ch <- data:
		default:
		}
	}
}

// HasSubscribers melaporkan apakah ada operator yang menonton kanal ini.
func (b *Broker) HasSubscribers(deviceID, channel string) bool {
	key := brokerKey(deviceID, channel)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[key]) > 0
}
