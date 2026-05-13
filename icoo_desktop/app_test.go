package main

import (
	"fmt"
	"os/exec"
	"testing"
	"time"
)

func TestStartServerWakesPackagedServer(t *testing.T) {
	if findServerExecutable(".") == "" {
		t.Skip("icoo_llm_bridge executable not found")
	}

	app := NewApp()
	app.config.Port = 19193

	if err := app.StartServer(); err != nil {
		t.Fatalf("StartServer() error = %v", err)
	}
	defer func() {
		if err := app.StopServer(); err != nil {
			t.Fatalf("StopServer() error = %v", err)
		}
	}()

	info := app.GetServerProcessInfo()
	if !info.Running {
		t.Fatalf("server info = %+v, want running", info)
	}
	if info.PID == 0 {
		t.Fatalf("server PID = 0, info = %+v", info)
	}
}

func TestStopServerStopsExistingListener(t *testing.T) {
	app := NewApp()
	app.config.Port = 19194

	serverExe := findServerExecutable(".")
	if serverExe == "" {
		t.Skip("icoo_llm_bridge executable not found")
	}
	listenAddr := fmt.Sprintf("%s:%d", app.config.Host, app.config.Port)
	cmd := exec.Command(serverExe, "-data-dir", t.TempDir(), "-addr", listenAddr)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start external server: %v", err)
	}
	defer func() {
		if isRunning(cmd) {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !serverHealthOK(app.config.URL()) {
		time.Sleep(100 * time.Millisecond)
	}
	if !serverHealthOK(app.config.URL()) {
		t.Fatal("external server did not become ready")
	}

	if err := app.StopServer(); err != nil {
		t.Fatalf("StopServer() error = %v", err)
	}
	if serverHealthOK(app.config.URL()) {
		t.Fatal("server still responds after StopServer")
	}
}
