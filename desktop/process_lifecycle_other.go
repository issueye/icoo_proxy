//go:build !windows

package main

import (
	"fmt"
	"syscall"
)

// processJob is a no-op on non-Windows; process-group kill is used instead.
type processJob struct{}

func newManagedProcessJob() (*processJob, error) {
	return &processJob{}, nil
}

func (j *processJob) assign(pid int) error {
	_ = pid
	return nil
}

func (j *processJob) close() {}

// killProcessTree sends SIGKILL to the process group (bridge + plugins when Setpgid).
func killProcessTree(pid int) error {
	if pid <= 0 {
		return nil
	}
	// Negative PID = process group id (set via Setpgid in configureServerCommand).
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		// Fallback: kill the process itself.
		if err2 := syscall.Kill(pid, syscall.SIGKILL); err2 != nil {
			return fmt.Errorf("kill process tree pid=%d: %v / %v", pid, err, err2)
		}
	}
	return nil
}
