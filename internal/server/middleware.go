package server

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// middleware adalah pembungkus http.Handler.
type middleware func(http.Handler) http.Handler

// chain menerapkan sejumlah middleware ke handler (yang terluar dijalankan lebih dulu).
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// recoverMiddleware menangkap panic pada handler agar server tidak crash dan
// mencatatnya sebagai error.
func recoverMiddleware(log *zap.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic pada handler HTTP",
						zap.Any("panic", rec), zap.String("path", r.URL.Path))
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder mencatat status code respons untuk keperluan logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Hijack meneruskan ke ResponseWriter asli agar upgrade WebSocket tetap berfungsi
// meski respons dibungkus middleware ini.
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter tidak mendukung Hijack")
	}
	return hj.Hijack()
}

// requestLogger mencatat setiap request HTTP (metode, path, status, durasi).
func requestLogger(log *zap.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Debug("http",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rec.status),
				zap.Duration("dur", time.Since(start)))
		})
	}
}
