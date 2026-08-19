package usecase

import (
	"context"
	"downloader/internal/domain"
	"downloader/internal/mocks"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/mock/gomock"
)

type dummyClient struct {
	client.Client

	Err error
}

type dummyRun struct {
	client.WorkflowRun
}

func (c *dummyClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) (client.WorkflowRun, error) {
	if c.Err != nil {
		return nil, c.Err
	}

	return &dummyRun{}, nil
}

func TestDownloadService_GetDownload(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	mockRepo.EXPECT().GetDownload(gomock.Any(), 1).Return(domain.DownloadTask{ID: 1}, nil)

	svc := NewDownloadService(mockRepo, &dummyClient{}, "TEST_QUEUE", 10*time.Minute)
	task, err := svc.GetDownload(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, task.ID)
}

func TestDownloadService_GetFileContent(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	mockRepo.EXPECT().GetFileContent(gomock.Any(), 1, 2).Return([]byte("hello"), nil)

	svc := NewDownloadService(mockRepo, &dummyClient{}, "TEST_QUEUE", 10*time.Minute)
	content, err := svc.GetFileContent(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), content)
}

func TestDownloadService_StartDownload(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	urls := []string{"http://example.com/1", "http://example.com/2"}

	mockRepo.EXPECT().CreateDownloadAndFiles(gomock.Any(), gomock.Any()).Return(777, nil)

	svc := NewDownloadService(mockRepo, &dummyClient{}, "TEST_QUEUE", 10*time.Minute)
	id, err := svc.StartDownload(context.Background(), urls, 10*time.Second)
	require.NoError(t, err)
	assert.Equal(t, 777, id)
}

func TestDownloadService_StartDownload_EmptyUrls(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	svc := NewDownloadService(mockRepo, &dummyClient{}, "TEST_QUEUE", 10*time.Minute)
	id, err := svc.StartDownload(context.Background(), []string{}, 10*time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrBusiness)
	assert.Equal(t, 0, id)
}

func TestDownloadService_StartDownload_WorkflowError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	urls := []string{"http://example.com/1"}
	mockRepo.EXPECT().CreateDownloadAndFiles(gomock.Any(), gomock.Any()).Return(777, nil)

	expectedErr := errors.New("workflow error")
	svc := NewDownloadService(
		mockRepo,
		&dummyClient{Err: expectedErr},
		"TEST_QUEUE",
		10*time.Minute,
	)
	id, err := svc.StartDownload(context.Background(), urls, 10*time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 777, id)
}
