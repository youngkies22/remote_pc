package input

import "strings"

// vkFor memetakan nama tombol (dari browser) ke Virtual-Key Code Windows.
func vkFor(key string) (uint16, bool) {
	switch strings.ToLower(key) {
	case "enter", "return":
		return 0x0D, true
	case "backspace":
		return 0x08, true
	case "tab":
		return 0x09, true
	case "escape", "esc":
		return 0x1B, true
	case "space", " ":
		return 0x20, true
	case "delete", "del":
		return 0x2E, true
	case "insert":
		return 0x2D, true
	case "home":
		return 0x24, true
	case "end":
		return 0x23, true
	case "pageup":
		return 0x21, true
	case "pagedown":
		return 0x22, true
	case "arrowleft", "left":
		return 0x25, true
	case "arrowup", "up":
		return 0x26, true
	case "arrowright", "right":
		return 0x27, true
	case "arrowdown", "down":
		return 0x28, true
	}
	// Fungsi F1..F12
	if len(key) >= 2 && (key[0] == 'f' || key[0] == 'F') {
		if n := atoiSmall(key[1:]); n >= 1 && n <= 12 {
			return uint16(0x70 + n - 1), true
		}
	}
	// Huruf tunggal a-z / A-Z
	if len(key) == 1 {
		c := key[0]
		if c >= 'a' && c <= 'z' {
			return uint16(0x41 + (c - 'a')), true
		}
		if c >= 'A' && c <= 'Z' {
			return uint16(0x41 + (c - 'A')), true
		}
		if c >= '0' && c <= '9' {
			return uint16(0x30 + (c - '0')), true
		}
	}
	return 0, false
}

// modVK memetakan nama modifier ke Virtual-Key Code.
func modVK(mod string) (uint16, bool) {
	switch strings.ToLower(mod) {
	case "ctrl", "control":
		return 0x11, true
	case "alt":
		return 0x12, true
	case "shift":
		return 0x10, true
	case "win", "meta", "super":
		return 0x5B, true
	}
	return 0, false
}

func atoiSmall(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}
