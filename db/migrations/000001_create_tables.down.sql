DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS downloads;

--migrate -path db/migrations "postgres://admin:admin@localhost:5432/downloader_db?sslmode=disable" down 1