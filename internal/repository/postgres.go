package repository

import (
	"context"
	"database/sql"
	"downloader/internal/domain"
	"errors"

	"github.com/jmoiron/sqlx"
)

type PostgresRepo struct {
	db *sqlx.DB
}

func NewPostgresRepo(db *sqlx.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) CreateDownloadAndFiles(ctx context.Context, task *domain.DownloadTask) (int, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query, args, err := sqlx.Named(`
		INSERT INTO downloads (status)
		VALUES (:status)
		RETURNING id
	`, downloadCreateParams{Status: task.Status})
	if err != nil {
		return 0, err
	}
	query = tx.Rebind(query)
	rows, err := tx.QueryxContext(ctx, query, args...)
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
	if err := rows.Close(); err != nil {
		return 0, err
	}

	for i := range task.Files {
		query, args, err := sqlx.Named(`
			INSERT INTO files (download_id, url)
			VALUES (:download_id, :url)
			RETURNING id
		`, fileCreateParams{DownloadID: taskID, URL: task.Files[i].URL})
		if err != nil {
			return 0, err
		}
		query = tx.Rebind(query)
		rows, err := tx.QueryxContext(ctx, query, args...)
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
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return taskID, nil
}

func (r *PostgresRepo) UpdateFile(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var ec *string
	if errCode != "" {
		ec = &errCode
	}

	query, args, err := sqlx.Named(`
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
	if err != nil {
		return err
	}
	query = tx.Rebind(query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepo) UpdateDownloadStatus(ctx context.Context, taskID int, newStatus string) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query, args, err := sqlx.Named(`
		UPDATE downloads
		SET status = :status
		WHERE id = :id
	`, downloadUpdateStatusParams{ID: taskID, Status: newStatus})
	if err != nil {
		return err
	}
	query = tx.Rebind(query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepo) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query, args, err := sqlx.Named(`
		SELECT content
		FROM files
		WHERE id = :id AND download_id = :download_id
	`, fileContentParams{DownloadID: taskID, ID: fileID})
	if err != nil {
		return nil, err
	}
	query = tx.Rebind(query)
	rows, err := tx.QueryxContext(ctx, query, args...)
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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return content, nil
}

func (r *PostgresRepo) GetDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return domain.DownloadTask{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query, args, err := sqlx.Named(`
		SELECT id, status
		FROM downloads
		WHERE id = :id
	`, downloadIDParams{ID: taskID})
	if err != nil {
		return domain.DownloadTask{}, err
	}
	query = tx.Rebind(query)
	rows, err := tx.QueryxContext(ctx, query, args...)
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

	query, args, err = sqlx.Named(`
		SELECT id, download_id, url, error_code
		FROM files
		WHERE download_id = :download_id
	`, fileDownloadParams{DownloadID: taskID})
	if err != nil {
		return domain.DownloadTask{}, err
	}
	query = tx.Rebind(query)
	fileRows, err := tx.QueryxContext(ctx, query, args...)
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

	if err := tx.Commit(); err != nil {
		return result, err
	}

	return result, nil
}
