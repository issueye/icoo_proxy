package service

import "errors"

// ErrPluginRuntimeUnavailable is returned when no process-plugin host is configured.
var ErrPluginRuntimeUnavailable = errors.New("plugin runtime is not configured")

// ErrPluginUIDisabled is returned when AdminProxyTarget is called for a plugin
// whose host policy has admin_enabled=false.
var ErrPluginUIDisabled = errors.New("plugin UI is disabled (admin_enabled=false)")

// Backward-compatible unexported aliases used inside the service package.
var (
	errPluginRuntimeUnavailable = ErrPluginRuntimeUnavailable
	errPluginUIDisabled         = ErrPluginUIDisabled
)
