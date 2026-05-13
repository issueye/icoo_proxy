package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type App struct {
	ctx             context.Context
	mu              sync.RWMutex
	tray            *trayController
	config          ServerConfig
	serverCmd       *exec.Cmd
	serverExe       string
	serverDir       string
	serverArgs      []string
	serverStartedAt time.Time
	serverLogPath   string
	serverLog       *os.File
	serverLastError string
}

type ServerProcessInfo struct {
	Running          bool     `json:"running"`
	Status           string   `json:"status"`
	PID              int      `json:"pid"`
	Executable       string   `json:"executable"`
	WorkingDirectory string   `json:"working_directory"`
	DataDir          string   `json:"data_dir"`
	ListenAddr       string   `json:"listen_addr"`
	StartedAt        string   `json:"started_at"`
	Args             []string `json:"args"`
	LogPath          string   `json:"log_path"`
	LastError        string   `json:"last_error"`
}

func NewApp() *App {
	return &App{
		config: loadConfig(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startTray()
	go func() {
		if err := a.StartServer(); err != nil {
			a.mu.Lock()
			a.serverLastError = err.Error()
			a.mu.Unlock()
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.StopServer()
}

func isRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	// ProcessState is set after Wait() returns; nil means still running
	return cmd.ProcessState == nil
}

// StartServer starts the icoo_llm_bridge child process.
func (a *App) StartServer() error {
	a.mu.Lock()

	if isRunning(a.serverCmd) {
		a.mu.Unlock()
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		a.mu.Unlock()
		return err
	}
	exeDir := filepath.Dir(exePath)
	a.config = normalizeConfig(a.config)
	_ = saveConfig(a.config)

	listenAddr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	if serverHealthOK(a.config.URL()) {
		a.serverExe = findServerExecutable(exeDir)
		a.serverDir = exeDir
		a.serverArgs = []string{"-data-dir", ".", "-addr", listenAddr}
		a.serverLastError = ""
		a.mu.Unlock()
		return nil
	}

	serverExe := findServerExecutable(exeDir)
	if serverExe == "" {
		a.mu.Unlock()
		return fmt.Errorf("bridge.exe not found; run build-all.ps1 or place bridge.exe next to icoo_desktop.exe")
	}

	args := []string{"-data-dir", ".", "-addr", listenAddr}
	cmd := exec.Command(serverExe, args...)
	cmd.Dir = exeDir
	logPath := filepath.Join(exeDir, ".data", "icoo_llm_bridge.log")
	logFile, err := openServerLog(logPath)
	if err != nil {
		a.mu.Unlock()
		return err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	configureServerCommand(cmd)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		a.mu.Unlock()
		return fmt.Errorf("failed to start server: %w", err)
	}

	a.serverCmd = cmd
	a.serverExe = serverExe
	a.serverDir = exeDir
	a.serverArgs = args
	a.serverStartedAt = time.Now()
	a.serverLogPath = logPath
	a.serverLog = logFile
	a.serverLastError = ""
	a.mu.Unlock()

	// goroutine to clean up after process exits
	go func() {
		_ = cmd.Wait()
		a.mu.Lock()
		if a.serverCmd == cmd {
			a.serverCmd = nil
			if a.serverLog == logFile {
				a.serverLog = nil
			}
			a.serverStartedAt = time.Time{}
			if cmd.ProcessState != nil && !cmd.ProcessState.Success() {
				a.serverLastError = fmt.Sprintf("server exited: %s", cmd.ProcessState.String())
			}
		}
		a.mu.Unlock()
		_ = logFile.Close()
	}()

	if err := waitForServerHealth(a.config.URL(), cmd, 3*time.Second); err != nil {
		a.mu.Lock()
		a.serverLastError = err.Error()
		a.mu.Unlock()
		return err
	}

	return nil
}

func (a *App) GetServerProcessInfo() ServerProcessInfo {
	a.mu.RLock()
	url := a.config.URL()
	cmd := a.serverCmd
	info := ServerProcessInfo{
		Running:          false,
		Status:           "stopped",
		PID:              0,
		Executable:       a.serverExe,
		WorkingDirectory: a.serverDir,
		DataDir:          ".",
		ListenAddr:       fmt.Sprintf("%s:%d", a.config.Host, a.config.Port),
		Args:             append([]string(nil), a.serverArgs...),
		LogPath:          a.serverLogPath,
		LastError:        a.serverLastError,
	}
	if !a.serverStartedAt.IsZero() {
		info.StartedAt = a.serverStartedAt.Format("2006-01-02 15:04:05")
	}
	a.mu.RUnlock()

	if isRunning(cmd) {
		info.Running = true
		info.Status = "running"
		info.PID = cmd.Process.Pid
	} else if serverHealthOK(url) {
		info.Running = true
		info.Status = "running"
	}
	return info
}

func findServerExecutable(exeDir string) string {
	candidates := []string{
		filepath.Join(exeDir, "bridge.exe"),
		filepath.Join(exeDir, "icoo_llm_bridge.exe"),
		filepath.Join(exeDir, "build", "bin", "bridge.exe"),
		filepath.Join(exeDir, "..", "bridge.exe"),
		filepath.Join(exeDir, "..", "icoo_llm_bridge.exe"),
		filepath.Join(exeDir, "..", "icoo_llm_bridge", "build", "bridge.exe"),
		filepath.Join(exeDir, "..", "icoo_llm_bridge", "bridge.exe"),
		filepath.Join(exeDir, "..", "icoo_proxy", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "icoo_llm_bridge.exe"),
		filepath.Join(exeDir, "..", "..", "icoo_llm_bridge", "build", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "icoo_llm_bridge", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "icoo_proxy", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "..", "icoo_llm_bridge", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "..", "icoo_llm_bridge", "build", "bridge.exe"),
		filepath.Join(exeDir, "..", "..", "..", "icoo_llm_bridge", "cmd", "bridge", "bridge.exe"),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "bridge.exe"),
			filepath.Join(cwd, "icoo_llm_bridge.exe"),
			filepath.Join(cwd, "icoo_llm_bridge", "build", "bridge.exe"),
			filepath.Join(cwd, "icoo_proxy", "bridge.exe"),
			filepath.Join(cwd, "..", "icoo_llm_bridge", "bridge.exe"),
			filepath.Join(cwd, "..", "icoo_llm_bridge", "build", "bridge.exe"),
			filepath.Join(cwd, "..", "icoo_proxy", "bridge.exe"),
		)
	}
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if info, err := os.Stat(cleaned); err == nil && !info.IsDir() {
			return cleaned
		}
	}
	return ""
}

