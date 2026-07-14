//go:build windows

package discovery

import (
	"encoding/json"
	"fmt"
	"net"
	"syscall"
	"time"
)

// Discover menyiarkan probe UDP ke seluruh LAN dan menunggu balasan server
// pertama. Mengembalikan IP server (dari alamat sumber balasan), port WebSocket,
// dan mode TLS. timeout membatasi lama tunggu total. Dipanggil agent (Windows).
func Discover(port int, timeout time.Duration) (host string, wsPort int, useTLS bool, err error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return "", 0, false, err
	}
	defer conn.Close()

	// Izinkan pengiriman ke alamat broadcast (SO_BROADCAST). Tanpa ini, WriteToUDP
	// ke 255.255.255.255 ditolak oleh OS.
	if rc, cerr := conn.SyscallConn(); cerr == nil {
		_ = rc.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		})
	}

	for _, t := range broadcastTargets(port) {
		_, _ = conn.WriteToUDP([]byte(probeMagic), t)
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 512)
	for {
		n, src, rerr := conn.ReadFromUDP(buf)
		if rerr != nil {
			return "", 0, false, fmt.Errorf("tidak ada server yang menjawab dalam %s: %w", timeout, rerr)
		}
		var rp reply
		if json.Unmarshal(buf[:n], &rp) != nil || rp.Service != serviceName {
			continue // paket asing, abaikan
		}
		p := rp.Port
		if p == 0 {
			p = port
		}
		return src.IP.String(), p, rp.TLS, nil
	}
}
