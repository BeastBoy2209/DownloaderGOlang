package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"errors"

	_ "github.com/lib/pq"

	"downloader/internal/config"
	"downloader/internal/repository"
	"downloader/internal/transport"
	"downloader/internal/usecase"
)

func main() {
	cfg := config.Load()
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("sql can't open db %v", err)
	}
	
	defer db.Close() 
	
	if err := db.Ping(); err != nil {
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

	log.Println("Server was successfully shut down")
}