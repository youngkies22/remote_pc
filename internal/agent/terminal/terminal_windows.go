// Package terminal menjalankan shell (cmd/PowerShell) dan mengalirkan output-nya
// secara realtime melalui callback. Input dikirim ke stdin shell.
package terminal

import (
	"io"
	"os/exec"
	"sync"
	"syscall"
)

const createNoWindow = 0x08000000

// Session mewakili satu sesi shell yang berjalan.
type Session struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	onOutput func(string)
	once     sync.Once
}

// Start menjalankan shell ("powershell" atau default "cmd") dan mulai membaca
// output-nya. onOutput dipanggil setiap ada keluaran baru.
func Start(shell string, onOutput func(string)) (*Session, error) {
	name := "cmd.exe"
	var args []string
	if shell == "powershell" {
		name = "powershell.exe"
		args = []string{"-NoLogo", "-NoProfile"}
	}

	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s := &Session{cmd: cmd, stdin: stdin, onOutput: onOutput}

	go s.readLoop(pr)
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
		onOutput("\r\n[shell berakhir]\r\n")
	}()

	// Paksa output UTF-8 pada cmd agar karakter non-ASCII lebih rapi.
	if name == "cmd.exe" {
		_, _ = io.WriteString(stdin, "chcp 65001>nul\r\n")
	}
	return s, nil
}

func (s *Session) readLoop(pr io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			s.onOutput(string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

// Write mengirim data ke stdin shell (mis. baris perintah diakhiri \r\n).
func (s *Session) Write(data string) error {
	_, err := io.WriteString(s.stdin, data)
	return err
}

// Close menutup stdin dan menghentikan proses shell.
func (s *Session) Close() {
	s.once.Do(func() {
		_ = s.stdin.Close()
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
	})
}
