// Package discovery menyediakan penemuan server otomatis di LAN lewat UDP
// broadcast. Tujuannya: agent (PC siswa) cukup di-install tanpa perlu diberi tahu
// IP/port server — agent menyiarkan probe ke jaringan, server yang mendengar
// membalas dengan port & mode TLS-nya, dan alamat IP server diketahui dari alamat
// sumber paket balasan.
//
// Batasan: UDP broadcast hanya menjangkau satu segmen LAN (broadcast domain) yang
// sama. Bila server dan agent dipisah oleh router/subnet berbeda, broadcast tidak
// menyeberang — dalam kasus itu pakai konfigurasi IP manual (server_host).
package discovery

import (
	"context"
	"encoding/json"
	"net"

	"go.uber.org/zap"
)

// probeMagic adalah isi persis paket probe yang dikirim agent. Server hanya
// membalas paket yang cocok dengan string ini.
const probeMagic = "REMOTEPC_DISCOVER_1"

// serviceName menandai balasan agar agent yakin balasan berasal dari server
// Remote PC (bukan layanan UDP lain yang kebetulan di port sama).
const serviceName = "remote_pc"

// reply adalah balasan server berisi info koneksi WebSocket. Host tidak disertakan
// karena agent memakai alamat sumber paket balasan sebagai IP server.
type reply struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
	TLS     bool   `json:"tls"`
}

// Serve mendengarkan probe UDP broadcast pada port yang sama dengan port
// WebSocket server, lalu membalas dengan port & mode TLS. Berjalan sampai ctx
// dibatalkan. Aman dipanggil di goroutine terpisah; kegagalan bind hanya
// menonaktifkan auto-discovery (server tetap jalan seperti biasa).
func Serve(ctx context.Context, port int, useTLS bool, log *zap.Logger) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Warn("auto-discovery nonaktif: gagal bind UDP (agent harus memakai IP manual)",
			zap.Int("port", port), zap.Error(err))
		return
	}
	log.Info("auto-discovery aktif: agent bisa menemukan server tanpa setting IP",
		zap.Int("udp_port", port))

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	respBytes, _ := json.Marshal(reply{Service: serviceName, Port: port, TLS: useTLS})
	buf := make([]byte, 512)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return // ditutup saat shutdown
			}
			continue
		}
		if string(buf[:n]) != probeMagic {
			continue
		}
		if _, err := conn.WriteToUDP(respBytes, src); err != nil {
			log.Debug("gagal membalas probe discovery", zap.Error(err))
		}
	}
}

// broadcastTargets mengumpulkan alamat broadcast tujuan probe: broadcast global
// (255.255.255.255) plus broadcast tiap interface aktif (lebih andal di host
// dengan banyak adapter jaringan).
func broadcastTargets(port int) []*net.UDPAddr {
	out := []*net.UDPAddr{{IP: net.IPv4bcast, Port: port}}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 || ifc.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if bc := broadcastAddr(ipnet.IP, ipnet.Mask); bc != nil {
				out = append(out, &net.UDPAddr{IP: bc, Port: port})
			}
		}
	}
	return out
}

// broadcastAddr menghitung alamat broadcast dari IP + netmask (IPv4 saja).
func broadcastAddr(ip net.IP, mask net.IPMask) net.IP {
	ip4 := ip.To4()
	if ip4 == nil || len(mask) != 4 {
		return nil
	}
	bc := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		bc[i] = ip4[i] | ^mask[i]
	}
	return bc
}
