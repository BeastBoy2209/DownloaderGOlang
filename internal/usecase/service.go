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

func (d *DownloadService) GetDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	task, err := d.repo.GetDownload(ctx, taskID)
	if err != nil {
		return domain.DownloadTask{}, fmt.Errorf("get download %d: %w", taskID, err)
	}
	return task, nil
}

func (d *DownloadService) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	content, err := d.repo.GetFileContent(ctx, taskID, fileID)
	if err != nil {
		return nil, fmt.Errorf("get file content for download %d file %d: %w", taskID, fileID, err)
	}
	return content, nil
}

func (d *DownloadService) downloadSingleFile(ctx context.Context, taskID int, file domain.File) {
	url := file.URL
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("task %d file %d (%s): failed to create request: %v", taskID, file.ID, url, err)
		if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
			log.Printf("task %d file %d (%s): failed to persist failure state: %v", taskID, file.ID, url, saveErr)
		}
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("task %d file %d (%s): download request failed: %v", taskID, file.ID, url, err)
		if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
			log.Printf("task %d file %d (%s): failed to persist failure state: %v", taskID, file.ID, url, saveErr)
		}
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("task %d file %d (%s): unexpected response status %d", taskID, file.ID, url, resp.StatusCode)
		if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
			log.Printf("task %d file %d (%s): failed to persist failure state: %v", taskID, file.ID, url, saveErr)
		}
		return
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("task %d file %d (%s): failed to read response body: %v", taskID, file.ID, url, err)
		if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
			log.Printf("task %d file %d (%s): failed to persist failure state: %v", taskID, file.ID, url, saveErr)
		}
		return
	}

	if len(bytes) == 0 {
		log.Printf("task %d file %d (%s): downloaded empty content", taskID, file.ID, url)
	} else {
		log.Printf("task %d file %d (%s): downloaded %d bytes", taskID, file.ID, url, len(bytes))
	}

	if err := d.repo.UpdateFile(ctx, taskID, file.ID, "", bytes); err != nil {
		log.Printf("task %d file %d (%s): failed to persist file content: %v", taskID, file.ID, url, err)
	}
}

func (d *DownloadService) runBackgroundProcess(ctx context.Context, cancel context.CancelFunc, taskID int, files []domain.File) {
	defer cancel()

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(10)

	for _, f := range files {
		file := f
		eg.Go(func() error {
			d.downloadSingleFile(egCtx, taskID, file)
			return nil
		})
	}

	_ = eg.Wait()
	if err := d.repo.UpdateDownloadStatus(context.Background(), taskID, "DONE"); err != nil {
		log.Printf("task %d: failed to mark download as DONE: %v", taskID, err)
	}
}

func (d *DownloadService) StartDownload(ctx context.Context, urls []string, timeout time.Duration) (int, error) {
	if len(urls) == 0 {
		return 0, fmt.Errorf("start download: no file URLs provided: %w", domain.ErrBusiness)
	}

	var files []domain.File
	for _, u := range urls {
		files = append(files, domain.File{URL: u})
	}
	task := domain.DownloadTask{
		Status: "PROCESS",
		Files:  files,
	}

	id, err := d.repo.CreateDownloadAndFiles(ctx, &task)
	if err != nil {
		return 0, fmt.Errorf("start download: %w", err)
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), timeout)
	go d.runBackgroundProcess(bgCtx, cancel, id, task.Files)

	return id, nil
}
