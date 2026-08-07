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
	log.Println("OK")

	repo := repository.NewPostgresRepo(db)       
	service := usecase.NewDownloadService(repo)
	handler := transport.NewHandler(service)      

	mux := handler.InitRoutes()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("PORT: %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Error %v", err)
	}
}