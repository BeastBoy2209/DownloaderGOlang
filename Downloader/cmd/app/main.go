package main

import (
	"database/sql"
	"log"
	"net/http"

	// Бланк-импорт: мы не используем драйвер напрямую, 
	// он нужен только для того, чтобы зарегистрировать себя в database/sql
	_ "github.com/lib/pq" 
	
	"downloader/internal/repository"
	"downloader/internal/usecase"
)

func main() {
	connStr := "host=localhost port=5432 user=postgres password=postgres dbname=downloader_db sslmode=disable"
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Не удалось инициализировать подключение к БД: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}
	log.Println("Успешно подключились к базе данных!")
	repo := repository.NewPostgresRepo(db)
	service := usecase.NewDownloadService(repo)

	///Место под HTTP
	
	// заглушка вместо htTP
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Сервер загрузок работает!"))
	})

	log.Println("Запуск сервера на порту :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}