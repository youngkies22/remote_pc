// Package winservices menyediakan daftar dan kontrol Windows Service.
package winservices

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"remote_pc/internal/protocol"
)

// List mengembalikan seluruh service beserta status. Menggunakan akses SCM
// read-only sehingga tidak memerlukan hak administrator.
func List() (protocol.SvcListResponse, error) {
	handle, err := windows.OpenSCManager(nil, nil,
		windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return protocol.SvcListResponse{}, fmt.Errorf("buka SCM: %w", err)
	}
	m := &mgr.Mgr{Handle: handle}
	defer m.Disconnect()

	names, err := m.ListServices()
	if err != nil {
		return protocol.SvcListResponse{}, err
	}
	out := make([]protocol.SvcInfo, 0, len(names))
	for _, name := range names {
		info := protocol.SvcInfo{Name: name, Display: name, Status: "unknown"}
		if s, err := openQuery(m, name); err == nil {
			if cfg, err := s.Config(); err == nil && cfg.DisplayName != "" {
				info.Display = cfg.DisplayName
			}
			if st, err := s.Query(); err == nil {
				info.Status = stateString(st.State)
			}
			windows.CloseServiceHandle(s.Handle)
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Display) < strings.ToLower(out[j].Display)
	})
	return protocol.SvcListResponse{Services: out}, nil
}

// openQuery membuka service dengan akses query saja (tersedia untuk user biasa).
func openQuery(m *mgr.Mgr, name string) (*mgr.Service, error) {
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := windows.OpenService(m.Handle, ptr,
		windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return nil, err
	}
	return &mgr.Service{Name: name, Handle: h}, nil
}

// Control menjalankan start/stop/restart pada service. Operasi ini memerlukan
// hak administrator.
func Control(name, action string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("perlu hak administrator untuk mengontrol service: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q tidak ditemukan: %w", name, err)
	}
	defer s.Close()

	switch action {
	case "start":
		return s.Start()
	case "stop":
		_, err := s.Control(svc.Stop)
		return err
	case "restart":
		if err := stopAndWait(s); err != nil {
			return err
		}
		return s.Start()
	default:
		return fmt.Errorf("aksi tidak dikenal: %q", action)
	}
}

// stopAndWait menghentikan service lalu menunggu sampai benar-benar berhenti.
func stopAndWait(s *mgr.Service) error {
	status, err := s.Control(svc.Stop)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(20 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout menunggu service berhenti")
		}
		time.Sleep(300 * time.Millisecond)
		if status, err = s.Query(); err != nil {
			return err
		}
	}
	return nil
}

func stateString(s svc.State) string {
	switch s {
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "start_pending"
	case svc.StopPending:
		return "stop_pending"
	case svc.Running:
		return "running"
	case svc.ContinuePending:
		return "continue_pending"
	case svc.PausePending:
		return "pause_pending"
	case svc.Paused:
		return "paused"
	default:
		return "unknown"
	}
}
