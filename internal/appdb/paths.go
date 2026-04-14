package appdb

import (
	"os"
	"path/filepath"
)

func WorkingDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return workingDir
}

func DBPath() string {
	return filepath.Join(WorkingDir(), "icoo_proxy.db")
}

func LegacyConfigPath() string {
	return filepath.Join(WorkingDir(), "icoo_proxy.toml")
}

func KeyPath() string {
	return filepath.Join(WorkingDir(), "icoo_proxy.key")
}
