package repository

import (
	"context"
	"database/sql"
	"downloader/internal/domain"
	"fmt"

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
		return 0, fmt.Errorf("begin transaction for creating download and files: %w", err)
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
		return 0, fmt.Errorf("prepare download insert query: %w", err)
	}
	query = tx.Rebind(query)
	rows, err := tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("execute download insert query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("create download and files: download id was not returned: %w", domain.ErrServer)
	}

	var taskID int
	if err := rows.Scan(&taskID); err != nil {
		return 0, fmt.Errorf("scan created download id: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("close download insert rows: %w", err)
	}

	for i := range task.Files {
		query, args, err := sqlx.Named(`
			INSERT INTO files (download_id, url)
			VALUES (:download_id, :url)
			RETURNING id
		`, fileCreateParams{DownloadID: taskID, URL: task.Files[i].URL})
		if err != nil {
			return 0, fmt.Errorf("prepare file insert query for download %d file %d: %w", taskID, i, err)
		}
		query = tx.Rebind(query)
		rows, err := tx.QueryxContext(ctx, query, args...)
		if err != nil {
			return 0, fmt.Errorf("execute file insert query for download %d file %d: %w", taskID, i, err)
		}
		if !rows.Next() {
			rows.Close()
			return 0, fmt.Errorf("create file for download %d: file id was not returned: %w", taskID, domain.ErrServer)
		}
		if err := rows.Scan(&task.Files[i].ID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan created file id for download %d file %d: %w", taskID, i, err)
		}
		if err := rows.Close(); err != nil {
			return 0, fmt.Errorf("close file insert rows for download %d file %d: %w", taskID, i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit create download and files transaction: %w", err)
	}
	return taskID, nil
}

func (r *PostgresRepo) UpdateFile(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction for updating file: %w", err)
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
		return fmt.Errorf("prepare file update query for download %d file %d: %w", taskID, fileID, err)
	}
	query = tx.Rebind(query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("execute file update query for download %d file %d: %w", taskID, fileID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit file update transaction for download %d file %d: %w", taskID, fileID, err)
	}

	return nil
}

func (r *PostgresRepo) UpdateDownloadStatus(ctx context.Context, taskID int, newStatus string) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction for updating download status: %w", err)
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
		return fmt.Errorf("prepare download status update query for download %d: %w", taskID, err)
	}
	query = tx.Rebind(query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("execute download status update query for download %d: %w", taskID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit download status update transaction for download %d: %w", taskID, err)
	}

	return nil
}

func (r *PostgresRepo) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin transaction for reading file content: %w", err)
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
		return nil, fmt.Errorf("prepare file content query for download %d file %d: %w", taskID, fileID, err)
	}
	query = tx.Rebind(query)
	rows, err := tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("execute file content query for download %d file %d: %w", taskID, fileID, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("file content for download %d file %d not found: %w", taskID, fileID, domain.ErrClient)
	}

	var content []byte
	if err := rows.Scan(&content); err != nil {
		return nil, fmt.Errorf("scan file content for download %d file %d: %w", taskID, fileID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit file content read transaction for download %d file %d: %w", taskID, fileID, err)
	}

	return content, nil
}

func (r *PostgresRepo) GetDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return domain.DownloadTask{}, fmt.Errorf("begin transaction for reading download: %w", err)
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
		return domain.DownloadTask{}, fmt.Errorf("prepare download query for download %d: %w", taskID, err)
	}
	query = tx.Rebind(query)
	rows, err := tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return domain.DownloadTask{}, fmt.Errorf("execute download query for download %d: %w", taskID, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.DownloadTask{}, fmt.Errorf("download %d not found: %w", taskID, domain.ErrClient)
	}

	var task downloadRow
	if err := rows.StructScan(&task); err != nil {
		return domain.DownloadTask{}, fmt.Errorf("scan download %d: %w", taskID, err)
	}

	query, args, err = sqlx.Named(`
		SELECT id, download_id, url, error_code
		FROM files
		WHERE download_id = :download_id
	`, fileDownloadParams{DownloadID: taskID})
	if err != nil {
		return domain.DownloadTask{}, fmt.Errorf("prepare file list query for download %d: %w", taskID, err)
	}
	query = tx.Rebind(query)
	fileRows, err := tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return domain.DownloadTask{}, fmt.Errorf("execute file list query for download %d: %w", taskID, err)
	}
	defer fileRows.Close()

	result := domain.DownloadTask{ID: task.ID, Status: task.Status}
	for fileRows.Next() {
		var row fileRow
		if err := fileRows.StructScan(&row); err != nil {
			return result, fmt.Errorf("scan file row for download %d file %d: %w", taskID, row.ID, err)
		}

		file := domain.File{ID: row.ID, URL: row.URL}
		if row.ErrorCode.Valid {
			file.ErrorCode = &domain.ErrorCode{Code: row.ErrorCode.String}
		}
		result.Files = append(result.Files, file)
	}
	if err := fileRows.Err(); err != nil {
		return result, fmt.Errorf("iterate file rows for download %d: %w", taskID, err)
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit download read transaction for download %d: %w", taskID, err)
	}

	return result, nil
}
