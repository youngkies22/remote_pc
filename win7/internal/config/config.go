// Package config memuat dan memvalidasi konfigurasi server dan agent dari file YAML.
// Semua nilai memiliki default yang wajar sehingga aplikasi tetap berjalan bila
// sebagian field tidak diisi.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// TLSConfig menyimpan pengaturan sertifikat untuk HTTPS/WSS.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// ServerSection menyimpan pengaturan HTTP server.
type ServerSection struct {
	Host string    `yaml:"host"`
	Port int       `yaml:"port"`
	TLS  TLSConfig `yaml:"tls"`
}

// Addr mengembalikan alamat listen dalam format host:port.
func (s ServerSection) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// AuthSection menyimpan pengaturan autentikasi operator.
type AuthSection struct {
	JWTSecret      string `yaml:"jwt_secret"`
	JWTExpiryHours int    `yaml:"jwt_expiry_hours"`
}

// StorageSection menyimpan lokasi direktori penyimpanan berbasis file.
type StorageSection struct {
	DataDir        string `yaml:"data_dir"`
	LogsDir        string `yaml:"logs_dir"`
	ScreenshotsDir string `yaml:"screenshots_dir"`
	UploadsDir     string `yaml:"uploads_dir"`
	DownloadsDir   string `yaml:"downloads_dir"`
}

// HeartbeatSection mengatur ambang online/offline device.
type HeartbeatSection struct {
	IntervalSeconds     int `yaml:"interval_seconds"`
	OfflineAfterSeconds int `yaml:"offline_after_seconds"`
}

