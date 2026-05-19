package repository

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func OpenSQLite(path string) (*gorm.DB, error) {
	if dbPath := sqliteFilesystemPath(path); dbPath != "" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, err
		}
	}
	return gorm.Open(sqlite.Open(sqliteReadWriteDSN(path)), &gorm.Config{})
}

func sqliteFilesystemPath(path string) string {
	dsn := strings.TrimSpace(path)
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return ""
	}
	if strings.HasPrefix(dsn, "file:") {
		dsn = strings.TrimPrefix(dsn, "file:")
	}
	if index := strings.Index(dsn, "?"); index >= 0 {
		dsn = dsn[:index]
	}
	return dsn
}

func sqliteReadWriteDSN(path string) string {
	dsn := strings.TrimSpace(path)
	if dsn == "" {
		return "file::memory:?mode=rwc"
	}
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + filepath.ToSlash(dsn)
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		if strings.Contains(dsn, "mode=") {
			return dsn
		}
		separator = "&"
	}
	return dsn + separator + "mode=rwc"
}
