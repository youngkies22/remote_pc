// Package version menyimpan info build server. Diisi lewat -ldflags saat
// compile (lihat build.ps1 & docker/Dockerfile) supaya bisa dicek dari
// halaman /version untuk memastikan deployment memang memakai kode terbaru
// (bukan cuma menebak-nebak apakah rebuild berhasil mengambil perubahan).
package version

var (
	GitCommit = "unknown"
	BuildTime = "unknown"
)