// StopServer stops the icoo_llm_bridge child process.
func (a *App) StopServer() error {
	a.mu.Lock()
	cmd := a.serverCmd
	listenAddr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	serverURL := a.config.URL()
	if isRunning(cmd) {
		if err := cmd.Process.Kill(); err != nil {
			a.mu.Unlock()
			return err
		}
	} else if serverHealthOK(serverURL) {
		if err := terminateServerByAddr(listenAddr); err != nil {
			a.mu.Unlock()
			return err
		}
	}
	a.serverCmd = nil
	a.serverStartedAt = time.Time{}
	a.serverLastError = ""
	if a.serverLog != nil {
		_ = a.serverLog.Close()
		a.serverLog = nil
	}
	a.mu.Unlock()

	for i := 0; i < 20; i++ {
		if !serverHealthOK(serverURL) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server is still responding at %s after stop", listenAddr)
}

// ServerStatus returns the current server status
func (a *App) ServerStatus() string {
	a.mu.RLock()
	url := a.config.URL()
	cmd := a.serverCmd
	a.mu.RUnlock()
	if isRunning(cmd) || serverHealthOK(url) {
		return "running"
	}
	return "stopped"
}

// GetServerConfig returns the current server configuration
func (a *App) GetServerConfig() ServerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// SaveServerConfig saves and applies new server configuration
func (a *App) SaveServerConfig(cfg ServerConfig) error {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port <= 0 {
		cfg.Port = 18181
	}

	a.mu.Lock()
	a.config = cfg
	a.mu.Unlock()

	return saveConfig(cfg)
}

// GetAppVersion returns the app version
func (a *App) GetAppVersion() string {
	return "0.1.0"
}

func openServerLog(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create server log dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open server log: %w", err)
	}
	_, _ = fmt.Fprintf(file, "\n[%s] starting icoo_llm_bridge\n", time.Now().Format("2006-01-02 15:04:05"))
	return file, nil
}

func waitForServerHealth(url string, cmd *exec.Cmd, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil {
			return fmt.Errorf("server exited before ready: %s", cmd.ProcessState.String())
		}
		if serverHealthOK(url) {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("server did not become ready within %s", timeout)
}

func serverHealthOK(url string) bool {
	client := http.Client{Timeout: 700 * time.Millisecond}
	resp, err := client.Get(url + "/healthz")
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
