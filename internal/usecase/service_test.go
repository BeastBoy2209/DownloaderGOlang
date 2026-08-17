package usecase

import (
	"bytes"
	"context"
	"downloader/internal/domain"
	"downloader/internal/mocks"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newMockHTTPClient(fn roundTripperFunc) *http.Client {
	return &http.Client{Transport: fn}
}

var okHTTPClient = newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte("data"))),
	}, nil
})

type errorReader struct{}

var logCaptureMu sync.Mutex

func (e *errorReader) Read(p []byte) (int, error) { return 0, errors.New("read error") }
func (e *errorReader) Close() error               { return nil }

func TestDownloadService_GetDownload(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	expectedTask := domain.DownloadTask{
		ID:     42,
		Status: "finished",
		Files:  []domain.File{{ID: 1, URL: "https://example.com/file1"}},
	}
	mockRepo.EXPECT().
		GetDownload(gomock.Any(), 42).
		Return(expectedTask, nil).
		Times(1)

	svc := NewDownloadService(mockRepo, nil)
	gotTask, err := svc.GetDownload(context.Background(), 42)

	require.NoError(t, err)
	assert.Equal(t, expectedTask, gotTask)
}

func TestDownloadService_GetFileContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		taskID    int
		fileID    int
		mockSetup func(repo *mocks.MockRepository)
		wantData  []byte
		wantErr   error
	}{
		{
			name:   "success – returns file bytes",
			taskID: 42,
			fileID: 7,
			mockSetup: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					GetFileContent(gomock.Any(), 42, 7).
					Return([]byte("payload"), nil).
					Times(1)
			},
			wantData: []byte("payload"),
		},
		{
			name:   "repo returns error",
			taskID: 13,
			fileID: 2,
			mockSetup: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					GetFileContent(gomock.Any(), 13, 2).
					Return(nil, domain.ErrServer).
					Times(1)
			},
			wantData: nil,
			wantErr:  domain.ErrServer,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			logCaptureMu.Lock()
			defer logCaptureMu.Unlock()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockRepository(ctrl)
			tc.mockSetup(mockRepo)

			svc := NewDownloadService(mockRepo, nil)
			data, err := svc.GetFileContent(context.Background(), tc.taskID, tc.fileID)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantData, data)
		})
	}
}

func TestDownloadService_downloadSingleFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		file          domain.File
		httpClient    *http.Client
		expectSave    func(repo *mocks.MockRepository)
		wantLogSubstr string
	}{
		{
			name: "request creation error (bad URL)",
			file: domain.File{ID: 1, URL: "://bad_url"},
			httpClient: newMockHTTPClient(
				func(req *http.Request) (*http.Response, error) {
					return nil, nil //nolint:nilnil
				},
			),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					UpdateFile(gomock.Any(), gomock.Any(), 1, "ERROR", nil).
					Times(1)
			},
			wantLogSubstr: "failed to create request",
		},
		{
			name: "client Do returns error",
			file: domain.File{ID: 2, URL: "https://123.com/fail"},
			httpClient: newMockHTTPClient(
				func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network down")
				},
			),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					UpdateFile(gomock.Any(), gomock.Any(), 2, "ERROR", nil).
					Times(1)
			},
			wantLogSubstr: "download request failed",
		},
		{
			name: "non-200 status code",
			file: domain.File{ID: 3, URL: "https://123.com/500"},
			httpClient: newMockHTTPClient(
				func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewReader(nil)),
					}, nil
				},
			),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					UpdateFile(gomock.Any(), gomock.Any(), 3, "ERROR", nil).
					Times(1)
			},
			wantLogSubstr: "unexpected response status",
		},
		{
			name: "io.ReadAll returns error",
			file: domain.File{ID: 4, URL: "https://123.com/readerror"},
			httpClient: newMockHTTPClient(
				func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(&errorReader{}),
					}, nil
				},
			),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					UpdateFile(gomock.Any(), gomock.Any(), 4, "ERROR", nil).
					Times(1)
			},
			wantLogSubstr: "failed to read response body",
		},
		{
			name: "success – empty body",
			file: domain.File{ID: 5, URL: "https://123.com/empty"},
			httpClient: newMockHTTPClient(
				func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}

					return resp, nil
				},
			),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					UpdateFile(gomock.Any(), gomock.Any(), 5, "", []byte{}).
					Times(1)
			},
			wantLogSubstr: "downloaded empty content",
		},
		{
			name: "success – non empty body",
			file: domain.File{ID: 6, URL: "https://123.com/ok"},
			httpClient: newMockHTTPClient(
				func(req *http.Request) (*http.Response, error) {
					bodyContent := []byte("hello world")

					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							bytes.NewReader(bodyContent),
						),
					}, nil
				},
			),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().
					UpdateFile(
						gomock.Any(),
						gomock.Any(),
						6,
						"",
						[]byte("hello world"),
					).
					Times(1)
			},
			wantLogSubstr: "downloaded bytes",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			logCaptureMu.Lock()
			defer logCaptureMu.Unlock()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockRepository(ctrl)
			tc.expectSave(mockRepo)

			svc := NewDownloadService(mockRepo, tc.httpClient)

			var logBuf bytes.Buffer
			oldLogger := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&logBuf, nil)))
			defer slog.SetDefault(oldLogger)

			svc.downloadSingleFile(context.Background(), 123, tc.file)

			if tc.wantLogSubstr != "" {
				assert.Contains(t, logBuf.String(), tc.wantLogSubstr)
			}
		})
	}
}

