package app

import (
	"context"
	"log/slog"
	"net"
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
	TrafficDB   *gorm.DB
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

	trafficDB, err := repository.OpenSQLite(cfg.TrafficDBPath)
	if err != nil {
		return nil, err
	}
	if err := repository.AutoMigrateTraffic(trafficDB); err != nil {
		return nil, err
	}

	repos := repository.NewRepositories(db, trafficDB)
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
	// Start the background traffic writer so DB persistence never blocks the
	// proxy hot path. It is drained on Close().
	if started, ok := services.Proxy.(interface{ StartTrafficWriter() }); ok {
		started.StartTrafficWriter()
	}
	controllers := controller.NewControllers(services)
	middlewares := middleware.NewMiddlewares(services.Auth, cfg.AllowLocalWithoutAuth)
	engine := router.New(controllers, middlewares)
	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}

	return &Container{
		Config:      cfg,
		Logger:      logger,
		DB:          db,
		TrafficDB:   trafficDB,
		Repos:       repos,
		Services:    services,
		Controllers: controllers,
		Middlewares: middlewares,
		Server:      server,
	}, nil
}

func (c *Container) Start() error {
	listener, err := net.Listen("tcp", c.Server.Addr)
	if err != nil {
		return err
	}
	c.Logger.Info("icoo_llm_bridge started", "addr", c.Config.Addr())
	go func() {
		if err := c.Server.Serve(listener); err != nil && err != http.ErrServerClosed {
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
	if c == nil {
		return nil
	}
	var closeErr error
	// Drain buffered traffic records before closing the DB so records in flight
	// are not lost on graceful shutdown.
	if closer, ok := c.Services.Proxy.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			closeErr = err
		} else if err := sqlDB.Close(); err != nil {
			closeErr = err
		}
	}
	if c.TrafficDB != nil && c.TrafficDB != c.DB {
		sqlDB, err := c.TrafficDB.DB()
		if err != nil && closeErr == nil {
			closeErr = err
		} else if err == nil {
			if err := sqlDB.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	}
	return closeErr
}
