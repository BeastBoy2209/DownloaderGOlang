package usecase

import (
	"context"
	"downloader/internal/domain"
	"io"
	"net/http"
	"sync"
	"time"
)

type DownloadService struct{
	repo domain.Repository
}

func (d *DownloadService) GetDownload(taskID int) (domain.DownloadTask, error) {
    task, err := d.repo.RecieveDownload(taskID)
    return task, err 
}

func (d *DownloadService) GetFileContent(taskID int, fileID int) ([]byte, error) {
    content, err := d.repo.GetFile(taskID, fileID)
    if err != nil {
        return nil, err
    }
    return content, err
}

func (d *DownloadService) downloadSingleFile(ctx context.Context, taskID int,file domain.File ){
	url := file.URL
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil{
		d.repo.SaveFile(taskID, file.ID, "ERROR", nil)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil{
		d.repo.SaveFile(taskID, file.ID, "ERROR", nil)
		return
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil{
		d.repo.SaveFile(taskID, file.ID, "ERROR", nil)
		return
	}
	d.repo.SaveFile(taskID, file.ID, "", bytes)
}

func (d *DownloadService) runBackgroundProcess(ctx context.Context, taskID int, files []domain.File){
	var wg sync.WaitGroup
	for _, f := range files{
		wg.Add(1)
		go func(file domain.File){
			defer wg.Done()
			d.downloadSingleFile(ctx, taskID, file)
		}(f)
	}
	wg.Wait()
	d.repo.UpdateStatus(taskID, "DONE")
}

func NewDownloadService(r domain.Repository) *DownloadService{
	return &DownloadService{
		repo:r,
	}
}

func (d *DownloadService) StartDownload(urls []string, timeout string) (int, error) {
	var files []domain.File
	for _, u := range urls {
		files = append(files, domain.File{URL: u})
	}
	task := domain.DownloadTask{
		Status: "PROCESS",
		Files: files,
	}

	id, err := d.repo.TaskCreation(task)
	if err != nil {
		return 0, err
	}

	duration, durerr := time.ParseDuration(timeout)
	if durerr != nil{
		return 0, durerr
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	go d.runBackgroundProcess(ctx, id, task.Files)

	return id, nil


}