func TestDownloadService_runBackgroundProcess(t *testing.T) {
	t.Parallel()
	logCaptureMu.Lock()
	defer logCaptureMu.Unlock()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	files := []domain.File{
		{ID: 1, URL: "https://123.com/1"},
		{ID: 2, URL: "https://123.com/2"},
		{ID: 3, URL: "https://123.com/3"},
	}
	taskID := 123

	for _, file := range files {
		mockRepo.EXPECT().
			UpdateFile(gomock.Any(), taskID, file.ID, "", []byte("data")).
			Times(1)
	}
	mockRepo.EXPECT().UpdateDownloadStatus(gomock.Any(), taskID, "DONE").Times(1)

	svc := NewDownloadService(mockRepo, okHTTPClient)

	bgCtx, cancel := context.WithCancel(context.Background())
	svc.runBackgroundProcess(bgCtx, cancel, taskID, files)
}

func TestDownloadService_StartDownload(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		logCaptureMu.Lock()
		defer logCaptureMu.Unlock()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repoMock := mocks.NewMockRepository(ctrl)

		urls := []string{"https://123.com/one", "https://123.com/two"}
		timeout := 200 * time.Millisecond
		expectedID := 777

		repoMock.EXPECT().
			CreateDownloadAndFiles(
				gomock.Any(),
				gomock.AssignableToTypeOf(&domain.DownloadTask{}),
			).
			DoAndReturn(
				func(_ context.Context, task *domain.DownloadTask) (int, error) {
					assert.Equal(t, "PROCESS", task.Status)
					assert.Len(t, task.Files, len(urls))
					for i, file := range task.Files {
						assert.Equal(t, urls[i], file.URL)
					}

					return expectedID, nil
				},
			).
			Times(1)

		done := make(chan struct{}, len(urls)+1)

		for range urls {
			repoMock.EXPECT().
				UpdateFile(
					gomock.Any(),
					expectedID,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).
				Do(func(_ context.Context, _, _ int, _ string, _ []byte) {
					done <- struct{}{}
				}).
				Times(1)
		}
		repoMock.EXPECT().
			UpdateDownloadStatus(gomock.Any(), expectedID, "DONE").
			Do(func(_ context.Context, _ int, _ string) {
				done <- struct{}{}
			}).
			Times(1)

		mockClient := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("file content")),
				Header:     make(http.Header),
			}, nil
		})

		svc := NewDownloadService(repoMock, mockClient)
		id, err := svc.StartDownload(context.Background(), urls, timeout)
		require.NoError(t, err)
		assert.Equal(t, expectedID, id)

		timeoutWait := time.After(2 * time.Second)
		for range len(urls) + 1 {
			select {
			case <-done:
			case <-timeoutWait:
				t.Fatalf("timeout: background process didn't finish")
			}
		}
	})

	t.Run("empty urls returns business error", func(t *testing.T) {
		t.Parallel()
		logCaptureMu.Lock()
		defer logCaptureMu.Unlock()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repoMock := mocks.NewMockRepository(ctrl)

		svc := NewDownloadService(repoMock, nil)
		id, err := svc.StartDownload(context.Background(), nil, time.Second)

		require.ErrorIs(t, err, domain.ErrBusiness)
		assert.Zero(t, id)
	})
}
