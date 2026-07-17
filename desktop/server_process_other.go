//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// configureServerCommand puts bridge in its own process group so StopServer can
// kill the whole tree (bridge + plugins) via kill(-pid).
func configureServerCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}