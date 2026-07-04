// Package wol mengirim Wake-on-LAN magic packet untuk menyalakan PC dari jarak
// jauh lewat jaringan.
package wol

import (
	"fmt"
	"net"
	"strings"
)

// Send mengirim magic packet ke alamat MAC tertentu lewat UDP broadcast (port
// 9). Ini murni operasi jaringan L2/L3 — tidak melibatkan agent maupun
// WebSocket sama sekali, sehingga bekerja walau device sedang mati total,
// SELAMA fitur Wake-on-LAN sudah diaktifkan di BIOS/UEFI dan pengaturan power
// management NIC PC target. Itu konfigurasi hardware yang harus disetel sekali
// secara langsung di PC itu — tidak bisa diaktifkan dari jarak jauh.
func Send(mac string) error {
	hw, err := net.ParseMAC(normalize(mac))
	if err != nil || len(hw) != 6 {
		return fmt.Errorf("alamat MAC tidak valid: %q", mac)
	}

	packet := make([]byte, 0, 102)
	for i := 0; i < 6; i++ {
		packet = append(packet, 0xFF)
	}
	for i := 0; i < 16; i++ {
		packet = append(packet, hw...)
	}

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return fmt.Errorf("wol: buka socket UDP: %w", err)
	}
	defer conn.Close()

	dst := &net.UDPAddr{IP: net.IPv4bcast, Port: 9}
	if _, err := conn.WriteTo(packet, dst); err != nil {
		return fmt.Errorf("wol: kirim magic packet: %w", err)
	}
	return nil
}

// normalize menyeragamkan format MAC (menerima '-' atau ':') sebelum diparse.
func normalize(mac string) string {
	return strings.ReplaceAll(strings.TrimSpace(mac), "-", ":")
}
