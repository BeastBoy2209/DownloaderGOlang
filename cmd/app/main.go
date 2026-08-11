package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"downloader/internal/config"
	"downloader/internal/repository"
	"downloader/internal/transport"
	"downloader/internal/usecase"
)

func main() {
	cfg := config.Load()
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(startupCtx, "pgx", connStr)
	if err != nil {
		log.Fatalf("sqlx can't connect db %v", err)
	}
	defer db.Close()
	log.Println("DB OK")

	repo := repository.NewPostgresRepo(db)
	service := usecase.NewDownloadService(repo, http.DefaultClient)
	handler := transport.NewHandler(service)

	mux := handler.InitRoutes()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("PORT: %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("Server started on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Starting shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server was stopped cause error or timeout: %v", err)
	}

	log.Println("server was successfully shut down")
}
