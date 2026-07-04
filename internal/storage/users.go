package storage

import (
	"path/filepath"

	"remote_pc/internal/model"
)

// UserRepo mengelola koleksi user operator di users.json.
type UserRepo struct {
	c *Collection[model.User]
}

// NewUserRepo membuka (atau membuat) users.json di dalam dataDir.
func NewUserRepo(dataDir string) (*UserRepo, error) {
	c, err := NewCollection[model.User](filepath.Join(dataDir, "users.json"))
	if err != nil {
		return nil, err
	}
	return &UserRepo{c: c}, nil
}

// All mengembalikan salinan seluruh user.
func (r *UserRepo) All() []model.User {
	return r.c.Read()
}

// Count mengembalikan jumlah user terdaftar.
func (r *UserRepo) Count() int {
	return len(r.c.Read())
}

// GetByUsername mencari user berdasarkan username (case-sensitive).
func (r *UserRepo) GetByUsername(username string) (model.User, bool) {
	for _, u := range r.c.Read() {
		if u.Username == username {
			return u, true
		}
	}
	return model.User{}, false
}

// Get mencari user berdasarkan ID.
func (r *UserRepo) Get(id string) (model.User, bool) {
	for _, u := range r.c.Read() {
		if u.ID == id {
			return u, true
		}
	}
	return model.User{}, false
}

// Save menyimpan user (upsert berdasarkan ID).
func (r *UserRepo) Save(user model.User) error {
	return r.c.Mutate(func(items []model.User) ([]model.User, error) {
		for i := range items {
			if items[i].ID == user.ID {
				items[i] = user
				return items, nil
			}
		}
		return append(items, user), nil
	})
}
