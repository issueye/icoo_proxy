package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Settings holds plugin-local configuration (proxy, etc.).
type Settings struct {
	// HTTPProxy is an optional upstream proxy URL for OAuth + Grok API traffic.
	// Examples: http://127.0.0.1:7890  socks5://127.0.0.1:7891
	// Empty means use environment (HTTP_PROXY / HTTPS_PROXY / ALL_PROXY).
	HTTPProxy string `json:"http_proxy"`
}

// SettingsStore persists settings.json under the plugin data dir.
type SettingsStore struct {
	path string
	mu   sync.Mutex
}

func NewSettingsStore(dataDir string) *SettingsStore {
	return &SettingsStore{path: filepath.Join(dataDir, "settings.json")}
}

func (s *SettingsStore) Load() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil
		}
		return Settings{}, err
	}
	var out Settings
	if err := json.Unmarshal(raw, &out); err != nil {
		return Settings{}, err
	}
	out.HTTPProxy = strings.TrimSpace(out.HTTPProxy)
	return out, nil
}

func (s *SettingsStore) Save(in Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.HTTPProxy = strings.TrimSpace(in.HTTPProxy)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
