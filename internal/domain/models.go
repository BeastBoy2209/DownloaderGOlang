package domain

import "context"

type ErrorCode struct {
	Code string `json:"code"`
}

type File struct {
	ID        int        `json:"file_id,omitempty"`
	URL       string     `json:"url"`
	Content   []byte     `json:"-"`
	ErrorCode *ErrorCode `json:"error,omitempty"`
}

type DownloadTask struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Files  []File `json:"files,omitempty"`
}

type Repository interface {
	GetFile(context.Context, int, int) ([]byte, error)
	TaskCreation(context.Context, *DownloadTask) (int, error)
	RecieveDownload(context.Context, int) (DownloadTask, error)
	UpdateStatus(context.Context, int, string) error
	SaveFile(context.Context, int, int, string, []byte) error
}
