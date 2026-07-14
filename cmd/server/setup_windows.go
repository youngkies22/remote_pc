//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"remote_pc/internal/autostart"
	"remote_pc/internal/config"
	"remote_pc/internal/winui"
)

// createNoWindow mencegah jendela console muncul saat menjalankan program
// console (netsh) dari exe GUI-subsystem ini.
const createNoWindow = 0x08000000

// taskName adalah nama scheduled task auto-start server.
const taskName = "RemotePCServer"

// firewallRuleName adalah nama rule Windows Firewall yang dibuat/dihapus enable/disable.
const firewallRuleName = "Remote PC Server"

// enableAutostart mendaftarkan server agar berjalan otomatis saat Windows boot
// (sebagai SYSTEM, sebelum ada yang login), membuka port di Windows Firewall,
// lalu menjalankannya sekarang juga. Meminta elevasi (UAC) sendiri bila belum
// admin, sehingga tidak perlu membuka terminal admin manual.
func enableAutostart(configPath string) {
	if !winui.IsAdmin() {
		if err := winui.RunSelfElevated("enable"); err != nil {
			winui.MessageBox(appName, "Persetujuan Administrator dibutuhkan untuk mengaktifkan "+
				"auto-start, tetapi dibatalkan atau gagal:\n"+err.Error(), true)
		}
		return
	}

	created, err := ensureConfig(configPath)
	if err != nil {
		winui.MessageBox(appName, "Gagal menyiapkan konfigurasi:\n"+err.Error(), true)
		return
	}
	if created {
		winui.MessageBox(appName,
			"Konfigurasi baru dibuat di:\n"+configPath+
				"\n\nNilai default sudah aman. Jalankan installer server lagi untuk mengaktifkan auto-start.", false)
		return
	}

	cfg, err := config.LoadServerConfig(configPath)
	if err != nil {
		winui.MessageBox(appName, "Gagal membaca konfigurasi:\n"+err.Error(), true)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		winui.MessageBox(appName, "Gagal menemukan lokasi program:\n"+err.Error(), true)
		return
	}

	opts := autostart.Options{
		Trigger:     autostart.TriggerBoot,
		Description: "Server Remote PC - berjalan otomatis saat Windows menyala.",
	}
	if err := autostart.Install(taskName, exe, configPath, opts); err != nil {
		winui.MessageBox(appName, "Gagal mengaktifkan auto-start:\n"+err.Error(), true)
		return
	}

	fw := fmt.Sprintf("Port %d dibuka di Windows Firewall (semua profil).", cfg.Server.Port)
	if err := addFirewallRule(cfg.Server.Port); err != nil {
		fw = fmt.Sprintf("Catatan: gagal membuka port %d di firewall otomatis (%v).\n"+
			"Buka manual bila perlu.", cfg.Server.Port, err)
	}
	_ = autostart.Run(taskName) // langsung jalankan (best-effort)

	winui.MessageBox(appName,
		"Berhasil! Server kini berjalan tersembunyi di latar belakang dan akan "+
			"otomatis menyala setiap Windows boot.\n\n"+fw, false)
}

// disableAutostart menghapus task auto-start dan rule firewall yang dibuat enable.
func disableAutostart() {
	if !winui.IsAdmin() {
		if err := winui.RunSelfElevated("disable"); err != nil {
			winui.MessageBox(appName, "Persetujuan Administrator dibutuhkan:\n"+err.Error(), true)
		}
		return
	}
	if err := autostart.Uninstall(taskName); err != nil {
		winui.MessageBox(appName, "Gagal menonaktifkan auto-start:\n"+err.Error(), true)
		return
	}
	_ = removeFirewallRule()
	winui.MessageBox(appName, "Auto-start server dinonaktifkan & port firewall ditutup.", false)
}

// addFirewallRule membuka port inbound di semua profil Windows Firewall: TCP
// untuk WebSocket/HTTP dan UDP untuk auto-discovery (keduanya dengan nama rule
// sama agar sekali hapus membersihkan keduanya). Butuh hak Administrator; wajar
// gagal bila dijalankan tanpa elevasi.
func addFirewallRule(port int) error {
	for _, proto := range []string{"TCP", "UDP"} {
		cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
			"name="+firewallRuleName,
			"dir=in",
			"action=allow",
			"protocol="+proto,
			fmt.Sprintf("localport=%d", port),
		)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("netsh gagal (%s): %v: %s", proto, err, string(out))
		}
	}
	return nil
}

// removeFirewallRule menghapus rule yang dibuat addFirewallRule.
func removeFirewallRule() error {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule",
		"name="+firewallRuleName,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("netsh gagal: %v: %s", err, string(out))
	}
	return nil
}
