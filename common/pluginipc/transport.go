package pluginipc

// ListenConfig configures a platform IPC listener (plugin side).
type ListenConfig struct {
	// Endpoint is a Windows named pipe path or Unix socket path.
	Endpoint string
	// SecurityDescriptor is Windows-only SDDL; empty uses owner-only default.
	SecurityDescriptor string
}

// DialConfig configures a platform IPC dial (host side).
type DialConfig struct {
	Endpoint string
}
