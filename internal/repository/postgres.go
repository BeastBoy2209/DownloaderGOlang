package repository

import (
	"context"
	"downloader/internal/domain"
	"errors"
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

func (r *PostgresRepo) CreateDownloadAndFiles(ctx context.Context, task *domain.DownloadTask) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.NamedQueryContext(ctx, `
		INSERT INTO downloads (status)
		VALUES (:status)
		RETURNING id
	`, downloadCreateParams{Status: task.Status})
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, errors.New("download id was not returned")
	}

	var taskID int
	if err := rows.Scan(&taskID); err != nil {
		return 0, err
	}

	for i := range task.Files {
		rows, err := r.db.NamedQueryContext(ctx, `
			INSERT INTO files (download_id, url)
			VALUES (:download_id, :url)
			RETURNING id
		`, fileCreateParams{DownloadID: taskID, URL: task.Files[i].URL})
		if err != nil {
			return 0, err
		}
		if !rows.Next() {
			rows.Close()
			return 0, errors.New("file id was not returned")
		}
		if err := rows.Scan(&task.Files[i].ID); err != nil {
			rows.Close()
			return 0, err
		}
		if err := rows.Close(); err != nil {
			return 0, err
		}
	}
	return taskID, nil
}

func (r *PostgresRepo) UpdateFile(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var ec *string
	if errCode != "" {
		ec = &errCode
	}

	_, err := r.db.NamedExecContext(ctx, `
		UPDATE files
		SET error_code = :error_code,
		    content = :content
		WHERE id = :id AND download_id = :download_id
	`, fileUpdateParams{
		DownloadID: taskID,
		ID:         fileID,
		ErrorCode:  ec,
		Content:    content,
	})
	return err
}

func (r *PostgresRepo) UpdateDownloadStatus(ctx context.Context, taskID int, newStatus string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.NamedExecContext(ctx, `
		UPDATE downloads
		SET status = :status
		WHERE id = :id
	`, downloadUpdateStatusParams{ID: taskID, Status: newStatus})
	return err
}

func (r *PostgresRepo) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.NamedQueryContext(ctx, `
		SELECT content
		FROM files
		WHERE id = :id AND download_id = :download_id
	`, fileContentParams{DownloadID: taskID, ID: fileID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, domain.ErrClient
	}

	var content []byte
	if err := rows.Scan(&content); err != nil {
		return nil, err
	}

	return content, nil
}

func (r *PostgresRepo) GetDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.NamedQueryContext(ctx, `
		SELECT id, status
		FROM downloads
		WHERE id = :id
	`, downloadIDParams{ID: taskID})
	if err != nil {
		return domain.DownloadTask{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.DownloadTask{}, domain.ErrClient
	}

	var task downloadRow
	if err := rows.StructScan(&task); err != nil {
		return domain.DownloadTask{}, err
	}

	fileRows, err := r.db.NamedQueryContext(ctx, `
		SELECT id, download_id, url, error_code
		FROM files
		WHERE download_id = :download_id
	`, fileDownloadParams{DownloadID: taskID})
	if err != nil {
		return domain.DownloadTask{}, err
	}
	defer fileRows.Close()

	result := domain.DownloadTask{ID: task.ID, Status: task.Status}
	for fileRows.Next() {
		var row fileRow
		if err := fileRows.StructScan(&row); err != nil {
			return result, err
		}

		file := domain.File{ID: row.ID, URL: row.URL}
		if row.ErrorCode.Valid {
			file.ErrorCode = &domain.ErrorCode{Code: row.ErrorCode.String}
		}
		result.Files = append(result.Files, file)
	}
	if err := fileRows.Err(); err != nil {
		return result, err
	}

	return result, nil
}
