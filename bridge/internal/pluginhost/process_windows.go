//go:build windows

package pluginhost

import (
	"os/exec"
	"syscall"
)

// configurePluginCommand applies Windows process attributes.
// Full Job Object KILL_ON_JOB_CLOSE is applied in attachJobObject after Start
// when available; CreationFlags ensure hidden console for desktop use.
func configurePluginCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
