package ws

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"remote_pc/internal/protocol"
)

var errTokenMismatch = errors.New("device token tidak cocok")

// clientIP mengekstrak alamat IP klien dari request, memperhitungkan proxy header.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// mustJSON menserialkan envelope untuk diteruskan apa adanya ke operator.
func mustJSON(env *protocol.Envelope) []byte {
	data, err := json.Marshal(env)
	if err != nil {
		return []byte("{}")
	}
	return data
}
