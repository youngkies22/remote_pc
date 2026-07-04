package auth

import (
	"time"

	"github.com/google/uuid"

	"remote_pc/internal/model"
	"remote_pc/internal/storage"
)

// DefaultAdminUsername dan DefaultAdminPassword dipakai saat belum ada user apa pun.
const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123"
)

// EnsureDefaultAdmin membuat user admin default bila users.json masih kosong.
// Mengembalikan true bila user baru dibuat (agar pemanggil bisa memberi peringatan
// untuk segera mengganti password).
func EnsureDefaultAdmin(users *storage.UserRepo) (bool, error) {
	if users.Count() > 0 {
		return false, nil
	}
	hash, err := HashPassword(DefaultAdminPassword)
	if err != nil {
		return false, err
	}
	admin := model.User{
		ID:           uuid.NewString(),
		Username:     DefaultAdminUsername,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		CreatedAt:    time.Now(),
	}
	if err := users.Save(admin); err != nil {
		return false, err
	}
	return true, nil
}
