package auth

import (
	"context"
	"net/http"
	"strings"
)

// CookieName adalah nama cookie penyimpan token JWT operator.
const CookieName = "rp_token"

type ctxKey struct{}

// Middleware menyediakan proteksi rute berbasis JWT.
type Middleware struct {
	tokens *TokenManager
}

// NewMiddleware membuat Middleware dengan TokenManager tertentu.
func NewMiddleware(tokens *TokenManager) *Middleware {
	return &Middleware{tokens: tokens}
}

// API memproteksi endpoint REST: mengembalikan 401 JSON bila token tidak valid.
func (m *Middleware) API(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := m.authenticate(r)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
	})
}

// Page memproteksi halaman: mengalihkan ke /login bila token tidak valid.
func (m *Middleware) Page(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := m.authenticate(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
	})
}

// authenticate mengekstrak token dari cookie atau header Authorization lalu memverifikasinya.
func (m *Middleware) authenticate(r *http.Request) (*Claims, bool) {
	token := tokenFromRequest(r)
	if token == "" {
		return nil, false
	}
	claims, err := m.tokens.Verify(token)
	if err != nil {
		return nil, false
	}
	return claims, true
}

func tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// ClaimsFrom mengambil klaim operator dari context request (bila ada).
func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok
}
