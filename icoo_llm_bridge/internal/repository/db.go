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
	db, err := gorm.Open(sqlite.Open(sqliteReadWriteDSN(path)), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	configureSQLite(db)
	return db, nil
}

// configureSQLite tunes the connection pool for SQLite. Because SQLite serializes
// writes through a single database file, allowing multiple writer connections is
// the primary cause of "database is locked" errors under concurrent traffic. We
// cap the pool to a single open connection, which makes every write effectively
// serialized by the driver and lets busy_timeout ride out brief lock contention.
func configureSQLite(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
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

// sqlitePragma is appended to the DSN via the go-sqlite driver's _pragma query
// parameter. These are applied on every new connection so the pool (size 1) stays
// consistent regardless of reconnects.
const sqlitePragma = "_pragma=journal_mode(WAL)" +
	"&_pragma=busy_timeout(5000)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)"

func sqliteReadWriteDSN(path string) string {
	dsn := strings.TrimSpace(path)
	if dsn == "" {
		return "file::memory:?mode=rwc&" + sqlitePragma
	}
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + filepath.ToSlash(dsn)
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		if strings.Contains(dsn, "mode=") {
			return dsn + "&" + sqlitePragma
		}
		separator = "&"
	}
	return dsn + separator + "mode=rwc&" + sqlitePragma
}
