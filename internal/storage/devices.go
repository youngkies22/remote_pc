package storage

import (
	"path/filepath"
	"time"

	"remote_pc/internal/model"
)

// DeviceRepo mengelola koleksi device di devices.json.
type DeviceRepo struct {
	c *Collection[model.Device]
}

// NewDeviceRepo membuka (atau membuat) devices.json di dalam dataDir.
func NewDeviceRepo(dataDir string) (*DeviceRepo, error) {
	c, err := NewCollection[model.Device](filepath.Join(dataDir, "devices.json"))
	if err != nil {
		return nil, err
	}
	return &DeviceRepo{c: c}, nil
}

// All mengembalikan salinan seluruh device.
func (r *DeviceRepo) All() []model.Device {
	return r.c.Read()
}

// Get mencari device berdasarkan ID.
func (r *DeviceRepo) Get(id string) (model.Device, bool) {
	for _, d := range r.c.Read() {
		if d.ID == id {
			return d, true
		}
	}
	return model.Device{}, false
}

// GetByToken mencari device berdasarkan token (dipakai autentikasi agent).
func (r *DeviceRepo) GetByToken(token string) (model.Device, bool) {
	if token == "" {
		return model.Device{}, false
	}
	for _, d := range r.c.Read() {
		if d.Token == token {
			return d, true
		}
	}
	return model.Device{}, false
}

// Save menyimpan device (upsert berdasarkan ID).
func (r *DeviceRepo) Save(dev model.Device) error {
	return r.c.Mutate(func(items []model.Device) ([]model.Device, error) {
		for i := range items {
			if items[i].ID == dev.ID {
				items[i] = dev
				return items, nil
			}
		}
		return append(items, dev), nil
	})
}

// Delete menghapus device berdasarkan ID. Tidak error bila ID tidak ditemukan.
func (r *DeviceRepo) Delete(id string) error {
	return r.c.Mutate(func(items []model.Device) ([]model.Device, error) {
		out := items[:0]
		for _, d := range items {
			if d.ID != id {
				out = append(out, d)
			}
		}
		return out, nil
	})
}

// MarkStaleOffline menandai device offline bila LastSeen melewati ambang batas.
// Mengembalikan daftar ID device yang statusnya baru saja berubah menjadi offline.
func (r *DeviceRepo) MarkStaleOffline(threshold time.Duration) ([]string, error) {
	var changed []string
	err := r.c.Mutate(func(items []model.Device) ([]model.Device, error) {
		now := time.Now()
		for i := range items {
			if items[i].Status == model.StatusOnline && now.Sub(items[i].LastSeen) > threshold {
				items[i].Status = model.StatusOffline
				changed = append(changed, items[i].ID)
			}
		}
		return items, nil
	})
	return changed, err
}
