// Package version menyimpan info build server. Diisi lewat -ldflags saat
// compile (lihat build.ps1 & docker/Dockerfile) supaya bisa dicek dari
// halaman /version untuk memastikan deployment memang memakai kode terbaru
// (bukan cuma menebak-nebak apakah rebuild berhasil mengambil perubahan).
package version

var (
	// AppVersion = "v" + jumlah commit git ("git rev-list --count HEAD") saat
	// build — nomor urut yang naik otomatis tiap ada perubahan, tanpa perlu
	// di-bump manual. Ini yang ditampilkan sbg "Versi Aplikasi" di /version.
	AppVersion = "dev"
	GitCommit  = "unknown"
	BuildTime  = "unknown"
)
