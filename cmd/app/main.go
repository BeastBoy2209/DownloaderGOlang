package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	_ "github.com/lib/pq"

	"downloader/internal/config"
	"downloader/internal/repository"
	"downloader/internal/transport"
	"downloader/internal/usecase"
)

func main() {
	cfg := config.Load()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("что то со строкой подключния: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("не заупстился: %v", err)
	}
	log.Println("запустился")

	repo := repository.NewPostgresRepo(db)       
	service := usecase.NewDownloadService(repo)
	handler := transport.NewHandler(service)      

	mux := handler.InitRoutes()

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("порт: %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("упал %v", err)
	}
}