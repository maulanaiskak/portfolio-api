package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/maulanaiskak/portfolio-api/internal/config"
	"github.com/maulanaiskak/portfolio-api/internal/database"
	"github.com/maulanaiskak/portfolio-api/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}

	pool, err := database.NewPool(ctx, cfg.DbConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	if cfg.MigrateOnBoot {
		if err := database.Migrate(pool); err != nil {
			return err
		}
		slog.Info("migrations applied")
	}

	svc := server.NewServices(pool)

	if cfg.Email != "" {
		admin, err := svc.AdminUser.EnsureAdmin(ctx, cfg.Email, cfg.PasswordBcryptHash)
		if err != nil {
			return err
		}
		slog.Info("admin user ensured", "id", admin.ID)
	}

	return server.Run(ctx, cfg, server.NewRouter(cfg, pool, svc))
}
