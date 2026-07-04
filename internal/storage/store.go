// Package storage adalah satu-satunya pintu baca/tulis data berbasis file JSON.
// Tidak ada package lain yang boleh membaca file data JSON secara langsung.
// Setiap koleksi dijaga RWMutex dan ditulis secara atomik (tulis .tmp lalu rename)
// agar tidak korup bila proses berhenti di tengah penulisan.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Collection mengelola sebuah file JSON yang berisi slice item bertipe T.
// Semua akses dilindungi RWMutex sehingga aman dipakai banyak goroutine.
type Collection[T any] struct {
	mu    sync.RWMutex
	path  string
	items []T
}

// NewCollection memuat file pada path ke memori. File yang belum ada dianggap
// koleksi kosong (bukan error) dan akan dibuat saat penyimpanan pertama.
func NewCollection[T any](path string) (*Collection[T], error) {
	c := &Collection[T]{path: path, items: []T{}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("storage: buat direktori %s: %w", filepath.Dir(path), err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("storage: baca %s: %w", path, err)
	}
	if len(data) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(data, &c.items); err != nil {
		return nil, fmt.Errorf("storage: parse %s: %w", path, err)
	}
	return c, nil
}

// Read memberikan salinan seluruh item di bawah read-lock. Salinan mencegah
// pemanggil memodifikasi state internal tanpa melewati Mutate.
func (c *Collection[T]) Read() []T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]T, len(c.items))
	copy(out, c.items)
	return out
}

// Mutate menjalankan fn di bawah write-lock. fn menerima slice item saat ini dan
// mengembalikan slice baru yang akan dipersist secara atomik ke disk.
func (c *Collection[T]) Mutate(fn func(items []T) ([]T, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	next, err := fn(c.items)
	if err != nil {
		return err
	}
	if err := c.persist(next); err != nil {
		return err
	}
	c.items = next
	return nil
}

// persist menulis items ke file secara atomik. Dipanggil sudah dalam write-lock.
func (c *Collection[T]) persist(items []T) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: marshal %s: %w", c.path, err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("storage: tulis tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("storage: rename %s: %w", c.path, err)
	}
	return nil
}
