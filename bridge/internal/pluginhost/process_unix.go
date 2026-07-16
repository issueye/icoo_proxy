//go:build unix

package pluginhost

import (
	"os/exec"
	"syscall"
)

// configurePluginCommand puts the plugin in a new process group so the host
// can signal the whole group on shutdown / force-kill.
func configurePluginCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
