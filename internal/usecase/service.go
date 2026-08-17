package usecase

import (
	"context"
	"downloader/internal/domain"
	"fmt"
	"io"
	"log/slog"
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

func (d *DownloadService) persistFileFailureState(
	ctx context.Context,
	taskID int,
	file domain.File,
	url string,
) {
	saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil)
	if saveErr != nil {
		slog.Default().Error("failed to persist failure state",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Any("error", saveErr),
		)
	}
}

func (d *DownloadService) downloadSingleFile(ctx context.Context, taskID int, file domain.File) {
	url := file.URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		slog.Default().Error("failed to create request",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Any("error", err),
		)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		slog.Default().Error("download request failed",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Any("error", err),
		)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		slog.Default().Info("unexpected response status",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Int("status", resp.StatusCode),
		)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Any("error", err),
		)
		d.persistFileFailureState(ctx, taskID, file, url)

		return
	}

	if len(bytes) == 0 {
		slog.Info("downloaded empty content",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
		)
	} else {
		slog.Info("downloaded bytes",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Int("size", len(bytes)),
		)
	}

	err = d.repo.UpdateFile(ctx, taskID, file.ID, "", bytes)
	if err != nil {
		slog.Error("failed to persist file content",
			slog.Int("task_id", taskID),
			slog.Int("file_id", file.ID),
			slog.String("url", url),
			slog.Any("error", err),
		)
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
		slog.Default().Error("failed to mark download as DONE",
			slog.Int("task_id", taskID),
			slog.Any("error", err),
		)
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
