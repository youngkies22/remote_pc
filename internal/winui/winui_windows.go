//go:build windows

// Package winui menyediakan interaksi UI Windows minimal untuk aplikasi yang
// dikompilasi sebagai GUI-subsystem (tanpa jendela console): menampilkan dialog
// pesan, memeriksa hak administrator, dan menjalankan ulang diri sendiri dengan
// elevasi (UAC).
package winui

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32     = windows.NewLazySystemDLL("user32.dll")
	procMsgBox = user32.NewProc("MessageBoxW")
)

const (
	mbOK            = 0x00000000
	mbIconError     = 0x00000010
	mbIconInfo      = 0x00000040
	mbSetForeground = 0x00010000
	mbTopMost       = 0x00040000

	swHide = 0
)

// MessageBox menampilkan dialog Windows. Bila iconError true, memakai ikon error;
// selain itu ikon informasi. Aman dipanggil dari Session 0 (tidak tampil, tapi
// tidak menggantung).
func MessageBox(title, text string, iconError bool) {
	flags := uintptr(mbOK | mbSetForeground | mbTopMost)
	if iconError {
		flags |= mbIconError
	} else {
		flags |= mbIconInfo
	}
	textPtr, _ := windows.UTF16PtrFromString(text)
	titlePtr, _ := windows.UTF16PtrFromString(title)
	procMsgBox.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), flags)
}

// IsAdmin melaporkan apakah proses berjalan dengan hak administrator (elevated).
func IsAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// RunSelfElevated menjalankan ulang executable ini dengan hak administrator
// (memicu prompt UAC), meneruskan args. Dipakai agar perintah enable/disable
// bisa dijalankan tanpa harus membuka terminal admin manual. Mengembalikan error
// bila user menolak UAC atau elevasi gagal.
func RunSelfElevated(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verbPtr, _ := windows.UTF16PtrFromString("runas")
	filePtr, _ := windows.UTF16PtrFromString(exe)
	argsPtr, _ := windows.UTF16PtrFromString(strings.Join(args, " "))
	cwdPtr, _ := windows.UTF16PtrFromString(filepath.Dir(exe))
	return windows.ShellExecute(0, verbPtr, filePtr, argsPtr, cwdPtr, swHide)
}
