// Package sysinfo mengumpulkan informasi statis dan metrik runtime dari host
// Windows menggunakan gopsutil (tanpa cgo).
package sysinfo

import (
	"context"
	"net"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"remote_pc/internal/model"
)

// Static adalah informasi host yang relatif tidak berubah selama runtime.
type Static struct {
	Hostname       string
	Username       string
	IP             string
	MAC            string
	OS             string
	WindowsVersion string
	Arch           string
}

// Collect mengumpulkan informasi statis host.
func Collect() (Static, error) {
	s := Static{Arch: runtime.GOARCH, IP: localIP(), MAC: primaryMAC()}

	if info, err := host.Info(); err == nil {
		s.Hostname = info.Hostname
		s.OS = capitalize(info.OS) // "windows" -> "Windows"
		s.WindowsVersion = composeVersion(info.Platform, info.PlatformVersion)
		if info.KernelArch != "" {
			s.Arch = info.KernelArch
		}
	}
	if u, err := user.Current(); err == nil {
		s.Username = u.Username
	}
	return s, nil
}

// PrimeCPU melakukan pembacaan CPU awal agar pembacaan berikutnya (delta) akurat.
func PrimeCPU() {
	_, _ = cpu.Percent(0, false)
}

// CollectMetrics mengumpulkan metrik runtime (CPU, RAM, disk, uptime).
// ctx dipakai membatasi pembacaan CPU yang membutuhkan sedikit waktu sampling.
func CollectMetrics(ctx context.Context) model.Metrics {
	var m model.Metrics

	if pcts, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(pcts) > 0 {
		m.CPUPercent = round1(pcts[0])
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		m.RAMPercent = round1(vm.UsedPercent)
		m.RAMTotalMB = vm.Total / (1024 * 1024)
		m.RAMUsedMB = vm.Used / (1024 * 1024)
	}
	if du, err := disk.UsageWithContext(ctx, systemDrive()); err == nil {
		m.DiskPercent = round1(du.UsedPercent)
		m.DiskTotalGB = du.Total / (1024 * 1024 * 1024)
		m.DiskUsedGB = du.Used / (1024 * 1024 * 1024)
	}
	if up, err := host.UptimeWithContext(ctx); err == nil {
		m.UptimeSec = up
	}
	return m
}

func composeVersion(platform, version string) string {
	platform = strings.TrimSpace(platform)
	version = strings.TrimSpace(version)
	switch {
	case platform != "" && version != "":
		return platform + " " + version
	case platform != "":
		return platform
	default:
		return version
	}
}

func systemDrive() string {
	// Windows menggunakan C:\ sebagai drive sistem umum.
	return "C:\\"
}

// localIP mencari alamat IPv4 lokal yang dipakai untuk koneksi keluar.
func localIP() string {
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 2*time.Second)
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return addr.IP.String()
		}
	}
	return firstInterfaceIP()
}

func firstInterfaceIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return ""
}

func primaryMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		if mac := ifc.HardwareAddr.String(); mac != "" {
			return mac
		}
	}
	return ""
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
