// Package server merangkai konfigurasi, storage, logger, WebSocket, dan REST API
// menjadi satu HTTP server yang dapat dijalankan dan dimatikan dengan rapi.
package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"go.uber.org/zap"

	"remote_pc/internal/auth"
	"remote_pc/internal/config"
	"remote_pc/internal/discovery"
	"remote_pc/internal/logger"
	"remote_pc/internal/server/api"
	"remote_pc/internal/server/ws"
	"remote_pc/internal/storage"
	"remote_pc/web"
)

// Server membungkus http.Server beserta dependency-nya.
type Server struct {
	cfg        *config.ServerConfig
	log        *logger.Loggers
	store      *storage.Store
	hub        *ws.Hub
	httpServer *http.Server
}

// New membangun Server lengkap dengan seluruh rute terpasang.
func New(cfg *config.ServerConfig, store *storage.Store, log *logger.Loggers) (*Server, error) {
	hub := ws.NewHub(log.WS)
	tokens := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiryHours)
	mw := auth.NewMiddleware(tokens)

	apiHandler := api.New(api.Config{
		Store:          store,
		Tokens:         tokens,
		Hub:            hub,
		Log:            log.App,
		CookieTTL:      time.Duration(cfg.Auth.JWTExpiryHours) * time.Hour,
		SecureCookie:   cfg.Server.TLS.Enabled,
		ScreenshotsDir: cfg.Storage.ScreenshotsDir,
	})
	wsHandler := ws.NewHandler(hub, store.Devices, log.WS)

	mux := http.NewServeMux()
	if err := registerRoutes(mux, apiHandler, wsHandler, mw); err != nil {
		return nil, err
	}

	s := &Server{
		cfg:   cfg,
		log:   log,
		store: store,
		hub:   hub,
		httpServer: &http.Server{
			Addr:              cfg.Server.Addr(),
			Handler:           chain(mux, recoverMiddleware(log.App), requestLogger(log.App)),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
	return s, nil
}

// Run memulai server dan sweeper offline, lalu menunggu ctx dibatalkan untuk
// melakukan graceful shutdown. Blocking sampai server benar-benar berhenti.
func (s *Server) Run(ctx context.Context) error {
	sweepCtx, stopSweep := context.WithCancel(ctx)
	go s.runOfflineSweeper(sweepCtx)

	// Responder auto-discovery: agent di LAN yang sama bisa menemukan server ini
	// tanpa perlu disetel IP/port-nya secara manual.
	go discovery.Serve(sweepCtx, s.cfg.Server.Port, s.cfg.Server.TLS.Enabled, s.log.App)

	errCh := make(chan error, 1)
	go func() {
		s.log.App.Info("server berjalan", zap.String("addr", s.cfg.Server.Addr()),
			zap.Bool("tls", s.cfg.Server.TLS.Enabled))
		errCh <- s.listen()
	}()
	s.logReachableURLs()

	select {
	case <-ctx.Done():
		stopSweep()
		return s.shutdown()
	case err := <-errCh:
		stopSweep()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func (s *Server) listen() error {
	if s.cfg.Server.TLS.Enabled {
		return s.httpServer.ListenAndServeTLS(s.cfg.Server.TLS.CertFile, s.cfg.Server.TLS.KeyFile)
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) shutdown() error {
	s.log.App.Info("mematikan server...")
	s.hub.CloseAll()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// runOfflineSweeper secara periodik menandai device yang berhenti mengirim
// heartbeat sebagai offline.
func (s *Server) runOfflineSweeper(ctx context.Context) {
	threshold := time.Duration(s.cfg.Heartbeat.OfflineAfterSeconds) * time.Second
	ticker := time.NewTicker(time.Duration(s.cfg.Heartbeat.IntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := s.store.Devices.MarkStaleOffline(threshold)
			if err != nil {
				s.log.App.Error("sweeper offline gagal", zap.Error(err))
				continue
			}
			for _, id := range changed {
				s.log.App.Info("device menjadi offline (heartbeat kadaluarsa)",
					zap.String("device_id", id))
			}
		}
	}
}

// staticFS mengembalikan sub-filesystem untuk aset statis yang di-embed.
func staticFS() (fs.FS, error) {
	return fs.Sub(web.FS, "static")
}
