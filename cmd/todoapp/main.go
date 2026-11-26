package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/makmanu/golang_todoapp/internal/core/config"
	pgxpool "github.com/makmanu/golang_todoapp/internal/core/postgres/pool/pgx"
	"github.com/makmanu/golang_todoapp/internal/core/transport/http/server"
	"github.com/makmanu/golang_todoapp/internal/features/users/repository/postgres"
	"github.com/makmanu/golang_todoapp/internal/features/users/service"
	usershttp "github.com/makmanu/golang_todoapp/internal/features/users/transport/http"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	cfg := config.NewConfigMust()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, log); err != nil {
		log.Error("application stopped", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := pgxpool.New(ctx, cfg.PostgresURL, cfg.PostgresTimeout, cfg.PostgresMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()
	repository := postgres.NewUsersRepository(pool)
	usersService := service.NewUsersService(repository)
	handler := usershttp.NewUsersHTTPHandler(usersService)
	router := server.NewAPIVersionRouter(server.ApiVersion1)
	router.AddRoutes(handler.Routes()...)
	httpServer := server.NewHTTPServer(cfg.HTTPAddress, cfg.ShutdownTimeout, log)
	httpServer.RegisterAPIRouters(router)
	return httpServer.Run(ctx)
}
