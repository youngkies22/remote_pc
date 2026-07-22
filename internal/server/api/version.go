package api

import (
	"net/http"
	"runtime"

	"remote_pc/internal/version"
)

// Version mengembalikan info build server (commit & waktu compile) agar admin
// bisa memastikan deployment yang berjalan memang sudah memakai kode terbaru.
func (a *API) Version(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, map[string]string{
		"app_version": version.AppVersion,
		"git_commit":  version.GitCommit,
		"build_time":  version.BuildTime,
		"go_version":  runtime.Version(),
	})
}
