//go:generate go run go.uber.org/mock/mockgen@latest -destination=../mocks/mock_repository.go -package=mocks downloader/internal/domain Repository
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
	GetFileContent(context.Context, int, int) ([]byte, error)
	CreateDownloadAndFiles(context.Context, *DownloadTask) (int, error)
	GetDownload(context.Context, int) (DownloadTask, error)
	UpdateDownloadStatus(context.Context, int, string) error
	UpdateFile(context.Context, int, int, string, []byte) error
}
