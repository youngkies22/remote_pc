// Package input menyuntikkan event mouse dan keyboard ke desktop menggunakan
// user32!SendInput (syscall murni, tanpa cgo). Berfungsi bila agent berjalan di
// dalam sesi desktop interaktif (bukan Session 0).
package input

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"remote_pc/internal/protocol"
)

var (
	user32       = windows.NewLazySystemDLL("user32.dll")
	procSendInput = user32.NewProc("SendInput")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseeventfMove     = 0x0001
	mouseeventfLeftDown = 0x0002
	mouseeventfLeftUp   = 0x0004
	mouseeventfRightDown = 0x0008
	mouseeventfRightUp  = 0x0010
	mouseeventfWheel    = 0x0800
	mouseeventfAbsolute = 0x8000
	wheelDelta          = 120

	keyeventfKeyUp   = 0x0002
	keyeventfUnicode = 0x0004
)

// mouseInputT dan keybdInputT adalah anggota union INPUT. mouseInputT adalah yang
// terbesar sehingga dipakai sebagai penyimpanan; keyboard ditulis via overlay.
type mouseInputT struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type keybdInputT struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// inputT setara struct INPUT Win32. Padding antar-field dihitung otomatis oleh
// Go sehingga ukurannya benar untuk amd64 (40) maupun 386 (28).
type inputT struct {
	inputType uint32
	mi        mouseInputT
}

func send(in *inputT) {
	procSendInput.Call(1, uintptr(unsafe.Pointer(in)), unsafe.Sizeof(*in))
}

func sendMouse(flags uint32, dx, dy int32, data uint32) {
	in := inputT{inputType: inputMouse, mi: mouseInputT{
		dx: dx, dy: dy, mouseData: data, dwFlags: flags,
	}}
	send(&in)
}

func sendKey(vk uint16, scan uint16, flags uint32) {
	in := inputT{inputType: inputKeyboard}
	ki := (*keybdInputT)(unsafe.Pointer(&in.mi))
	ki.wVk = vk
	ki.wScan = scan
	ki.dwFlags = flags
	send(&in)
}

// Mouse mengeksekusi satu event mouse. Koordinat X/Y relatif 0..1 terhadap layar.
func Mouse(ev protocol.MouseEvent) {
	absX := int32(ev.X * 65535)
	absY := int32(ev.Y * 65535)
	move := func() { sendMouse(mouseeventfMove|mouseeventfAbsolute, absX, absY, 0) }

	switch ev.Action {
	case "move":
		move()
	case "down":
		move()
		sendMouse(mouseeventfLeftDown, absX, absY, 0)
	case "up":
		sendMouse(mouseeventfLeftUp, absX, absY, 0)
	case "click":
		move()
		sendMouse(mouseeventfLeftDown, absX, absY, 0)
		sendMouse(mouseeventfLeftUp, absX, absY, 0)
	case "rclick":
		move()
		sendMouse(mouseeventfRightDown, absX, absY, 0)
		sendMouse(mouseeventfRightUp, absX, absY, 0)
	case "dblclick":
		move()
		sendMouse(mouseeventfLeftDown, absX, absY, 0)
		sendMouse(mouseeventfLeftUp, absX, absY, 0)
		sendMouse(mouseeventfLeftDown, absX, absY, 0)
		sendMouse(mouseeventfLeftUp, absX, absY, 0)
	case "scroll":
		sendMouse(mouseeventfWheel, 0, 0, uint32(int32(ev.Scroll*wheelDelta)))
	}
}

// Key mengeksekusi event keyboard: teks Unicode biasa atau tombol khusus + modifier.
func Key(ev protocol.KeyEvent) {
	if ev.Text != "" {
		for _, r := range ev.Text {
			if r > 0xFFFF {
				continue // lewati karakter di luar BMP untuk saat ini
			}
			sendKey(0, uint16(r), keyeventfUnicode)
			sendKey(0, uint16(r), keyeventfUnicode|keyeventfKeyUp)
		}
		return
	}
	if ev.Key == "" {
		return
	}
	vk, ok := vkFor(ev.Key)
	if !ok {
		return
	}
	var mods []uint16
	for _, m := range ev.Modifiers {
		if mvk, ok := modVK(m); ok {
			mods = append(mods, mvk)
			sendKey(mvk, 0, 0) // tekan modifier
		}
	}
	sendKey(vk, 0, 0)
	sendKey(vk, 0, keyeventfKeyUp)
	for i := len(mods) - 1; i >= 0; i-- {
		sendKey(mods[i], 0, keyeventfKeyUp) // lepas modifier
	}
}
