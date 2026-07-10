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

func TestSaveServerConfigRestartsManagedServer(t *testing.T) {
	if findServerExecutable(".") == "" {
		t.Skip("icoo_llm_bridge executable not found")
	}

	app := NewApp()
	app.config.Port = 19196
	oldURL := app.config.URL()
	if err := app.StartServer(); err != nil {
		t.Fatalf("StartServer() error = %v", err)
	}
	t.Cleanup(func() {
		if err := app.StopServer(); err != nil {
			t.Errorf("StopServer() error = %v", err)
		}
	})

	cfg := app.GetServerConfig()
	cfg.Port = 19197
	if err := app.SaveServerConfig(cfg); err != nil {
		t.Fatalf("SaveServerConfig() error = %v", err)
	}
	if serverHealthOK(oldURL) {
		t.Fatal("old server address still responds after config reload")
	}
	if !serverHealthOK(cfg.URL()) {
		t.Fatal("new server address is not healthy after config reload")
	}
	if info := app.GetServerProcessInfo(); !info.Running || info.ListenAddr != "127.0.0.1:19197" {
		t.Fatalf("server info after reload = %+v", info)
	}
}

func TestStopServerRefusesToKillExistingListener(t *testing.T) {
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

	if err := app.StopServer(); err == nil {
		t.Fatal("StopServer() error = nil, want unmanaged-process refusal")
	}
	if !serverHealthOK(app.config.URL()) {
		t.Fatal("unmanaged server was terminated")
	}
}
