package temporal

import (
	"context"
	"downloader/internal/domain"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.temporal.io/sdk/workflow"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	StatusDone = "DONE"
)

type Activities struct {
	Repo   domain.Repository
	Client *http.Client
}

type DownloadFileParams struct {
	TaskID int
	File   domain.File
}

func (a *Activities) DownloadFileActivity(ctx context.Context, params DownloadFileParams) error {
	taskID := params.TaskID
	file := params.File
	url := file.URL

	logger := slog.With(
		slog.Int("task_id", taskID),
		slog.Int("file_id", file.ID),
		slog.String("url", url),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		logger.Error("failed to create request", slog.Any("error", err))

		return a.Repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil)
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := a.Client.Do(req)
	if err != nil {
		logger.Error("download request failed", slog.Any("error", err))

		return a.Repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		logger.Info("unexpected response status", slog.Int("status", resp.StatusCode))

		return a.Repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed to read response body", slog.Any("error", err))

		return a.Repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil)
	}

	if len(bytes) == 0 {
		logger.Info("downloaded empty content")
	} else {
		logger.Info("downloaded bytes", slog.Int("size", len(bytes)))
	}

	err = a.Repo.UpdateFile(ctx, taskID, file.ID, "", bytes)
	if err != nil {
		logger.Error("failed to persist file content", slog.Any("error", err))

		return err
	}

	return nil
}

type UpdateStatusParams struct {
	TaskID int
	Status string
}

func (a *Activities) UpdateDownloadStatusActivity(
	ctx context.Context,
	params UpdateStatusParams,
) error {
	logger := slog.With(
		slog.Int("task_id", params.TaskID),
		slog.String("status", params.Status),
	)

	err := a.Repo.UpdateDownloadStatus(ctx, params.TaskID, params.Status)
	if err != nil {
		logger.Error("failed to mark download status", slog.Any("error", err))

		return err
	}

	return nil
}

type DownloadWorkflowParams struct {
	TaskID          int
	Files           []domain.File
	ActivityTimeout time.Duration
}

func DownloadWorkflow(ctx workflow.Context, params DownloadWorkflowParams) error {
	// Retry policy for activities
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: params.ActivityTimeout,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	futures := make([]workflow.Future, 0, len(params.Files))
	for _, file := range params.Files {
		f := workflow.ExecuteActivity(ctx, "DownloadFileActivity", DownloadFileParams{
			TaskID: params.TaskID,
			File:   file,
		})
		futures = append(futures, f)
	}

	for _, future := range futures {
		var result interface{}
		_ = future.Get(ctx, &result)
	}

	err := workflow.ExecuteActivity(ctx, "UpdateDownloadStatusActivity", UpdateStatusParams{
		TaskID: params.TaskID,
		Status: StatusDone,
	}).Get(ctx, nil)

	return err
}
