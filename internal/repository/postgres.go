package repository

import (
	"context"
	"downloader/internal/domain"
	"sync"

	"github.com/jackc/pgx/v5"
)

type PostgresRepo struct {
	conn *pgx.Conn
	mu   sync.Mutex
}

func NewPostgresRepo(conn *pgx.Conn) *PostgresRepo {
	return &PostgresRepo{
		conn: conn,
	}
}

func (r *PostgresRepo) TaskCreation(ctx context.Context, task *domain.DownloadTask) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var taskID int
	err := r.conn.QueryRow(ctx, "INSERT INTO downloads (status) VALUES ($1) RETURNING id", task.Status).Scan(&taskID)
	if err != nil {
		return 0, err
	}
	for i := range task.Files {
		err := r.conn.QueryRow(ctx, "INSERT INTO files (download_id, url) VALUES ($1, $2) RETURNING id", taskID, task.Files[i].URL).Scan(&task.Files[i].ID)
		if err != nil {
			return 0, err
		}
	}
	return taskID, nil
}

func (p *PostgresRepo) SaveFile(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var ec *string
	if errCode != "" {
		ec = &errCode
	}

	_, err := p.conn.Exec(ctx, "UPDATE files SET error_code = $1, content = $2 WHERE id = $3 AND download_id = $4", ec, content, fileID, taskID)
	return err
}

func (p *PostgresRepo) UpdateStatus(ctx context.Context, taskID int, newStatus string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.conn.Exec(ctx, "UPDATE downloads SET status = $1 WHERE id = $2", newStatus, taskID)
	return err
}

func (p *PostgresRepo) GetFile(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var content []byte
	err := p.conn.QueryRow(ctx, "SELECT content FROM files WHERE id = $1 AND download_id = $2", fileID, taskID).Scan(&content)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrClient
	}
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (p *PostgresRepo) RecieveDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var task domain.DownloadTask
	err := p.conn.QueryRow(ctx, "SELECT id, status FROM downloads WHERE id = $1", taskID).Scan(&task.ID, &task.Status)
	if err == pgx.ErrNoRows {
		return task, domain.ErrClient
	}
	if err != nil {
		return task, err
	}

	rows, err := p.conn.Query(ctx, "SELECT id, url, error_code FROM files WHERE download_id = $1", taskID)
	if err != nil {
		return task, err
	}
	defer rows.Close()

	for rows.Next() {
		var f domain.File
		var ec *string
		
		err := rows.Scan(&f.ID, &f.URL, &ec)
		if err != nil {
			return task, err
		}
		if ec != nil && *ec != "" {
			f.ErrorCode = &domain.ErrorCode{Code: *ec}
		}
		task.Files = append(task.Files, f)
	}
	if err := rows.Err(); err != nil {
		return task, err
	}

	return task, nil
}