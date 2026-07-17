//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// processJob wraps a Windows Job Object with KILL_ON_JOB_CLOSE so that when
// the desktop process exits (even abnormal termination), managed bridge
// processes assigned to the job are terminated. Plugin children of bridge are
// reaped by bridge's own job on bridge exit; taskkill /T is used as a backup
// process-tree kill on explicit stop.
type processJob struct {
	handle windows.Handle
}

func newManagedProcessJob() (*processJob, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		h,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(h)
		return nil, err
	}
	return &processJob{handle: h}, nil
}

func (j *processJob) assign(pid int) error {
	if j == nil || j.handle == 0 || pid <= 0 {
		return fmt.Errorf("invalid job or pid")
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.AssignProcessToJobObject(j.handle, h)
}

// close terminates all processes in the job (KILL_ON_JOB_CLOSE) and releases the handle.
func (j *processJob) close() {
	if j == nil || j.handle == 0 {
		return
	}
	_ = windows.CloseHandle(j.handle)
	j.handle = 0
}

// killProcessTree force-kills a process and all descendants (plugins under bridge).
func killProcessTree(pid int) error {
	if pid <= 0 {
		return nil
	}
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		// taskkill returns error if process already gone — treat as success when not found.
		msg := string(out)
		if containsAny(msg, "not found", "没有找到", "not running", "ERROR: The process") {
			return nil
		}
		return fmt.Errorf("taskkill pid=%d: %w (%s)", pid, err, truncateOut(msg, 200))
	}
	return nil
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) > 0 && (len(s) >= len(p)) {
			// simple case-insensitive contains via EqualFold scan
			for i := 0; i+len(p) <= len(s); i++ {
				if equalFoldASCII(s[i:i+len(p)], p) {
					return true
				}
			}
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func truncateOut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
