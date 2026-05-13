//go:build !windows

package main

import "os/exec"

func configureServerCommand(cmd *exec.Cmd) {}

func terminateServerByAddr(listenAddr string) error {
	return nil
}
