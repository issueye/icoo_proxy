//go:build !windows

package main

import "os/exec"

func configureServerCommand(cmd *exec.Cmd) {}
