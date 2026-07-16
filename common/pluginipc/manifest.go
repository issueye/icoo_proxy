package pluginipc

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manifest describes a plugin binary for discovery (plugin.manifest.json).
type Manifest struct {
	PluginID         string   `json:"plugin_id"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Executable       string   `json:"executable,omitempty"` // relative name hint
	Capabilities     []string `json:"capabilities"`
	SupportedIngress []string `json:"supported_ingress"`
	UpstreamKind     string   `json:"upstream_kind,omitempty"`
	Disclaimer       string   `json:"disclaimer,omitempty"`
}

// LoadManifest reads a plugin.manifest.json file.
func LoadManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.PluginID == "" {
		return nil, fmt.Errorf("pluginipc: manifest missing plugin_id")
	}
	return &m, nil
}