// LoggingSection mengatur level dan rotasi log.
type LoggingSection struct {
	Level      string `yaml:"level"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
}

// ServerConfig adalah konfigurasi lengkap untuk aplikasi server.
type ServerConfig struct {
	Server    ServerSection    `yaml:"server"`
	Auth      AuthSection      `yaml:"auth"`
	Storage   StorageSection   `yaml:"storage"`
	Heartbeat HeartbeatSection `yaml:"heartbeat"`
	Logging   LoggingSection   `yaml:"logging"`
}

// AgentSection menyimpan identitas dan pengaturan koneksi agent.
//
// Cara mengarahkan agent ke server:
//   - ServerHost = "auto" atau dikosongkan (default): agent MENEMUKAN server
//     otomatis di LAN lewat UDP broadcast (tidak perlu tahu IP/port sama sekali).
//   - ServerHost diisi IP: agent langsung ke IP itu (dipakai bila server & agent
//     beda subnet sehingga broadcast tidak menjangkau).
//
// ServerURL hanya dipakai langsung bila ditulis manual (kompatibilitas config lama).
type AgentSection struct {
	ServerHost       string `yaml:"server_host,omitempty"`
	ServerPort       int    `yaml:"server_port,omitempty"`
	UseTLS           bool   `yaml:"use_tls,omitempty"`
	ServerURL        string `yaml:"server_url,omitempty"`
	DeviceID         string `yaml:"device_id"`
	DeviceToken      string `yaml:"device_token"`
	ReconnectSeconds int    `yaml:"reconnect_seconds"`
	HeartbeatSeconds int    `yaml:"heartbeat_seconds"`

	// AutoDiscover dihitung saat applyDefaults (bukan dari file). Bila true, agent
	// mencari server via UDP broadcast alih-alih memakai ServerURL tetap.
	AutoDiscover bool `yaml:"-"`
}

// AgentConfig adalah konfigurasi lengkap untuk aplikasi agent.
type AgentConfig struct {
	Agent   AgentSection   `yaml:"agent"`
	Logging LoggingSection `yaml:"logging"`

	path string // jalur file agar bisa disimpan ulang setelah registrasi
}

// LoadServerConfig membaca konfigurasi server dari path lalu mengisi default.
func LoadServerConfig(path string) (*ServerConfig, error) {
	cfg := defaultServerConfig()
	if err := readYAML(path, cfg); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return cfg, nil
}

// LoadAgentConfig membaca konfigurasi agent dari path lalu mengisi default.
func LoadAgentConfig(path string) (*AgentConfig, error) {
	cfg := defaultAgentConfig()
	if err := readYAML(path, cfg); err != nil {
		return nil, err
	}
	cfg.path = path
	cfg.applyDefaults()
	return cfg, nil
}

// Save menulis konfigurasi agent kembali ke file (dipakai setelah registrasi
// untuk menyimpan device_id dan device_token yang diberikan server).
func (c *AgentConfig) Save() error {
	if c.path == "" {
		return fmt.Errorf("config: path agent kosong, tidak bisa menyimpan")
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal agent: %w", err)
	}
	if err := os.WriteFile(c.path, data, 0o600); err != nil {
		return fmt.Errorf("config: tulis agent: %w", err)
	}
	return nil
}

// readYAML membaca file YAML ke dalam out. File yang tidak ada bukan error —
// nilai default tetap dipakai.
func readYAML(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: baca %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	return nil
}

func defaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Server:    ServerSection{Host: "127.0.0.1", Port: 9000},
		Auth:      AuthSection{JWTSecret: "ubah-secret-ini-di-produksi", JWTExpiryHours: 24},
		Storage:   defaultStorage(),
		Heartbeat: HeartbeatSection{IntervalSeconds: 5, OfflineAfterSeconds: 15},
		Logging:   defaultLogging(),
	}
}

func defaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		Agent: AgentSection{
			// Kosong = mode auto-discovery (default): tanpa config apa pun, agent
			// tetap bisa menemukan server di LAN.
			ServerHost:       "auto",
			ServerPort:       9000,
			ReconnectSeconds: 5,
			HeartbeatSeconds: 2,
		},
		Logging: defaultLogging(),
	}
}

func defaultStorage() StorageSection {
	return StorageSection{
		DataDir:        "data",
		LogsDir:        "logs",
		ScreenshotsDir: "screenshots",
		UploadsDir:     "uploads",
		DownloadsDir:   "downloads",
	}
}

func defaultLogging() LoggingSection {
	return LoggingSection{Level: "info", MaxSizeMB: 50, MaxBackups: 5, MaxAgeDays: 30}
}

func (c *ServerConfig) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 9000
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "ubah-secret-ini-di-produksi"
	}
	if c.Auth.JWTExpiryHours == 0 {
		c.Auth.JWTExpiryHours = 24
	}
	if c.Heartbeat.IntervalSeconds == 0 {
		c.Heartbeat.IntervalSeconds = 5
	}
	if c.Heartbeat.OfflineAfterSeconds == 0 {
		c.Heartbeat.OfflineAfterSeconds = 15
	}
	c.Storage.fillDefaults()
	c.Logging.fillDefaults()
}

func (c *AgentConfig) applyDefaults() {
	if c.Agent.ServerPort == 0 {
		c.Agent.ServerPort = 9000 // juga dipakai sebagai port UDP discovery
	}
	host := strings.ToLower(strings.TrimSpace(c.Agent.ServerHost))
	switch {
	case host == "" || host == "auto":
		// Mode auto-discovery: server ditemukan saat runtime lewat UDP broadcast,
		// kecuali user sudah menuliskan server_url manual (config lama).
		if c.Agent.ServerURL == "" {
			c.Agent.AutoDiscover = true
		}
	default:
		// server_host berisi IP: bangun server_url langsung. User cukup mengetik IP
		// dan port tanpa perlu tahu format URL WebSocket.
		scheme := "ws"
		if c.Agent.UseTLS {
			scheme = "wss"
		}
		c.Agent.ServerURL = fmt.Sprintf("%s://%s:%d/ws/agent",
			scheme, c.Agent.ServerHost, c.Agent.ServerPort)
	}
	if c.Agent.ReconnectSeconds == 0 {
		c.Agent.ReconnectSeconds = 5
	}
	if c.Agent.HeartbeatSeconds == 0 {
		c.Agent.HeartbeatSeconds = 2
	}
	c.Logging.fillDefaults()
}

func (s *StorageSection) fillDefaults() {
	d := defaultStorage()
	if s.DataDir == "" {
		s.DataDir = d.DataDir
	}
	if s.LogsDir == "" {
		s.LogsDir = d.LogsDir
	}
	if s.ScreenshotsDir == "" {
		s.ScreenshotsDir = d.ScreenshotsDir
	}
	if s.UploadsDir == "" {
		s.UploadsDir = d.UploadsDir
	}
	if s.DownloadsDir == "" {
		s.DownloadsDir = d.DownloadsDir
	}
}

func (l *LoggingSection) fillDefaults() {
	if l.Level == "" {
		l.Level = "info"
	}
	if l.MaxSizeMB == 0 {
		l.MaxSizeMB = 50
	}
	if l.MaxBackups == 0 {
		l.MaxBackups = 5
	}
	if l.MaxAgeDays == 0 {
		l.MaxAgeDays = 30
	}
}
