package api

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"remote_pc/internal/model"
	"remote_pc/internal/protocol"
	"remote_pc/internal/wol"
)

// deviceDTO adalah representasi device yang aman dikirim ke frontend
// (tanpa membocorkan device token).
type deviceDTO struct {
	ID             string        `json:"id"`
	Hostname       string        `json:"hostname"`
	Username       string        `json:"username"`
	IP             string        `json:"ip"`
	MAC            string        `json:"mac"`
	OS             string        `json:"os"`
	WindowsVersion string        `json:"windows_version"`
	Arch           string        `json:"arch"`
	Metrics        model.Metrics `json:"metrics"`
	Status         string        `json:"status"`
	FirstSeen      time.Time     `json:"first_seen"`
	LastSeen       time.Time     `json:"last_seen"`
}

// liveStatus menentukan status tampilan berdasarkan koneksi hub yang aktual.
func (a *API) liveStatus(d model.Device) model.DeviceStatus {
	if a.hub.IsOnline(d.ID) {
		return model.StatusOnline
	}
	return model.StatusOffline
}

func (a *API) toDTO(d model.Device) deviceDTO {
	return deviceDTO{
		ID:             d.ID,
		Hostname:       d.Hostname,
		Username:       d.Username,
		IP:             d.IP,
		MAC:            d.MAC,
		OS:             d.OS,
		WindowsVersion: d.WindowsVersion,
		Arch:           d.Arch,
		Metrics:        d.Metrics,
		Status:         string(a.liveStatus(d)),
		FirstSeen:      d.FirstSeen,
		LastSeen:       d.LastSeen,
	}
}

// ListDevices mengembalikan seluruh device, diurutkan online lebih dulu lalu
// berdasarkan hostname.
func (a *API) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices := a.store.Devices.All()
	out := make([]deviceDTO, 0, len(devices))
	for _, d := range devices {
		out = append(out, a.toDTO(d))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			return out[i].Status == string(model.StatusOnline)
		}
		return out[i].Hostname < out[j].Hostname
	})
	a.writeJSON(w, http.StatusOK, out)
}

// GetDevice mengembalikan detail satu device berdasarkan ID.
func (a *API) GetDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dev, ok := a.store.Devices.Get(id)
	if !ok {
		a.writeError(w, http.StatusNotFound, "device tidak ditemukan")
		return
	}
	a.writeJSON(w, http.StatusOK, a.toDTO(dev))
}

// DeleteDevice menghapus device dari storage.
func (a *API) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.store.Devices.Get(id); !ok {
		a.writeError(w, http.StatusNotFound, "device tidak ditemukan")
		return
	}
	if err := a.store.Devices.Delete(id); err != nil {
		a.writeError(w, http.StatusInternalServerError, "gagal menghapus device")
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Wake mengirim Wake-on-LAN magic packet ke device berdasarkan MAC address
// tersimpan. Tidak butuh device online — inilah gunanya WOL (menyalakan PC yang
// sedang mati total). Hanya berhasil bila PC target sudah mengaktifkan
// Wake-on-LAN di BIOS/UEFI & pengaturan NIC-nya (harus disetel manual di PC itu,
// tidak bisa dari jarak jauh).
func (a *API) Wake(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dev, ok := a.store.Devices.Get(id)
	if !ok {
		a.writeError(w, http.StatusNotFound, "device tidak ditemukan")
		return
	}
	if dev.MAC == "" {
		a.writeError(w, http.StatusBadRequest, "device ini belum memiliki MAC address tersimpan")
		return
	}
	if err := wol.Send(dev.MAC); err != nil {
		a.writeError(w, http.StatusInternalServerError, "gagal mengirim wake-on-lan: "+err.Error())
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "mac": dev.MAC})
}

