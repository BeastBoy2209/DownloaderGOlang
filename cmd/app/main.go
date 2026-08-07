package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"errors"

	"github.com/jackc/pgx/v5"

	"downloader/internal/config"
	"downloader/internal/repository"
	"downloader/internal/transport"
	"downloader/internal/usecase"
)

func main() {
	cfg := config.Load()
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

	db, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("pgx can't connect db %v", err)
	}
	
	defer db.Close(context.Background()) 
	
	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("database cant start %v", err)
	}
	log.Println("DB OK")

	repo := repository.NewPostgresRepo(db)
	service := usecase.NewDownloadService(repo)
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