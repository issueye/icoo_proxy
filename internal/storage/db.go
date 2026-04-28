package storage

import (
	"icoo_proxy/internal/models"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Open 打开数据库连接
func Open(root string) (*gorm.DB, error) {
	storeDir := filepath.Join(root, ".data")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return nil, err
	}

	// 打开数据库连接
	db, err := gorm.Open(sqlite.Open(filepath.Join(storeDir, "icoo_proxy.db")), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移数据库
	if err := AutoMigrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// AutoMigrate 自动迁移数据库
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.EndpointModel{},
		&models.AuthKeyModel{},
	)
}
