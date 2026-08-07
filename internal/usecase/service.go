package usecase

import (
	"context"
	"downloader/internal/domain"
	"io"
	"log"
	"net/http"

	"golang.org/x/sync/errgroup"
	"time"
)

type DownloadService struct {
	repo domain.Repository
}

func NewDownloadService(r domain.Repository) *DownloadService {
	return &DownloadService{
		repo: r,
	}
}

func (d *DownloadService) GetDownload(ctx context.Context, taskID int) (domain.DownloadTask, error) {
	task, err := d.repo.RecieveDownload(ctx, taskID)
	return task, err
}

func (d *DownloadService) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
	content, err := d.repo.GetFile(ctx, taskID, fileID)
	if err != nil {
		return nil, err
	}
	return content, err
}

func (d *DownloadService) downloadSingleFile(ctx context.Context, taskID int, file domain.File) {
	url := file.URL
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("[Task %d] Creation request error %s: %v", taskID, url, err)
		d.repo.SaveFile(ctx, taskID, file.ID, "ERROR", nil)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[Task %d] downloading error %s: %v", taskID, url, err)
		d.repo.SaveFile(ctx, taskID, file.ID, "ERROR", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Task %d] no content or server error %s [error: %d]", taskID, url, resp.StatusCode)
		d.repo.SaveFile(ctx, taskID, file.ID, "ERROR", nil)
		return
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Task %d] bodyreading error %s: %v", taskID, url, err)
		d.repo.SaveFile(ctx, taskID, file.ID, "ERROR", nil)
		return
	}

	if len(bytes) == 0 {
		log.Printf("[Task %d] Link: %s is empty 0KB", taskID, url)
	} else {
		log.Printf("[Task %d] OK (%d b), link:  %s", taskID, len(bytes), url)
	}

	d.repo.SaveFile(ctx, taskID, file.ID, "", bytes)
}

func (d *DownloadService) runBackgroundProcess(ctx context.Context, cancel context.CancelFunc, taskID int, files []domain.File) {
	defer cancel()
	
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(10) // лимит го рутин

	for _, f := range files {
		file := f
		eg.Go(func() error {
			d.downloadSingleFile(egCtx, taskID, file)
			return nil
		})
	}
	
	_ = eg.Wait()
	d.repo.UpdateStatus(context.Background(), taskID, "DONE")
}

func (d *DownloadService) StartDownload(ctx context.Context, urls []string, timeout time.Duration) (int, error) {
	var files []domain.File
	for _, u := range urls {
		files = append(files, domain.File{URL: u})
	}
	task := domain.DownloadTask{
		Status: "PROCESS",
		Files:  files,
	}

	id, err := d.repo.TaskCreation(ctx, &task)
	if err != nil {
		return 0, domain.ErrServer
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), timeout)
	go d.runBackgroundProcess(bgCtx, cancel, id, task.Files)

	return id, nil
}