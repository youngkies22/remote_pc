package server

import (
	"net/http"

	"remote_pc/internal/auth"
	"remote_pc/internal/server/api"
	"remote_pc/internal/server/ws"
	"remote_pc/web"
)

// registerRoutes memasang seluruh rute HTTP: aset statis, halaman, REST API,
// dan endpoint WebSocket agent.
func registerRoutes(mux *http.ServeMux, a *api.API, wsh *ws.Handler, mw *auth.Middleware) error {
	sub, err := staticFS()
	if err != nil {
		return err
	}

	// Aset statis (CSS/JS/vendor).
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))

	// Halaman.
	mux.HandleFunc("GET /login", pageHandler("templates/login.html"))
	mux.Handle("GET /{$}", mw.Page(pageHandlerFunc("templates/dashboard.html")))
	mux.Handle("GET /device/{id}", mw.Page(pageHandlerFunc("templates/device.html")))
	mux.Handle("GET /hp", mw.Page(pageHandlerFunc("templates/devices_android.html")))
	mux.Handle("GET /version", mw.Page(pageHandlerFunc("templates/version.html")))

	// REST API publik.
	mux.HandleFunc("POST /api/login", a.Login)

	// Helper untuk memasang rute terproteksi JWT.
	get := func(pattern string, h http.HandlerFunc) { mux.Handle("GET "+pattern, mw.API(h)) }
	post := func(pattern string, h http.HandlerFunc) { mux.Handle("POST "+pattern, mw.API(h)) }

	post("/api/logout", a.Logout)
	get("/api/me", a.Me)
	get("/api/stats", a.Stats)
	get("/api/version", a.Version)
	get("/api/devices", a.ListDevices)
	get("/api/devices/{id}", a.GetDevice)
	mux.Handle("DELETE /api/devices/{id}", mw.API(http.HandlerFunc(a.DeleteDevice)))
	post("/api/devices/{id}/wake", a.Wake)
	post("/api/devices/{id}/power", a.Power)
	post("/api/devices/{id}/message", a.Message)

	// Aksi massal (banyak device sekaligus / per grup subnet).
	post("/api/devices/power-all", a.PowerAll)
	post("/api/devices/message-all", a.MessageAll)

	// Tahap 4/5/11/12 — perintah request/response ke agent.
	get("/api/devices/{id}/sysinfo", a.SysInfo)
	get("/api/devices/{id}/screenshot", a.Screenshot)
	get("/api/devices/{id}/processes", a.Processes)
	post("/api/devices/{id}/processes/kill", a.KillProcess)
	get("/api/devices/{id}/services", a.Services)
	post("/api/devices/{id}/services/control", a.ControlService)

	// Tahap 6 — File Explorer.
	get("/api/devices/{id}/fs/drives", a.FSDrives)
	get("/api/devices/{id}/fs/list", a.FSList)
	get("/api/devices/{id}/fs/download", a.FSDownload)
	post("/api/devices/{id}/fs/upload", a.FSUpload)
	post("/api/devices/{id}/fs/mkdir", a.FSMkdir)
	post("/api/devices/{id}/fs/delete", a.FSDelete)
	post("/api/devices/{id}/fs/rename", a.FSRename)
	post("/api/devices/{id}/fs/copy", a.FSCopy)
	post("/api/devices/{id}/fs/move", a.FSMove)

	// Endpoint WebSocket agent (autentikasi via device token pada handshake register).
	mux.HandleFunc("GET /ws/agent", wsh.ServeAgent)
	// Endpoint WebSocket operator (Tahap 7/8/9/10 — stream layar/terminal + input).
	mux.Handle("GET /ws/operator", mw.API(http.HandlerFunc(wsh.ServeOperator)))

	return nil
}

// pageHandler mengembalikan handler yang menyajikan sebuah file HTML ter-embed.
func pageHandler(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedHTML(w, name)
	}
}

// pageHandlerFunc sama seperti pageHandler tetapi mengembalikan http.Handler
// agar mudah dibungkus middleware.
func pageHandlerFunc(name string) http.Handler {
	return http.HandlerFunc(pageHandler(name))
}

func serveEmbeddedHTML(w http.ResponseWriter, name string) {
	data, err := web.FS.ReadFile(name)
	if err != nil {
		http.Error(w, "halaman tidak ditemukan", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
