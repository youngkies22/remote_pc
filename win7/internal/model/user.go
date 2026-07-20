package model

import "time"

// Role menentukan tingkat hak akses seorang user.
type Role string

const (
	// RoleAdmin memiliki akses penuh ke seluruh fitur.
	RoleAdmin Role = "admin"
	// RoleOperator dapat memonitor dan mengendalikan device.
	RoleOperator Role = "operator"
)

// User adalah operator dashboard. Disimpan di users.json.
// PasswordHash adalah hash bcrypt, bukan password mentah.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// Session mencatat sesi login operator (untuk audit/history). Disimpan di sessions.json.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
