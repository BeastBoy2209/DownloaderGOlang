package temporal

import (
	"downloader/internal/domain"
	"downloader/internal/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"
)

func TestDownloadWorkflow(t *testing.T) {
	t.Parallel()
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()
	activities := &Activities{}
	env.RegisterActivity(activities)

	params := DownloadWorkflowParams{
		TaskID: 123,
		Files: []domain.File{
			{ID: 1, URL: "http://example.com/1"},
			{ID: 2, URL: "http://example.com/2"},
		},
		ActivityTimeout: 10 * time.Minute,
	}

	env.OnActivity("DownloadFileActivity", mock.Anything, mock.Anything).Return(nil).Times(2)

	expectedUpdateParams := UpdateStatusParams{
		TaskID: 123,
		Status: StatusDone,
	}
	env.OnActivity(
		"UpdateDownloadStatusActivity",
		mock.Anything,
		expectedUpdateParams,
	).Return(nil).Times(1)

	env.ExecuteWorkflow(DownloadWorkflow, params)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestUpdateDownloadStatusActivity(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	mockRepo.EXPECT().
		UpdateDownloadStatus(gomock.Any(), 123, StatusDone).
		Return(nil).
		Times(1)
	activities := &Activities{
		Repo: mockRepo,
	}
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(activities.UpdateDownloadStatusActivity)

	params := UpdateStatusParams{
		TaskID: 123,
		Status: StatusDone,
	}
	val, err := env.ExecuteActivity(activities.UpdateDownloadStatusActivity, params)
	require.NoError(t, err)

	require.False(t, val.HasValue())
}
