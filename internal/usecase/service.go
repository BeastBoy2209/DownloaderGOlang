package usecase

import (
	"context"
	"downloader/internal/domain"
	"downloader/internal/temporal"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
)

type DownloadService struct {
	repo           domain.Repository
	temporalClient client.Client
}

func NewDownloadService(r domain.Repository, tc client.Client) *DownloadService {
	return &DownloadService{
		repo:           r,
		temporalClient: tc,
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

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("download-task-%d", id),
		TaskQueue: temporal.DownloadTaskQueue,
	}

	params := temporal.DownloadWorkflowParams{
		TaskID: id,
		Files:  task.Files,
	}

	_, err = d.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		temporal.DownloadWorkflow,
		params,
	)
	if err != nil {
		return id, fmt.Errorf("failed to start workflow: %w", err)
	}

	return id, nil
}