// Power meminta agent mematikan atau merestart komputer (action:
// "shutdown"|"restart"). Fire-and-forget: setelah perintah dikirim, agent akan
// segera mati sehingga tidak ada balasan yang ditunggu.
func (a *API) Power(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}
	if !a.decodeBody(w, r, &req) {
		return
	}
	var t protocol.MessageType
	switch req.Action {
	case "shutdown":
		t = protocol.TypePowerShutdown
	case "restart":
		t = protocol.TypePowerRestart
	default:
		a.writeError(w, http.StatusBadRequest, "action harus 'shutdown' atau 'restart'")
		return
	}
	deviceID := r.PathValue("id")
	if !a.hub.Notify(deviceID, t, nil) {
		a.writeError(w, http.StatusBadGateway, "agent tidak terhubung")
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "action": req.Action})
}

// Message meminta agent menampilkan dialog pesan di layar komputer siswa.
// Fire-and-forget: server tidak menunggu balasan agent.
func (a *API) Message(w http.ResponseWriter, r *http.Request) {
	var req protocol.MessageRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		a.writeError(w, http.StatusBadRequest, "isi pesan tidak boleh kosong")
		return
	}
	deviceID := r.PathValue("id")
	if !a.hub.Notify(deviceID, protocol.TypeMessage, req) {
		a.writeError(w, http.StatusBadGateway, "agent tidak terhubung")
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// targetIDs menentukan device tujuan aksi massal. Bila ids kosong, seluruh device
// online menjadi target; bila diisi, hanya ids tersebut (mis. satu grup subnet).
func (a *API) targetIDs(ids []string) []string {
	if len(ids) > 0 {
		return ids
	}
	out := make([]string, 0)
	for _, d := range a.store.Devices.All() {
		if a.hub.IsOnline(d.ID) {
			out = append(out, d.ID)
		}
	}
	return out
}

// PowerAll mengirim perintah shutdown/restart ke banyak device sekaligus. Body:
// {action, ids?}. Bila ids kosong, berlaku untuk seluruh device online.
func (a *API) PowerAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string   `json:"action"`
		IDs    []string `json:"ids"`
	}
	if !a.decodeBody(w, r, &req) {
		return
	}
	var t protocol.MessageType
	switch req.Action {
	case "shutdown":
		t = protocol.TypePowerShutdown
	case "restart":
		t = protocol.TypePowerRestart
	default:
		a.writeError(w, http.StatusBadRequest, "action harus 'shutdown' atau 'restart'")
		return
	}
	sent := 0
	for _, id := range a.targetIDs(req.IDs) {
		if a.hub.Notify(id, t, nil) {
			sent++
		}
	}
	a.writeJSON(w, http.StatusOK, map[string]interface{}{"status": "sent", "count": sent})
}

// MessageAll mengirim pesan ke banyak device sekaligus. Body: {title, text, ids?}.
// Bila ids kosong, berlaku untuk seluruh device online.
func (a *API) MessageAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string   `json:"title"`
		Text  string   `json:"text"`
		IDs   []string `json:"ids"`
	}
	if !a.decodeBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		a.writeError(w, http.StatusBadRequest, "isi pesan tidak boleh kosong")
		return
	}
	payload := protocol.MessageRequest{Title: req.Title, Text: req.Text}
	sent := 0
	for _, id := range a.targetIDs(req.IDs) {
		if a.hub.Notify(id, protocol.TypeMessage, payload) {
			sent++
		}
	}
	a.writeJSON(w, http.StatusOK, map[string]interface{}{"status": "sent", "count": sent})
}

// Stats mengembalikan ringkasan untuk kartu dashboard.
func (a *API) Stats(w http.ResponseWriter, r *http.Request) {
	devices := a.store.Devices.All()
	online := 0
	for _, d := range devices {
		if a.hub.IsOnline(d.ID) {
			online++
		}
	}
	a.writeJSON(w, http.StatusOK, map[string]int{
		"total":   len(devices),
		"online":  online,
		"offline": len(devices) - online,
	})
}
