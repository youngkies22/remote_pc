// Package fsops menjalankan operasi filesystem yang diminta server: menjelajah,
// unduh, unggah, rename, hapus, salin, pindah, dan buat folder.
package fsops

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"remote_pc/internal/protocol"
)

// maxFileSize membatasi ukuran unduh/unggah agar muat dalam satu pesan WebSocket.
const maxFileSize = 40 * 1024 * 1024 // 40 MB

// Drives mengembalikan daftar drive yang tersedia (mis. C:\, D:\).
func Drives() protocol.FSDrivesResponse {
	var drives []string
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + ":\\"
		if _, err := os.Stat(root); err == nil {
			drives = append(drives, root)
		}
	}
	return protocol.FSDrivesResponse{Drives: drives}
}

// List mengembalikan isi folder. Bila path kosong, mengembalikan daftar drive.
func List(path string) (protocol.FSListResponse, error) {
	if path == "" {
		res := protocol.FSListResponse{Path: ""}
		for _, d := range Drives().Drives {
			res.Entries = append(res.Entries, protocol.FSEntry{Name: d, IsDir: true})
		}
		return res, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return protocol.FSListResponse{}, err
	}
	res := protocol.FSListResponse{Path: path}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		res.Entries = append(res.Entries, protocol.FSEntry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}
	sort.Slice(res.Entries, func(i, j int) bool {
		if res.Entries[i].IsDir != res.Entries[j].IsDir {
			return res.Entries[i].IsDir
		}
		return res.Entries[i].Name < res.Entries[j].Name
	})
	return res, nil
}

// Read membaca isi file dan mengembalikannya dalam base64.
func Read(path string) (protocol.FSReadResponse, error) {
	info, err := os.Stat(path)
	if err != nil {
		return protocol.FSReadResponse{}, err
	}
	if info.IsDir() {
		return protocol.FSReadResponse{}, fmt.Errorf("%q adalah folder, bukan file", path)
	}
	if info.Size() > maxFileSize {
		return protocol.FSReadResponse{}, fmt.Errorf("file terlalu besar (maks %d MB)", maxFileSize/(1024*1024))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return protocol.FSReadResponse{}, err
	}
	return protocol.FSReadResponse{
		Name: filepath.Base(path),
		Size: info.Size(),
		Data: base64.StdEncoding.EncodeToString(data),
	}, nil
}

// Write menulis (unggah) file dari data base64 ke path.
func Write(path, b64 string) error {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("data base64 tidak valid: %w", err)
	}
	if len(data) > maxFileSize {
		return fmt.Errorf("file terlalu besar (maks %d MB)", maxFileSize/(1024*1024))
	}
	return os.WriteFile(path, data, 0o644)
}

// Mkdir membuat folder (termasuk parent bila perlu).
func Mkdir(path string) error { return os.MkdirAll(path, 0o755) }

// Delete menghapus file atau folder (rekursif).
func Delete(path string) error { return os.RemoveAll(path) }

// Rename mengganti nama / memindahkan dalam volume yang sama.
func Rename(src, dst string) error { return os.Rename(src, dst) }

// Move memindahkan file/folder; bila beda volume, salin lalu hapus sumber.
func Move(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := Copy(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// Copy menyalin file atau folder (rekursif) dari src ke dst.
func Copy(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else if err := copyFile(s, d); err != nil {
			return err
		}
	}
	return nil
}
