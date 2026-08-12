package repository

import "database/sql"

type downloadCreateParams struct {
	Status string `db:"status"`
}

type downloadRow struct {
	ID     int    `db:"id"`
	Status string `db:"status"`
}

type downloadIDParams struct {
	ID int `db:"id"`
}

type downloadUpdateStatusParams struct {
	ID     int    `db:"id"`
	Status string `db:"status"`
}

type fileCreateParams struct {
	DownloadID int    `db:"download_id"`
	URL        string `db:"url"`
}

type fileUpdateParams struct {
	DownloadID int     `db:"download_id"`
	ID         int     `db:"id"`
	ErrorCode  *string `db:"error_code"`
	Content    []byte  `db:"content"`
}

type fileContentParams struct {
	DownloadID int `db:"download_id"`
	ID         int `db:"id"`
}

type fileDownloadParams struct {
	DownloadID int `db:"download_id"`
}

type fileRow struct {
	ID         int            `db:"id"`
	DownloadID int            `db:"download_id"`
	URL        string         `db:"url"`
	ErrorCode  sql.NullString `db:"error_code"`
}
