package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"remote_pc/internal/protocol"
)

const downloadTimeout = 120 * time.Second

// FSDrives (Tahap 6) meminta daftar drive dari agent.
func (a *API) FSDrives(w http.ResponseWriter, r *http.Request) {
	a.proxy(w, r, protocol.TypeFSDrives, nil)
}

// FSList (Tahap 6) meminta daftar isi sebuah folder (query ?path=).
func (a *API) FSList(w http.ResponseWriter, r *http.Request) {
	a.proxy(w, r, protocol.TypeFSList, protocol.FSPathRequest{Path: r.URL.Query().Get("path")})
}

// FSDownload (Tahap 6) mengunduh sebuah file dari agent dan mengalirkannya ke browser.
func (a *API) FSDownload(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	path := r.URL.Query().Get("path")
	ctx, cancel := context.WithTimeout(r.Context(), downloadTimeout)
	defer cancel()

	reply, err := a.hub.Request(ctx, deviceID, protocol.TypeFSRead, protocol.FSPathRequest{Path: path})
	if err != nil {
		a.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	var res protocol.FSReadResponse
	if err := reply.Decode(&res); err != nil {
		a.writeError(w, http.StatusInternalServerError, "balasan tidak valid")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(res.Data)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "data file rusak")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+res.Name+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	_, _ = w.Write(raw)
}

// FSUpload (Tahap 6) mengunggah file ke agent. Body JSON: {path, data(base64)}.
func (a *API) FSUpload(w http.ResponseWriter, r *http.Request) {
	var req protocol.FSWriteRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	deviceID := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), downloadTimeout)
	defer cancel()
	reply, err := a.hub.Request(ctx, deviceID, protocol.TypeFSWrite, req)
	if err != nil {
		a.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if len(reply.Payload) == 0 {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
		return
	}
	_, _ = w.Write(reply.Payload)
}

// FSMkdir (Tahap 6) membuat folder baru.
func (a *API) FSMkdir(w http.ResponseWriter, r *http.Request) {
	var req protocol.FSPathRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeFSMkdir, req)
}

// FSDelete (Tahap 6) menghapus file/folder.
func (a *API) FSDelete(w http.ResponseWriter, r *http.Request) {
	var req protocol.FSPathRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeFSDelete, req)
}

// FSRename (Tahap 6) mengganti nama file/folder.
func (a *API) FSRename(w http.ResponseWriter, r *http.Request) {
	var req protocol.FSTwoPathRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeFSRename, req)
}

// FSCopy (Tahap 6) menyalin file/folder.
func (a *API) FSCopy(w http.ResponseWriter, r *http.Request) {
	var req protocol.FSTwoPathRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeFSCopy, req)
}

// FSMove (Tahap 6) memindahkan file/folder.
func (a *API) FSMove(w http.ResponseWriter, r *http.Request) {
	var req protocol.FSTwoPathRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	a.proxy(w, r, protocol.TypeFSMove, req)
}
