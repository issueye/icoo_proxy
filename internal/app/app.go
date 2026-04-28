package app

import (
	"icoo_proxy/internal/services"
	"icoo_proxy/internal/storage"

	"gorm.io/gorm"
)

type App struct {
	db *gorm.DB

	services *services.Services
}

// NewApp 创建应用实例
func NewApp(root string) (*App, error) {
	// 初始化数据库
	db, err := storage.Open(root)
	if err != nil {
		return nil, err
	}

	// 初始化服务
	services, err := services.NewServices(db, nil)
	if err != nil {
		return nil, err
	}

	// 初始化应用
	return &App{
		db:       db,
		services: services,
	}, nil
}

func (a *App) Services() *services.Services {
	return a.services
}
