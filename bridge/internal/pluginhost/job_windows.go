//go:build windows

package pluginhost

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// jobHolder keeps a Windows Job Object handle so plugins die with the host.
type jobHolder struct {
	handle windows.Handle
}

func newKillOnCloseJob() (*jobHolder, error) {
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
	return &jobHolder{handle: h}, nil
}

func (j *jobHolder) assign(pid uint32) error {
	if j == nil {
		return nil
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.AssignProcessToJobObject(j.handle, h)
}

func (j *jobHolder) close() {
	if j != nil && j.handle != 0 {
		_ = windows.CloseHandle(j.handle)
		j.handle = 0
	}
}

// attachProcessToJob is best-effort; failures are non-fatal for plugin start.
func attachProcessToJob(job *jobHolder, pid int) error {
	if job == nil || pid <= 0 {
		return fmt.Errorf("invalid job/pid")
	}
	return job.assign(uint32(pid))
}
