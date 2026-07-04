package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"remote_pc/internal/protocol"
)

const commandTimeout = 30 * time.Second

// proxy meneruskan sebuah command ke agent, menunggu balasan, lalu menuliskan
// payload balasan apa adanya sebagai JSON ke browser.
func (a *API) proxy(w http.ResponseWriter, r *http.Request, t protocol.MessageType, payload interface{}) {
	deviceID := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), commandTimeout)
	defer cancel()

	reply, err := a.hub.Request(ctx, deviceID, t, payload)
	if err != nil {
		a.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if len(reply.Payload) == 0 {
		_, _ = w.Write([]byte("{}"))
		return
	}
	_, _ = w.Write(reply.Payload)
}

// decodeBody membaca body JSON ke out; menulis 400 dan mengembalikan false bila gagal.
func (a *API) decodeBody(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		a.writeError(w, http.StatusBadRequest, "body tidak valid")
		return false
	}
	return true
}

// SysInfo (Tahap 4) meminta informasi sistem lengkap dari agent.
func (a *API) SysInfo(w http.ResponseWriter, r *http.Request) {
	a.proxy(w, r, protocol.TypeSysInfo, nil)
}

// Screenshot (Tahap 5) meminta tangkapan layar, menyimpannya ke folder
// screenshots/, lalu mengembalikan gambar (base64) ke browser.
func (a *API) Screenshot(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), commandTimeout)
	defer cancel()

	reply, err := a.hub.Request(ctx, deviceID, protocol.TypeScreenshot, nil)
	if err != nil {
		a.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	var shot protocol.ScreenShot
	if err := reply.Decode(&shot); err == nil {
		a.saveScreenshot(deviceID, shot)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(reply.Payload)
}

// saveScreenshot menyimpan JPEG ke folder screenshots/ (best-effort).
func (a *API) saveScreenshot(deviceID string, shot protocol.ScreenShot) {
	raw, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil || len(raw) == 0 {
		return
	}
	if err := os.MkdirAll(a.screenshotsDir, 0o755); err != nil {
		return
	}
	name := deviceID + "_" + time.Now().Format("20060102_150405") + ".jpg"
	if err := os.WriteFile(filepath.Join(a.screenshotsDir, name), raw, 0o644); err != nil {
		a.log.Warn("gagal menyimpan screenshot")
	}
}

// Processes (Tahap 11) meminta daftar proses.
func (a *API) Processes(w http.ResponseWriter, r *http.Request) {
	a.proxy(w, r, protocol.TypeProcList, nil)
}

// KillProcess (Tahap 11) meminta agent mematikan sebuah proses.
func (a *API) KillProcess(w http.ResponseWriter, r *http.Request) {
	var req protocol.ProcKillRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeProcKill, req)
}

// Services (Tahap 12) meminta daftar Windows Service.
func (a *API) Services(w http.ResponseWriter, r *http.Request) {
	a.proxy(w, r, protocol.TypeSvcList, nil)
}

// ControlService (Tahap 12) meminta start/stop/restart sebuah service.
func (a *API) ControlService(w http.ResponseWriter, r *http.Request) {
	var req protocol.SvcControlRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeSvcControl, req)
}
