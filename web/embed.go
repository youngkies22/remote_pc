// Package web menyematkan (embed) seluruh aset frontend ke dalam biner server
// sehingga server dapat dijalankan sebagai satu file tanpa folder web terpisah.
package web

import "embed"

// FS berisi aset statis (static/) dan halaman HTML (templates/).
//
//go:embed static templates
var FS embed.FS
