// Package model berisi struktur data bersama yang dipakai server maupun agent.
package model

import "time"

// DeviceStatus merepresentasikan status koneksi sebuah device.
type DeviceStatus string

const (
	// StatusOnline berarti device masih mengirim heartbeat dalam ambang waktu.
	StatusOnline DeviceStatus = "online"
	// StatusOffline berarti device melewati ambang batas heartbeat.
	StatusOffline DeviceStatus = "offline"
)

// Metrics adalah metrik runtime yang dikirim agent pada setiap heartbeat.
type Metrics struct {
	CPUPercent  float64 `json:"cpu_percent"`
	RAMPercent  float64 `json:"ram_percent"`
	RAMTotalMB  uint64  `json:"ram_total_mb"`
	RAMUsedMB   uint64  `json:"ram_used_mb"`
	DiskPercent float64 `json:"disk_percent"`
	DiskTotalGB uint64  `json:"disk_total_gb"`
	DiskUsedGB  uint64  `json:"disk_used_gb"`
	UptimeSec   uint64  `json:"uptime_sec"`
	// BatteryPercent & NetworkType hanya diisi oleh agent Android (omitempty
	// agar agent Windows yang tidak mengirimnya tidak memengaruhi payload lama).
	BatteryPercent float64 `json:"battery_percent,omitempty"`
	NetworkType    string  `json:"network_type,omitempty"` // "wifi" | "cellular" | "none"
}

// Device merepresentasikan satu komputer yang dimonitor. Disimpan di devices.json.
type Device struct {
	ID             string       `json:"id"`
	Token          string       `json:"token"` // device token untuk autentikasi agent
	Hostname       string       `json:"hostname"`
	Username       string       `json:"username"`
	IP             string       `json:"ip"`
	MAC            string       `json:"mac"`
	OS             string       `json:"os"`             // contoh: "Windows"
	WindowsVersion string       `json:"windows_version"` // contoh: "Windows 11 Pro 10.0.26200"
	Arch           string       `json:"arch"`
	Metrics        Metrics      `json:"metrics"`
	Status         DeviceStatus `json:"status"`
	FirstSeen      time.Time    `json:"first_seen"`
	LastSeen       time.Time    `json:"last_seen"`
}

// RegisterInfo adalah data statis yang dikirim agent saat registrasi awal.
type RegisterInfo struct {
	DeviceID       string `json:"device_id"` // kosong bila belum pernah registrasi
	Token          string `json:"token"`     // kosong pada registrasi pertama
	Hostname       string `json:"hostname"`
	Username       string `json:"username"`
	IP             string `json:"ip"`
	MAC            string `json:"mac"`
	OS             string `json:"os"`
	WindowsVersion string `json:"windows_version"`
	Arch           string `json:"arch"`
}

// RegisterResult adalah balasan server atas registrasi agent.
type RegisterResult struct {
	DeviceID string `json:"device_id"`
	Token    string `json:"token"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message,omitempty"`
}

// Heartbeat adalah payload yang dikirim agent secara periodik.
type Heartbeat struct {
	Hostname string  `json:"hostname"`
	Username string  `json:"username"`
	IP       string  `json:"ip"`
	MAC      string  `json:"mac"`
	Metrics  Metrics `json:"metrics"`
}
