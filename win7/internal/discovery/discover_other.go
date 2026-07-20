//go:build !windows

package discovery

import (
	"fmt"
	"time"
)

// Discover tidak dipakai di luar Windows (agent hanya berjalan di Windows). Stub
// ini ada agar paket tetap bisa dikompilasi bila server di-build untuk Linux/Docker.
func Discover(port int, timeout time.Duration) (host string, wsPort int, useTLS bool, err error) {
	return "", 0, false, fmt.Errorf("auto-discovery hanya didukung di Windows")
}
