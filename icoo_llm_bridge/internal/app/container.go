package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gorm.io/gorm"

	"icoo_llm_bridge/internal/config"
	"icoo_llm_bridge/internal/controller"
	"icoo_llm_bridge/internal/middleware"
	"icoo_llm_bridge/internal/repository"
	"icoo_llm_bridge/internal/router"
	"icoo_llm_bridge/internal/service"
	"icoo_llm_bridge/internal/utils/ai_llm_proxy"
)

type Options struct {
	ConfigPath   string
	DataDir      string
	AddrOverride string
}

type Container struct {
	Config      config.Config
	Logger      *slog.Logger
	DB          *gorm.DB
	Repos       repository.Repositories
	Services    service.Services
	Controllers controller.Controllers
	Middlewares middleware.Middlewares
	Server      *http.Server
}

func NewContainer(options Options) (*Container, error) {
	cfg, err := config.Load(options.ConfigPath)
	if err != nil {
		return nil, err
	}
	cfg.ApplyDataDir(options.DataDir)
	if err := cfg.ApplyAddrOverride(options.AddrOverride); err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	db, err := repository.OpenSQLite(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	if err := repository.AutoMigrate(db); err != nil {
		return nil, err
	}

	repos := repository.NewRepositories(db)
	if err := repository.SeedDefaults(context.Background(), repos); err != nil {
		return nil, err
	}

	converter := ai_llm_proxy.NewProtocolConverter()
	services := service.NewServices(service.Deps{
		Config:    cfg,
		Logger:    logger,
		Repos:     repos,
		Converter: converter,
	})
	controllers := controller.NewControllers(services)
	middlewares := middleware.NewMiddlewares(services.Auth, cfg.AllowLocalWithoutAuth)
	engine := router.New(controllers, middlewares)
	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	return &Container{
		Config:      cfg,
		Logger:      logger,
		DB:          db,
		Repos:       repos,
		Services:    services,
		Controllers: controllers,
		Middlewares: middlewares,
		Server:      server,
	}, nil
}

func (c *Container) Start() error {
	c.Logger.Info("icoo_llm_bridge started", "addr", c.Config.Addr())
	go func() {
		if err := c.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.Logger.Error("server error", "error", err)
		}
	}()
	return nil
}

func (c *Container) Shutdown(ctx context.Context) error {
	if c == nil || c.Server == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, c.Config.ShutdownTimeout)
	defer cancel()
	return c.Server.Shutdown(shutdownCtx)
}

func (c *Container) Close() error {
	if c == nil || c.DB == nil {
		return nil
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
