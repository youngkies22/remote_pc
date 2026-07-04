package api

import (
	"encoding/json"
	"net/http"
	"time"

	"remote_pc/internal/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login memvalidasi kredensial, menerbitkan JWT, dan menyetel cookie.
func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if req.Username == "" || req.Password == "" {
		a.writeError(w, http.StatusBadRequest, "username dan password wajib diisi")
		return
	}

	user, ok := a.store.Users.GetByUsername(req.Username)
	if !ok || !auth.CheckPassword(user.PasswordHash, req.Password) {
		a.writeError(w, http.StatusUnauthorized, "username atau password salah")
		return
	}

	token, expiresAt, err := a.tokens.Issue(user)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "gagal menerbitkan token")
		return
	}

	user.LastLogin = time.Now()
	if err := a.store.Users.Save(user); err != nil {
		a.log.Warn("gagal memperbarui last_login")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  userDTO{ID: user.ID, Username: user.Username, Role: string(user.Role)},
	})
}

// Logout menghapus cookie sesi operator.
func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Me mengembalikan identitas operator yang sedang login (dari klaim JWT).
func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		a.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	a.writeJSON(w, http.StatusOK, userDTO{
		ID: claims.UserID, Username: claims.Username, Role: string(claims.Role),
	})
}
