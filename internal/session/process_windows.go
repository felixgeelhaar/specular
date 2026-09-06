//go:build windows

package session

import (
	"os"
	"os/exec"
)

func configureProcessGroup(cmd *exec.Cmd) {
	// Windows has no process-group equivalent used here.
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Best-effort: Interrupt fails if the process is gone.
	return proc.Signal(os.Interrupt) == nil
}

func killProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
