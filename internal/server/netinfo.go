package server

import (
	"fmt"
	"net"

	"go.uber.org/zap"
)

// logReachableURLs mencetak alamat yang bisa dipakai untuk membuka dashboard dan
// untuk dikonfigurasikan pada agent siswa. Berguna saat server bind ke 0.0.0.0.
func (s *Server) logReachableURLs() {
	scheme := "http"
	wsScheme := "ws"
	if s.cfg.Server.TLS.Enabled {
		scheme, wsScheme = "https", "wss"
	}
	port := s.cfg.Server.Port

	ips := lanIPv4s()
	if len(ips) == 0 {
		return
	}
	for _, ip := range ips {
		s.log.App.Info("alamat server terdeteksi",
			zap.String("dashboard", fmt.Sprintf("%s://%s:%d", scheme, ip, port)),
			zap.String("agent_server_url", fmt.Sprintf("%s://%s:%d/ws/agent", wsScheme, ip, port)))
	}
	s.log.App.Info("isikan agent_server_url di atas ke config/agent.yaml pada PC siswa, " +
		"lalu buka Windows Firewall untuk port tersebut")
}

// lanIPv4s mengembalikan daftar alamat IPv4 non-loopback pada interface yang aktif.
func lanIPv4s() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
				out = append(out, ip4.String())
			}
		}
	}
	return out
}
