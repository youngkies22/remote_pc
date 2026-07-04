// Package procs menyediakan daftar proses dan kemampuan mematikan proses.
package procs

import (
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/v4/process"

	"remote_pc/internal/protocol"
)

// List mengembalikan daftar proses yang sedang berjalan, diurutkan CPU menurun.
func List() (protocol.ProcListResponse, error) {
	all, err := process.Processes()
	if err != nil {
		return protocol.ProcListResponse{}, err
	}
	out := make([]protocol.ProcInfo, 0, len(all))
	for _, p := range all {
		name, _ := p.Name()
		if name == "" {
			continue
		}
		var memMB uint64
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			memMB = mi.RSS / (1024 * 1024)
		}
		cpu, _ := p.CPUPercent()
		status := ""
		if st, err := p.Status(); err == nil && len(st) > 0 {
			status = st[0]
		}
		out = append(out, protocol.ProcInfo{
			PID: p.Pid, Name: name, CPU: round1(cpu), MemMB: memMB, Status: status,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPU > out[j].CPU })
	return protocol.ProcListResponse{Processes: out}, nil
}

// Kill mematikan proses berdasarkan PID.
func Kill(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("proses %d tidak ditemukan: %w", pid, err)
	}
	if err := p.Kill(); err != nil {
		return fmt.Errorf("gagal mematikan proses %d: %w", pid, err)
	}
	return nil
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}
