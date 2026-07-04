package protocol

// File ini mendefinisikan payload untuk tiap aksi command. Struktur dipakai
// bersama oleh server (pengirim request) dan agent (pelaksana).

// --- Tahap 4: System Information ---

// GPUInfo menjelaskan satu kartu grafis.
type GPUInfo struct {
	Name   string `json:"name"`
	RAMMB  uint64 `json:"ram_mb"`
	Driver string `json:"driver"`
}

// DiskInfo menjelaskan satu disk fisik/logis.
type DiskInfo struct {
	Name   string `json:"name"`
	SizeGB uint64 `json:"size_gb"`
}

// NetAdapter menjelaskan satu adapter jaringan.
type NetAdapter struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IP   string `json:"ip"`
}

// SysInfo adalah balasan aksi TypeSysInfo.
type SysInfo struct {
	Hostname    string       `json:"hostname"`
	Username    string       `json:"username"`
	OS          string       `json:"os"`
	Build       string       `json:"build"`
	Serial      string       `json:"serial"`
	Manufacturer string      `json:"manufacturer"`
	Model       string       `json:"model"`
	Motherboard string       `json:"motherboard"`
	BIOS        string       `json:"bios"`
	CPU         string       `json:"cpu"`
	CPUCores    int          `json:"cpu_cores"`
	RAMTotalMB  uint64       `json:"ram_total_mb"`
	GPUs        []GPUInfo    `json:"gpus"`
	Disks       []DiskInfo   `json:"disks"`
	Adapters    []NetAdapter `json:"adapters"`
}

// --- Tahap 5 & 8: Screenshot / frame layar ---

// ScreenShot adalah balasan aksi TypeScreenshot dan payload TypeScreenFrame.
type ScreenShot struct {
	Format string `json:"format"` // selalu "jpeg" untuk saat ini
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Data   string `json:"data"` // JPEG dalam base64
}

// ScreenQualityRequest dipakai TypeScreenQuality untuk mengubah kualitas stream
// Live Screen secara langsung (quality: "normal" atau "hd").
type ScreenQualityRequest struct {
	Quality string `json:"quality"`
}

// --- Tahap 6: File Explorer ---

// FSEntry adalah satu item dalam folder.
type FSEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"` // epoch detik
}

// FSPathRequest dipakai aksi yang hanya butuh satu path.
type FSPathRequest struct {
	Path string `json:"path"`
}

// FSListResponse adalah balasan TypeFSList.
type FSListResponse struct {
	Path    string    `json:"path"`
	Entries []FSEntry `json:"entries"`
}

// FSDrivesResponse adalah balasan TypeFSDrives.
type FSDrivesResponse struct {
	Drives []string `json:"drives"`
}

// FSReadResponse adalah balasan TypeFSRead (unduh file).
type FSReadResponse struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Data string `json:"data"` // isi file base64
}

// FSWriteRequest dipakai TypeFSWrite (unggah file).
type FSWriteRequest struct {
	Path string `json:"path"`
	Data string `json:"data"` // isi file base64
}

// FSTwoPathRequest dipakai rename/copy/move (sumber -> tujuan).
type FSTwoPathRequest struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

// --- Tahap 11: Process Manager ---

// ProcInfo menjelaskan satu proses.
type ProcInfo struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	MemMB  uint64  `json:"mem_mb"`
	Status string  `json:"status"`
}

// ProcListResponse adalah balasan TypeProcList.
type ProcListResponse struct {
	Processes []ProcInfo `json:"processes"`
}

// ProcKillRequest dipakai TypeProcKill.
type ProcKillRequest struct {
	PID int32 `json:"pid"`
}

// --- Tahap 12: Windows Service Manager ---

// SvcInfo menjelaskan satu Windows Service.
type SvcInfo struct {
	Name    string `json:"name"`
	Display string `json:"display"`
	Status  string `json:"status"`
}

// SvcListResponse adalah balasan TypeSvcList.
type SvcListResponse struct {
	Services []SvcInfo `json:"services"`
}

// SvcControlRequest dipakai TypeSvcControl (action: start|stop|restart).
type SvcControlRequest struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

// --- Tahap 9 & 10: Remote input ---

// MouseEvent adalah event mouse (action: move|down|up|click|rclick|dblclick|scroll).
// x,y dalam koordinat relatif 0..1 terhadap lebar/tinggi layar agar independen resolusi.
type MouseEvent struct {
	Action string  `json:"action"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Scroll int     `json:"scroll"`
}

// KeyEvent adalah event keyboard. Text untuk ketikan biasa; Key untuk tombol
// khusus (Enter, Backspace, dll); Modifiers seperti ctrl/alt/shift/win.
type KeyEvent struct {
	Text      string   `json:"text"`
	Key       string   `json:"key"`
	Modifiers []string `json:"modifiers"`
}

// --- Tahap 7: Terminal ---

// TermStart membuka shell (shell: "cmd" atau "powershell").
type TermStart struct {
	Shell string `json:"shell"`
}

// TermData membawa input/output terminal.
type TermData struct {
	Data string `json:"data"`
}
