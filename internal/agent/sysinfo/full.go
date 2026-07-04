package sysinfo

import (
	"net"
	"os/user"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/yusufpapurcu/wmi"

	"remote_pc/internal/protocol"
)

// Struktur WMI (hanya field yang dipakai).
type win32ComputerSystem struct {
	Manufacturer        string
	Model               string
	TotalPhysicalMemory uint64
}
type win32BIOS struct {
	SerialNumber      string
	SMBIOSBIOSVersion string
	Manufacturer      string
}
type win32BaseBoard struct {
	Product      string
	Manufacturer string
}
type win32Processor struct {
	Name          string
	NumberOfCores uint32
}
type win32VideoController struct {
	Name           string
	AdapterRAM     uint32
	DriverVersion  string
}
type win32DiskDrive struct {
	Model string
	Size  uint64
}

// FullInfo mengumpulkan informasi sistem lengkap. Setiap sumber bersifat
// best-effort: kegagalan satu query tidak menggagalkan keseluruhan.
func FullInfo() (protocol.SysInfo, error) {
	info := protocol.SysInfo{}

	if h, err := host.Info(); err == nil {
		info.Hostname = h.Hostname
		info.OS = composeVersion(h.Platform, h.PlatformVersion)
		info.Build = h.PlatformVersion
	}
	if u, err := user.Current(); err == nil {
		info.Username = u.Username
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		info.RAMTotalMB = vm.Total / (1024 * 1024)
	}

	var cs []win32ComputerSystem
	if err := wmi.Query("SELECT Manufacturer, Model, TotalPhysicalMemory FROM Win32_ComputerSystem", &cs); err == nil && len(cs) > 0 {
		info.Manufacturer = cs[0].Manufacturer
		info.Model = cs[0].Model
	}
	var bios []win32BIOS
	if err := wmi.Query("SELECT SerialNumber, SMBIOSBIOSVersion, Manufacturer FROM Win32_BIOS", &bios); err == nil && len(bios) > 0 {
		info.Serial = strings.TrimSpace(bios[0].SerialNumber)
		info.BIOS = strings.TrimSpace(bios[0].Manufacturer + " " + bios[0].SMBIOSBIOSVersion)
	}
	var board []win32BaseBoard
	if err := wmi.Query("SELECT Product, Manufacturer FROM Win32_BaseBoard", &board); err == nil && len(board) > 0 {
		info.Motherboard = strings.TrimSpace(board[0].Manufacturer + " " + board[0].Product)
	}
	var cpus []win32Processor
	if err := wmi.Query("SELECT Name, NumberOfCores FROM Win32_Processor", &cpus); err == nil && len(cpus) > 0 {
		info.CPU = strings.TrimSpace(cpus[0].Name)
		for _, c := range cpus {
			info.CPUCores += int(c.NumberOfCores)
		}
	}
	var gpus []win32VideoController
	if err := wmi.Query("SELECT Name, AdapterRAM, DriverVersion FROM Win32_VideoController", &gpus); err == nil {
		for _, g := range gpus {
			info.GPUs = append(info.GPUs, protocol.GPUInfo{
				Name: g.Name, RAMMB: uint64(g.AdapterRAM) / (1024 * 1024), Driver: g.DriverVersion,
			})
		}
	}
	var disks []win32DiskDrive
	if err := wmi.Query("SELECT Model, Size FROM Win32_DiskDrive", &disks); err == nil {
		for _, d := range disks {
			info.Disks = append(info.Disks, protocol.DiskInfo{
				Name: d.Model, SizeGB: d.Size / (1024 * 1024 * 1024),
			})
		}
	}
	info.Adapters = networkAdapters()
	return info, nil
}

// networkAdapters mengambil adapter jaringan aktif (via net, lebih andal dari WMI).
func networkAdapters() []protocol.NetAdapter {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []protocol.NetAdapter
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 || ifc.HardwareAddr.String() == "" {
			continue
		}
		ip := ""
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if v4 := ipnet.IP.To4(); v4 != nil {
					ip = v4.String()
					break
				}
			}
		}
		out = append(out, protocol.NetAdapter{Name: ifc.Name, MAC: ifc.HardwareAddr.String(), IP: ip})
	}
	return out
}
