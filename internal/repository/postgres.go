package repository

import (
	"context"
	"downloader/internal/domain"
	"sync"

	"github.com/jmoiron/sqlx"
)

type PostgresRepo struct {
	db *sqlx.DB
	mu sync.Mutex
}

func NewPostgresRepo(db *sqlx.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) TaskCreation(ctx context.Context, task *domain.DownloadTask) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var taskID int
	err := r.db.GetContext(ctx, &taskID, "INSERT INTO downloads (status) VALUES ($1) RETURNING id", task.Status)
	if err != nil {
		return 0, err
	}
	for i := range task.Files {
		err := r.db.GetContext(ctx, &task.Files[i].ID, "INSERT INTO files (download_id, url) VALUES ($1, $2) RETURNING id", taskID, task.Files[i].URL)
		if err != nil {
			return 0, err
		}
	}
	return taskID, nil
}

func (r *PostgresRepo) SaveFile(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var ec *string
	if errCode != "" {
		ec = &errCode
	}

	_, err := r.db.ExecContext(ctx, "UPDATE files SET error_code = $1, content = $2 WHERE id = $3 AND download_id = $4", ec, content, fileID, taskID)
	return err
}

func (r *PostgresRepo) UpdateStatus(ctx context.Context, taskID int, newStatus string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.ExecContext(ctx, "UPDATE downloads SET status = $1 WHERE id = $2", newStatus, taskID)
	return err
}

func (r *PostgresRepo) GetFile(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var rows []struct {
		Content []byte `db:"content"`
	}
	err := r.db.SelectContext(ctx, &rows, "SELECT content FROM files WHERE id = $1 AND download_id = $2", fileID, taskID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrClient
	}
	return rows[0].Content, nil
}

func (r *PostgresRepo) RecieveDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var tasks []domain.DownloadTask
	err := r.db.SelectContext(ctx, &tasks, "SELECT id, status FROM downloads WHERE id = $1", taskID)
	if err != nil {
		return domain.DownloadTask{}, err
	}
	if len(tasks) == 0 {
		return domain.DownloadTask{}, domain.ErrClient
	}
	task := tasks[0]

	type fileRow struct {
		ID        int    `db:"id"`
		URL       string `db:"url"`
		ErrorCode string `db:"error_code"`
	}

	var rows []fileRow
	err = r.db.SelectContext(ctx, &rows, "SELECT id, url, COALESCE(error_code, '') AS error_code FROM files WHERE download_id = $1", taskID)
	if err != nil {
		return task, err
	}

	for _, row := range rows {
		file := domain.File{ID: row.ID, URL: row.URL}
		if row.ErrorCode != "" {
			file.ErrorCode = &domain.ErrorCode{Code: row.ErrorCode}
		}
		task.Files = append(task.Files, file)
	}

	return task, nil
}
