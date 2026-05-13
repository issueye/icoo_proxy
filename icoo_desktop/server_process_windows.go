//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const createNoWindow = 0x08000000

func configureServerCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}

func terminateServerByAddr(listenAddr string) error {
	pids, err := findListeningPIDs(listenAddr)
	if err != nil {
		return err
	}
	if len(pids) == 0 {
		return fmt.Errorf("no process is listening on %s", listenAddr)
	}
	for _, pid := range pids {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find process %d: %w", pid, err)
		}
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", pid, err)
		}
	}
	return nil
}

func findListeningPIDs(listenAddr string) ([]int, error) {
	out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("query listening ports: %w", err)
	}
	needle := normalizeNetstatAddr(listenAddr)
	seen := map[int]bool{}
	var pids []int
	for _, raw := range bytes.Split(out, []byte{'\n'}) {
		line := strings.TrimSpace(string(raw))
		if line == "" || !strings.Contains(normalizeNetstatAddr(line), needle) {
			continue
		}
		if !strings.Contains(strings.ToUpper(line), "LISTEN") && !strings.Contains(line, "侦听") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil || pid <= 0 || seen[pid] {
			continue
		}
		seen[pid] = true
		pids = append(pids, pid)
	}
	return pids, nil
}

func normalizeNetstatAddr(value string) string {
	return strings.ReplaceAll(value, "0.0.0.0:", "127.0.0.1:")
}
