// Package api berisi handler REST untuk dashboard operator (login, device, statistik).
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"remote_pc/internal/auth"
	"remote_pc/internal/server/ws"
	"remote_pc/internal/storage"
)

// API mengelompokkan dependency yang dibutuhkan handler REST.
type API struct {
	store          *storage.Store
	tokens         *auth.TokenManager
	hub            *ws.Hub
	log            *zap.Logger
	cookieTTL      time.Duration
	secure         bool // set true bila server berjalan di atas TLS (cookie Secure)
	screenshotsDir string
}

// Config berisi parameter pembuatan API.
type Config struct {
	Store          *storage.Store
	Tokens         *auth.TokenManager
	Hub            *ws.Hub
	Log            *zap.Logger
	CookieTTL      time.Duration
	SecureCookie   bool
	ScreenshotsDir string
}

// New membuat instance API.
func New(cfg Config) *API {
	return &API{
		store:          cfg.Store,
		tokens:         cfg.Tokens,
		hub:            cfg.Hub,
		log:            cfg.Log,
		cookieTTL:      cfg.CookieTTL,
		secure:         cfg.SecureCookie,
		screenshotsDir: cfg.ScreenshotsDir,
	}
}

// writeJSON menulis respons JSON dengan status code tertentu.
func (a *API) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			a.log.Error("gagal encode JSON", zap.Error(err))
		}
	}
}

// writeError menulis respons error JSON standar.
func (a *API) writeError(w http.ResponseWriter, status int, msg string) {
	a.writeJSON(w, status, map[string]string{"error": msg})
}
