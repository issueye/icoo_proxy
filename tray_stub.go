//go:build !windows

package main

type trayController struct{}

func (a *App) registerTray(icon []byte) {}

func (a *App) startTray() {}
