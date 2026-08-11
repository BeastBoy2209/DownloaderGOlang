package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"downloader/internal/domain"
	"downloader/internal/mocks"
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

func (e *errorReader) Read(p []byte) (int, error) { return 0, errors.New("read error") }
func (e *errorReader) Close() error               { return nil }

func TestDownloadService_GetDownload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	expectedTask := domain.DownloadTask{
		ID:     42,
		Status: "finished",
		Files:  []domain.File{{ID: 1, URL: "https://example.com/file1"}},
	}
	mockRepo.EXPECT().
		RecieveDownload(gomock.Any(), 42).
		Return(expectedTask, nil).
		Times(1)

	svc := NewDownloadService(mockRepo, nil) // nil → fallback to http.DefaultClient
	gotTask, err := svc.GetDownload(context.Background(), 42)

	assert.NoError(t, err)
	assert.Equal(t, expectedTask, gotTask)
}

func TestDownloadService_GetFileContent(t *testing.T) {
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
					GetFile(gomock.Any(), 42, 7).
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
					GetFile(gomock.Any(), 13, 2).
					Return(nil, domain.ErrServer).
					Times(1)
			},
			wantData: nil,
			wantErr:  domain.ErrServer,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockRepository(ctrl)
			tc.mockSetup(mockRepo)

			svc := NewDownloadService(mockRepo, nil)
			data, err := svc.GetFileContent(context.Background(), tc.taskID, tc.fileID)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantData, data)
		})
	}
}

func TestDownloadService_downloadSingleFile(t *testing.T) {
	cases := []struct {
		name          string
		file          domain.File
		httpClient    *http.Client
		expectSave    func(repo *mocks.MockRepository)
		wantLogSubstr string
	}{
		{
			name:       "request creation error (bad URL)",
			file:       domain.File{ID: 1, URL: "://bad_url"},
			httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) { return nil, nil }),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().SaveFile(gomock.Any(), gomock.Any(), 1, "ERROR", nil).Times(1)
			},
			wantLogSubstr: "Creation request error",
		},
		{
			name: "client Do returns error",
			file: domain.File{ID: 2, URL: "https://123.com/fail"},
			httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			}),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().SaveFile(gomock.Any(), gomock.Any(), 2, "ERROR", nil).Times(1)
			},
			wantLogSubstr: "downloading error",
		},
		{
			name: "non-200 status code",
			file: domain.File{ID: 3, URL: "https://123.com/500"},
			httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader(nil)),
				}, nil
			}),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().SaveFile(gomock.Any(), gomock.Any(), 3, "ERROR", nil).Times(1)
			},
			wantLogSubstr: "no content or server error",
		},
		{
			name: "io.ReadAll returns error",
			file: domain.File{ID: 4, URL: "https://123.com/readerror"},
			httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(&errorReader{}),
				}, nil
			}),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().SaveFile(gomock.Any(), gomock.Any(), 4, "ERROR", nil).Times(1)
			},
			wantLogSubstr: "bodyreading error",
		},
		{
			name: "success – empty body",
			file: domain.File{ID: 5, URL: "https://123.com/empty"},
			httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}, nil
			}),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().SaveFile(gomock.Any(), gomock.Any(), 5, "", []byte{}).Times(1)
			},
			wantLogSubstr: "is empty 0KB",
		},
		{
			name: "success – non empty body",
			file: domain.File{ID: 6, URL: "https://123.com/ok"},
			httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("hello world"))),
				}, nil
			}),
			expectSave: func(repo *mocks.MockRepository) {
				repo.EXPECT().SaveFile(gomock.Any(), gomock.Any(), 6, "", []byte("hello world")).Times(1)
			},
			wantLogSubstr: "OK (11 b)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockRepository(ctrl)
			tc.expectSave(mockRepo)

			svc := NewDownloadService(mockRepo, tc.httpClient)

			var logBuf bytes.Buffer
			oldOut := log.Writer()
			log.SetOutput(&logBuf)
			defer log.SetOutput(oldOut)

			svc.downloadSingleFile(context.Background(), 123, tc.file)

			if tc.wantLogSubstr != "" {
				assert.Contains(t, logBuf.String(), tc.wantLogSubstr)
			}
		})
	}
}

func TestDownloadService_runBackgroundProcess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)

	files := []domain.File{
		{ID: 1, URL: "https://123.com/1"},
		{ID: 2, URL: "https://123.com/2"},
		{ID: 3, URL: "https://123.com/3"},
	}
	taskID := 123

	for _, f := range files {
		mockRepo.EXPECT().SaveFile(gomock.Any(), taskID, f.ID, "", []byte("data")).Times(1)
	}
	mockRepo.EXPECT().UpdateStatus(gomock.Any(), taskID, "DONE").Times(1)

	svc := NewDownloadService(mockRepo, okHTTPClient)

	bgCtx, cancel := context.WithCancel(context.Background())
	svc.runBackgroundProcess(bgCtx, cancel, taskID, files)
}

func TestDownloadService_StartDownload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repoMock := mocks.NewMockRepository(ctrl)

		urls := []string{"https://123.com/one", "https://123.com/two"}
		timeout := 200 * time.Millisecond
		expectedID := 777

		repoMock.EXPECT().
			TaskCreation(gomock.Any(), gomock.AssignableToTypeOf(&domain.DownloadTask{})).
			DoAndReturn(func(_ context.Context, task *domain.DownloadTask) (int, error) {
				assert.Equal(t, "PROCESS", task.Status)
				assert.Len(t, task.Files, len(urls))
				for i, f := range task.Files {
					assert.Equal(t, urls[i], f.URL)
				}
				return expectedID, nil
			}).
			Times(1)

		done := make(chan struct{}, len(urls)+1)

		for range urls {
			repoMock.EXPECT().
				SaveFile(gomock.Any(), expectedID, gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, _ int, _ int, _ string, _ []byte) { done <- struct{}{} }).
				Times(1)
		}
		repoMock.EXPECT().
			UpdateStatus(gomock.Any(), expectedID, "DONE").
			Do(func(_ context.Context, _ int, _ string) { done <- struct{}{} }).
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
		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)

		timeoutWait := time.After(2 * time.Second)
		for i := 0; i < len(urls)+1; i++ {
			select {
			case <-done:
			case <-timeoutWait:
				t.Fatalf("timeout: background process didn't finish")
			}
		}
	})

	t.Run("empty urls returns business error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repoMock := mocks.NewMockRepository(ctrl)

		svc := NewDownloadService(repoMock, nil)
		id, err := svc.StartDownload(context.Background(), nil, time.Second)

		assert.ErrorIs(t, err, domain.ErrBusiness)
		assert.Zero(t, id)
	})
}
