// Package autostart mendaftarkan sebuah exe agar berjalan otomatis di Windows,
// memakai Task Scheduler (schtasks.exe). Dipakai oleh agent maupun server agar
// keduanya cukup dijalankan sekali ("enable") lalu otomatis aktif setiap PC
// dinyalakan, tanpa perlu dibuka manual.
//
// Dua jenis trigger tersedia lewat Options.Trigger:
//
//   - TriggerLogon: jalan saat ada user login, di sesi desktop interaktif.
//     Dipakai agent, karena screenshot/live-screen/remote-input butuh sesi
//     desktop — Windows Service (Session 0) tidak bisa melakukan itu.
//   - TriggerBoot: jalan saat Windows boot, sebagai akun SYSTEM, sebelum ada
//     user login. Dipakai server, karena server tidak butuh sesi desktop dan
//     idealnya sudah aktif sebelum siapa pun login.
package autostart

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

// createNoWindow mencegah jendela console muncul saat menjalankan schtasks dari
// exe GUI-subsystem.
const createNoWindow = 0x08000000

// hidden mengembalikan atribut proses agar schtasks berjalan tanpa jendela.
func hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// Trigger menentukan kapan task dijalankan.
type Trigger int

const (
	// TriggerLogon menjalankan task saat user manapun login (sesi desktop).
	TriggerLogon Trigger = iota
	// TriggerBoot menjalankan task saat Windows boot, sebagai SYSTEM.
	TriggerBoot
)

// Options mengatur detail task yang didaftarkan.
type Options struct {
	Trigger     Trigger
	Description string
}

// Install membuat/mengganti scheduled task bernama taskName yang menjalankan
// exePath dengan argumen -config configPath sesuai Options.Trigger.
func Install(taskName, exePath, configPath string, opts Options) error {
	xml := buildTaskXML(exePath, configPath, opts)

	tmp, err := os.CreateTemp("", "remotepc-task-*.xml")
	if err != nil {
		return fmt.Errorf("autostart: buat file sementara: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	// schtasks /XML mensyaratkan file dalam encoding UTF-16LE dengan BOM.
	if _, err := tmp.Write(toUTF16LE(xml)); err != nil {
		tmp.Close()
		return fmt.Errorf("autostart: tulis definisi task: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("autostart: tutup definisi task: %w", err)
	}

	// /F menimpa task lama bila sudah ada, sehingga enable bisa dijalankan ulang
	// untuk memperbarui path exe/config.
	return runSchtasks("/Create", "/TN", taskName, "/XML", tmpPath, "/F")
}

// toUTF16LE mengubah string menjadi byte UTF-16 little-endian dengan BOM, format
// yang diwajibkan schtasks untuk input /XML.
func toUTF16LE(s string) []byte {
	units := utf16.Encode([]rune(s))
	buf := bytes.NewBuffer(make([]byte, 0, len(units)*2+2))
	buf.Write([]byte{0xFF, 0xFE}) // BOM little-endian
	for _, u := range units {
		_ = binary.Write(buf, binary.LittleEndian, u)
	}
	return buf.Bytes()
}

// Uninstall menghapus scheduled task. Bila task memang belum/tidak terpasang,
// ini dianggap berhasil (tidak ada apa-apa yang perlu dihapus) alih-alih error.
func Uninstall(taskName string) error {
	if !Exists(taskName) {
		return nil
	}
	return runSchtasks("/Delete", "/TN", taskName, "/F")
}

// Run menjalankan task sekarang juga (tanpa menunggu trigger berikutnya),
// berguna agar aplikasi langsung aktif setelah enable.
func Run(taskName string) error {
	return runSchtasks("/Run", "/TN", taskName)
}

// Exists melaporkan apakah task sudah terpasang.
func Exists(taskName string) bool {
	cmd := exec.Command("schtasks", "/Query", "/TN", taskName)
	cmd.SysProcAttr = hidden()
	return cmd.Run() == nil
}

func runSchtasks(args ...string) error {
	cmd := exec.Command("schtasks", args...)
	cmd.SysProcAttr = hidden()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks %s gagal: %v: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// buildTaskXML menghasilkan definisi Task Scheduler 1.2. Path di-escape agar aman
// terhadap karakter XML (&, <, >).
func buildTaskXML(exePath, configPath string, opts Options) string {
	command := escapeXML(exePath)
	arguments := escapeXML(fmt.Sprintf(`-config "%s"`, configPath))
	workDir := escapeXML(filepath.Dir(exePath))
	desc := escapeXML(opts.Description)

	triggerXML := `<LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>`
	principalXML := `<GroupId>S-1-5-32-545</GroupId>
      <RunLevel>HighestAvailable</RunLevel>`

	if opts.Trigger == TriggerBoot {
		triggerXML = `<BootTrigger>
      <Enabled>true</Enabled>
    </BootTrigger>`
		// S-1-5-18 = akun SYSTEM bawaan Windows: tidak butuh sesi desktop/login,
		// dan sudah punya hak penuh sehingga cocok untuk server yang harus aktif
		// sebelum siapa pun login.
		principalXML = `<UserId>S-1-5-18</UserId>
      <RunLevel>HighestAvailable</RunLevel>`
	}

	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>` + desc + `</Description>
  </RegistrationInfo>
  <Triggers>
    ` + triggerXML + `
  </Triggers>
  <Principals>
    <Principal id="Author">
      ` + principalXML + `
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>false</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>999</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>` + command + `</Command>
      <Arguments>` + arguments + `</Arguments>
      <WorkingDirectory>` + workDir + `</WorkingDirectory>
    </Exec>
  </Actions>
</Task>`
}

func escapeXML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(s)
}
