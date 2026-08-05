package repository

import (
	"database/sql"
	"downloader/internal/domain"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{
		db: db,
	}
}

func (r *PostgresRepo) TaskCreation(task domain.DownloadTask) (int, error) {
	var taskID int
	err := r.db.QueryRow("INSERT INTO downloads (status) VALUES ($1) RETURNING id", task.Status).Scan(&taskID)
	if err != nil {
		return 0, err
	}
	for i := range task.Files {
		err := r.db.QueryRow("INSERT INTO files (download_id, url) VALUES ($1, $2) RETURNING id", taskID, task.Files[i].URL).Scan(&task.Files[i].ID)
		if err != nil {
			return 0, err
		}
	}
	return taskID, nil
}

func (p *PostgresRepo) SaveFile(taskID int, fileID int, errCode string, content []byte) error {
	_, err := p.db.Exec("UPDATE files SET error_code = $1, content = $2 WHERE id = $3 AND download_id = $4", errCode, content, fileID, taskID)
	return err
}

func (p *PostgresRepo) UpdateStatus(taskID int, newStatus string) error {
	_, err := p.db.Exec("UPDATE downloads SET status = $1 WHERE id = $2", newStatus, taskID)
	return err
}

func (p *PostgresRepo) GetFile(taskID int, fileID int) ([]byte, error) {
	var content []byte
	err := p.db.QueryRow("SELECT content FROM files WHERE id = $1 AND download_id = $2", fileID, taskID).Scan(&content)
	return content, err
}

func (p *PostgresRepo) RecievingDownload(taskID int) (domain.DownloadTask, error) {
	var task domain.DownloadTask
	err := p.db.QueryRow("SELECT id, status FROM downloads WHERE id = $1", taskID).Scan(&task.ID, &task.Status)
	if err != nil {
		return task, err
	}

	rows, err := p.db.Query("SELECT id, url, error_code FROM files WHERE download_id = $1", taskID)
	if err != nil {
		return task, err
	}
	defer rows.Close()

	for rows.Next() {
		var f domain.File
		var nullErr sql.NullString
		
		err := rows.Scan(&f.ID, &f.URL, &nullErr)
		if err != nil {
			return task, err
		}
		if nullErr.Valid {
			f.ErrorCode = nullErr.String
		}
		task.Files = append(task.Files, f)
	}
	if err := rows.Err(); err != nil {
		return task, err
	}

	return task, nil
}