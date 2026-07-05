//go:build !windows

// Package winui menyediakan interaksi UI Windows minimal untuk aplikasi yang
// dikompilasi sebagai GUI-subsystem. Di platform selain Windows (mis. server
// yang berjalan di Linux/Docker) tidak ada dialog GUI atau UAC, jadi fungsi di
// sini hanya mencetak ke stderr / mengembalikan nilai netral agar cmd/server
// tetap bisa dikompilasi dan berjalan tanpa cabang kode khusus platform.
package winui

import "fmt"

// MessageBox mencetak pesan ke stderr (tidak ada dialog GUI di platform ini).
func MessageBox(title, text string, iconError bool) {
	fmt.Printf("[%s] %s\n", title, text)
}

// IsAdmin selalu true di luar Windows — tidak ada konsep UAC/elevasi terpisah;
// hak akses proses diatur lewat user Linux yang menjalankannya (mis. di Docker).
func IsAdmin() bool { return true }

// RunSelfElevated tidak berlaku di luar Windows (tidak ada UAC).
func RunSelfElevated(args ...string) error {
	return fmt.Errorf("elevasi (UAC) tidak berlaku di platform ini")
}
