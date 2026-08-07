CREATE TABLE IF NOT EXISTS downloads (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
    id SERIAL PRIMARY KEY,
    download_id INTEGER NOT NULL REFERENCES downloads(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    error_code VARCHAR(255),
    content BYTEA
);

--migrate -path db/migrations -database "postgres://admin:admin@localhost:5432/downloader_db?sslmode=disable" up
