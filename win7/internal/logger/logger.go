// Package logger membungkus zap untuk menghasilkan log terstruktur yang dipisah
// ke beberapa file: log utama (server.log/client.log), error.log, dan websocket.log.
// Rotasi file ditangani lumberjack.
package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options mengatur pembuatan Loggers.
type Options struct {
	Dir        string // direktori tempat file log ditulis
	MainFile   string // nama file log utama, mis. "server.log" atau "client.log"
	Level      string // debug|info|warn|error
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Console    bool // bila true, log juga ditampilkan ke stdout
}

// Loggers mengelompokkan logger aplikasi dan logger khusus WebSocket.
type Loggers struct {
	App *zap.Logger // log umum aplikasi (juga menulis error+ ke error.log)
	WS  *zap.Logger // log khusus lalu lintas WebSocket
}

// New membangun Loggers sesuai Options. Direktori log dibuat bila belum ada.
func New(opts Options) (*Loggers, error) {
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, err
	}

	level := parseLevel(opts.Level)
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encCfg)

	rotator := func(name string) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(opts.Dir, name),
			MaxSize:    opts.MaxSizeMB,
			MaxBackups: opts.MaxBackups,
			MaxAge:     opts.MaxAgeDays,
			Compress:   false,
		})
	}

	// Core log utama: semua level >= level konfigurasi.
	mainCore := zapcore.NewCore(encoder, rotator(opts.MainFile), level)
	// Core error: hanya level error ke atas, ke error.log.
	errCore := zapcore.NewCore(encoder, rotator("error.log"),
		zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l >= zapcore.ErrorLevel }))

	cores := []zapcore.Core{mainCore, errCore}
	if opts.Console {
		consoleEnc := zapcore.NewConsoleEncoder(devEncoderConfig())
		cores = append(cores, zapcore.NewCore(consoleEnc, zapcore.AddSync(os.Stdout), level))
	}

	app := zap.New(zapcore.NewTee(cores...), zap.AddCaller())

	// Logger WebSocket terpisah ke websocket.log.
	wsCore := zapcore.NewCore(encoder, rotator("websocket.log"), level)
	ws := zap.New(wsCore, zap.AddCaller())

	return &Loggers{App: app, WS: ws}, nil
}

// Sync membersihkan buffer kedua logger. Aman dipanggil saat shutdown.
func (l *Loggers) Sync() {
	if l == nil {
		return
	}
	if l.App != nil {
		_ = l.App.Sync()
	}
	if l.WS != nil {
		_ = l.WS.Sync()
	}
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func devEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	return cfg
}
