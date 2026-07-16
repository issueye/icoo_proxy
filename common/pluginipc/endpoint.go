package pluginipc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
)

// NewEndpoint generates a platform-appropriate IPC endpoint path with a random suffix.
// pluginID should be a short slug (e.g. "grokbuild").
// dataDir is used for Unix socket parent path; ignored on Windows.
func NewEndpoint(pluginID, dataDir string) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(b[:])
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\icoo-plugin-%s-%s`, pluginID, suffix), nil
	}
	dir := filepath.Join(dataDir, "plugins", pluginID)
	return filepath.Join(dir, fmt.Sprintf("run-%s.sock", suffix)), nil
}

// NewHostToken returns a 32-byte hex token for handshake auth.
func NewHostToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
