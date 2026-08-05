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
	connStr := "host=localhost port=5432 user=postgres password=postgres dbname=downloader_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка парсинга строки подключения: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("Нет связи с базой данных: %v", err)
	}
	log.Println("Успешное подключение к PostgreSQL!")

	repo := repository.NewPostgresRepo(db)       
	service := usecase.NewDownloadService(repo)
	handler := transport.NewHandler(service)      

	mux := handler.InitRoutes()

	log.Println("started")
	if err := http.ListenAndServe(":2020", mux); err != nil {
		log.Fatalf("Shit happens... %v", err)
	}
}