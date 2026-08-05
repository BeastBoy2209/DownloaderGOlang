package main

import (
	"database/sql"
	"log"
	"net/http"
	_ "github.com/lib/pq"

	"downloader/internal/repository"
	"downloader/internal/transport"
	"downloader/internal/usecase"
)

func main() {
	connStr := "host=localhost port=5432 user=admin password=admin dbname=downloader_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("что то со строкой подключния: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("рикошет: %v", err)
	}
	log.Println("есть пробитие")

	repo := repository.NewPostgresRepo(db)       
	service := usecase.NewDownloadService(repo)
	handler := transport.NewHandler(service)      

	mux := handler.InitRoutes()

	log.Println("started")
	if err := http.ListenAndServe(":2020", mux); err != nil {
		log.Fatalf("Shit happens... %v", err)
	}
}