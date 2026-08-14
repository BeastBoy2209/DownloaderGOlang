package main

import (
	"context"
	"downloader/internal/config"
	"downloader/internal/repository"
	"downloader/internal/transport"
	"downloader/internal/usecase"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

const (
	startupTimeout       = 15 * time.Second
	shutdownTimeout      = 15 * time.Second
	readHeaderTimeout    = 5 * time.Second
	serverStartedMessage = "server started"
	shutdownMessage      = "starting shutdown..."
)

func main() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
	cfg := config.Load()
	urll := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DB.User, cfg.DB.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.DB.Host, cfg.DB.Port),
		Path:   cfg.DB.Name,
	}
	q := urll.Query()
	q.Set("sslmode", "disable")
	urll.RawQuery = q.Encode()
	connStr := urll.String()
	startupCtx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	db, err := sqlx.ConnectContext(startupCtx, "pgx", connStr)
	if err != nil {
		slog.Error("sqlx can't connect to db", slog.String("error", err.Error()))

		return
	}
	defer func() {
		_ = db.Close()
	}()
	slog.Info("DB OK")

	repo := repository.NewPostgresRepo(db)
	service := usecase.NewDownloadService(repo, nil)
	handler := transport.NewHandler(service)

	e := handler.InitRoutes()
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		slog.Info(serverStartedMessage, slog.Int("port", int(cfg.Server.Port)))
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info(shutdownMessage)
	ctx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	err = server.Shutdown(ctx)
	if err != nil {
		slog.Error(
			"server was stopped cause error or timeout",
			slog.String("error", err.Error()),
		)

		return
	}

	slog.Info("server was successfully shut down")
}
