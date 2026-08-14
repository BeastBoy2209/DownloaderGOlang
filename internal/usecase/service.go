package usecase

import (
	"context"
	"downloader/internal/domain"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	downloadWorkerLimit = 10
	defaultUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/120.0.0.0 Safari/537.36"
)

type DownloadService struct {
	repo   domain.Repository
	client *http.Client
}

func NewDownloadService(r domain.Repository, client *http.Client) *DownloadService {
	if client == nil {
		client = http.DefaultClient
	}

	return &DownloadService{
		repo:   r,
		client: client,
	}
}

func (d *DownloadService) GetDownload(
	ctx context.Context,
	taskID int,
) (domain.DownloadTask, error) {
	task, err := d.repo.GetDownload(ctx, taskID)
	if err != nil {
		return domain.DownloadTask{}, fmt.Errorf("get download %d: %w", taskID, err)
	}

	return task, nil
}

func (d *DownloadService) GetFileContent(ctx context.Context, taskID, fileID int) ([]byte, error) {
	content, err := d.repo.GetFileContent(ctx, taskID, fileID)
	if err != nil {
		return nil, fmt.Errorf(
			"get file content for download %d file %d: %w",
			taskID,
			fileID,
			err,
		)
	}

	return content, nil
}

func logFileInfof(taskID, fileID int, url, format string, args ...any) {
	log.Printf("task %d file %d (%s): "+format, append([]any{taskID, fileID, url}, args...)...)
}

func logFileError(taskID, fileID int, url, message string, err error) {
	logFileInfof(taskID, fileID, url, "%s: %v", message, err)
}

func (d *DownloadService) persistFileFailureState(
	ctx context.Context,
	taskID int,
	file domain.File,
	url string,
) {
	saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil)
	if saveErr != nil {
		logFileError(taskID, file.ID, url, "failed to persist failure state", saveErr)
	}
}

func (d *DownloadService) downloadSingleFile(ctx context.Context, taskID int, file domain.File) {
	url := file.URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		logFileError(taskID, file.ID, url, "failed to create request", err)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		logFileError(taskID, file.ID, url, "download request failed", err)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		logFileInfof(taskID, file.ID, url, "unexpected response status %d", resp.StatusCode)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logFileError(taskID, file.ID, url, "failed to read response body", err)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}

	if len(bytes) == 0 {
		logFileInfof(taskID, file.ID, url, "downloaded empty content")
	} else {
		logFileInfof(taskID, file.ID, url, "downloaded %d bytes", len(bytes))
	}

	err = d.repo.UpdateFile(ctx, taskID, file.ID, "", bytes)
	if err != nil {
		logFileError(taskID, file.ID, url, "failed to persist file content", err)
	}
}

func (d *DownloadService) runBackgroundProcess(
	ctx context.Context,
	cancel context.CancelFunc,
	taskID int,
	files []domain.File,
) {
	defer cancel()

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(downloadWorkerLimit)

	for _, file := range files {
		currentFile := file
		eg.Go(func() error {
			d.downloadSingleFile(egCtx, taskID, currentFile)

			return nil
		})
	}

	_ = eg.Wait()
	err := d.repo.UpdateDownloadStatus(ctx, taskID, "DONE")
	if err != nil {
		log.Printf("task %d: failed to mark download as DONE: %v", taskID, err)
	}
}

func (d *DownloadService) StartDownload(
	ctx context.Context,
	urls []string,
	timeout time.Duration,
) (int, error) {
	if len(urls) == 0 {
		return 0, fmt.Errorf(
			"start download: no file URLs provided: %w",
			domain.ErrBusiness,
		)
	}

	files := make([]domain.File, 0, len(urls))
	for _, url := range urls {
		files = append(files, domain.File{URL: url})
	}
	task := domain.DownloadTask{
		Status: "PROCESS",
		Files:  files,
	}

	id, err := d.repo.CreateDownloadAndFiles(ctx, &task)
	if err != nil {
		return 0, fmt.Errorf("start download: %w", err)
	}

	bgCtx, cancel := context.WithTimeout(ctx, timeout)
	go d.runBackgroundProcess(bgCtx, cancel, id, task.Files)

	return id, nil
}
