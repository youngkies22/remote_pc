// Package screen menangkap layar desktop dan meng-encode-nya. Encoder dibuat
// sebagai interface agar implementasi JPEG bisa diganti H264/H265 di masa depan
// tanpa mengubah pemanggil.
package screen

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/kbinani/screenshot"

	"remote_pc/internal/protocol"
)

// Encoder mengubah gambar mentah menjadi byte terkompresi.
type Encoder interface {
	Encode(img image.Image) ([]byte, error)
	Format() string
}

// JPEGEncoder meng-encode frame sebagai JPEG dengan kualitas tertentu.
type JPEGEncoder struct {
	Quality int
}

// Encode mengimplementasikan Encoder.
func (e JPEGEncoder) Encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	q := e.Quality
	if q <= 0 || q > 100 {
		q = 60
	}
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Format mengimplementasikan Encoder.
func (e JPEGEncoder) Format() string { return "jpeg" }

// DefaultEncoder adalah encoder yang dipakai bila tidak ditentukan.
var DefaultEncoder Encoder = JPEGEncoder{Quality: 60}

// Capture menangkap layar utama memakai DefaultEncoder.
func Capture() (protocol.ScreenShot, error) {
	return CaptureWith(DefaultEncoder)
}

// CaptureWith menangkap layar utama menggunakan encoder yang diberikan.
func CaptureWith(enc Encoder) (protocol.ScreenShot, error) {
	if screenshot.NumActiveDisplays() <= 0 {
		return protocol.ScreenShot{}, fmt.Errorf("tidak ada display aktif (agent mungkin di session non-interaktif)")
	}
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return protocol.ScreenShot{}, err
	}
	data, err := enc.Encode(img)
	if err != nil {
		return protocol.ScreenShot{}, err
	}
	return protocol.ScreenShot{
		Format: enc.Format(),
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Data:   base64.StdEncoding.EncodeToString(data),
	}, nil
}
