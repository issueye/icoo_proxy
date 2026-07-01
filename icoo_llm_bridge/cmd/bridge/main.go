package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"icoo_llm_bridge/internal/app"
)

var Version = "2.0.1"

func main() {
	configPath := flag.String("config", "", "path to config.toml")
	dataDir := flag.String("data-dir", "", "data directory override")
	addr := flag.String("addr", "", "listen address override, for example 127.0.0.1:18181")
	flag.Parse()

	container, err := app.NewContainer(app.Options{
		ConfigPath:   *configPath,
		DataDir:      *dataDir,
		AddrOverride: *addr,
	})
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := container.Close(); err != nil {
			slog.Error("failed to close app", "error", err)
		}
	}()

	if err := container.Start(); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := container.Shutdown(context.Background()); err != nil {
		slog.Error("failed to shutdown server", "error", err)
		os.Exit(1)
	}
}
