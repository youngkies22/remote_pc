// Package winsvc menyediakan integrasi Windows Service untuk agent: menjalankan
// agent di bawah Service Control Manager (SCM) serta install/uninstall/kontrol.
package winsvc

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// RunFunc adalah fungsi kerja agent yang berhenti saat ctx dibatalkan.
type RunFunc func(ctx context.Context) error

// IsService melaporkan apakah proses sedang dijalankan oleh SCM (bukan konsol).
func IsService() (bool, error) {
	return svc.IsWindowsService()
}

// Run menjalankan agent di bawah SCM. Fungsi run dieksekusi di goroutine dan
// dihentikan lewat pembatalan context saat SCM meminta Stop/Shutdown.
func Run(name string, run RunFunc) error {
	return svc.Run(name, &handler{run: run})
}

type handler struct {
	run RunFunc
}

// Execute mengimplementasikan svc.Handler.
func (h *handler) Execute(_ []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.run(ctx)
	}()

	status <- svc.Status{State: svc.Running, Accepts: accepted}
loop:
	for {
		select {
		case c := <-req:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			}
		case <-done:
			break loop
		}
	}

	cancel()
	status <- svc.Status{State: svc.StopPending}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
	return false, 0
}

// Install mendaftarkan service dengan start otomatis dan argumen exec tertentu.
func Install(name, displayName, description, exePath string, args ...string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	if s, err := m.OpenService(name); err == nil {
		s.Close()
		return fmt.Errorf("service %q sudah terpasang", name)
	}
	s, err := m.CreateService(name, exePath, mgr.Config{
		DisplayName: displayName,
		Description: description,
		StartType:   mgr.StartAutomatic,
	}, args...)
	if err != nil {
		return err
	}
	defer s.Close()
	return nil
}

// Uninstall menghentikan (bila berjalan) dan menghapus service.
func Uninstall(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q tidak ditemukan: %w", name, err)
	}
	defer s.Close()

	_, _ = s.Control(svc.Stop)
	return s.Delete()
}

// Control mengirim perintah (Start/Stop) ke service.
func Control(name string, start bool) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q tidak ditemukan: %w", name, err)
	}
	defer s.Close()

	if start {
		return s.Start()
	}
	_, err = s.Control(svc.Stop)
	return err
}
