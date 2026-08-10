package usecase

import (
    "bytes"
	"time"
    "context"
    "errors"
    "io"
    "log"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "go.uber.org/mock/gomock"

    "downloader/internal/domain"
    "downloader/internal/mocks"
)

func TestDownloadService_GetDownload(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    mockRepo := mocks.NewMockRepository(ctrl)
    expectedTask := domain.DownloadTask{
        ID:     42,
        Status: "finished",
        Files: []domain.File{{ID: 1, URL: "https://example.com/file1"}},
    }
    mockRepo.EXPECT().
        RecieveDownload(gomock.Any(), 42).
        Return(expectedTask, nil).
        Times(1)

    svc := NewDownloadService(mockRepo)
    gotTask, err := svc.GetDownload(context.Background(), 42)

    assert.NoError(t, err, "ошибка не ожидается")
    assert.Equal(t, expectedTask, gotTask, "полученная задача должна совпадать с тем, что вернул мок")
}

func TestDownloadService_GetFileContent(t *testing.T) {
    type testCase struct {
        name      string
        taskID    int
        fileID    int
        mockSetup func(repo *mocks.MockRepository)
        wantData  []byte
        wantErr   error
    }
    cases := []testCase{
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
            wantErr:  nil,
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
            if tc.mockSetup != nil {
                tc.mockSetup(mockRepo)
            }
            svc := NewDownloadService(mockRepo)
            data, err := svc.GetFileContent(context.Background(), tc.taskID, tc.fileID)
            if tc.wantErr != nil {
                assert.ErrorIs(t, err, tc.wantErr, "expected error %v, got %v", tc.wantErr, err)
            } else {
                assert.NoError(t, err, "unexpected error")
            }
            assert.Equal(t, tc.wantData, data, "returned file bytes do not match expected")
        })
    }
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)
func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
type errorReader struct{}
func (e *errorReader) Read(p []byte) (int, error) { return 0, errors.New("read error") }
func (e *errorReader) Close() error{ return nil }

func TestDownloadService_downloadSingleFile(t *testing.T) {
    origTransport := http.DefaultClient.Transport
    defer func() { http.DefaultClient.Transport = origTransport }()

    type testCase struct {
        name          string
        file          domain.File
        transport     roundTripperFunc
        wantStatus    string
        wantData      []byte
        expectSave    func(repo *mocks.MockRepository)
        wantLogSubstr string
    }

    cases := []testCase{
        {
            name: "request creation error (bad URL)",
            file: domain.File{ID: 1, URL: "://bad_url"},
            transport: func(req *http.Request) (*http.Response, error) { return nil, nil },
            wantStatus: "ERROR",
            wantData:   nil,
            expectSave: func(repo *mocks.MockRepository) {
                repo.EXPECT().
                    SaveFile(gomock.Any(), gomock.Any(), 1, "ERROR", nil).
                    Times(1)
            },
            wantLogSubstr: "Creation request error",
        },
        {
            name: "client Do returns error",
            file: domain.File{ID: 2, URL: "https://123.com/fail"},
            transport: func(req *http.Request) (*http.Response, error) { return nil, errors.New("network down") },
            wantStatus: "ERROR",
            wantData:   nil,
            expectSave: func(repo *mocks.MockRepository) {
                repo.EXPECT().
                    SaveFile(gomock.Any(), gomock.Any(), 2, "ERROR", nil).
                    Times(1)
            },
            wantLogSubstr: "downloading error",
        },
        {
            name: "non‑200 status code",
            file: domain.File{ID: 3, URL: "https://123.com/500"},
            transport: func(req *http.Request) (*http.Response, error) {
                resp := &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(bytes.NewReader([]byte{}))}
                return resp, nil
            },
            wantStatus: "ERROR",
            wantData:   nil,
            expectSave: func(repo *mocks.MockRepository) {
                repo.EXPECT().
                    SaveFile(gomock.Any(), gomock.Any(), 3, "ERROR", nil).
                    Times(1)
            },
            wantLogSubstr: "no content or server error",
        },
        {
            name: "io.ReadAll returns error",
            file: domain.File{ID: 4, URL: "https://123.com/readerror"},
            transport: func(req *http.Request) (*http.Response, error) {
                resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(&errorReader{})}
                return resp, nil
            },
            wantStatus: "ERROR",
            wantData:   nil,
            expectSave: func(repo *mocks.MockRepository) {
                repo.EXPECT().
                    SaveFile(gomock.Any(), gomock.Any(), 4, "ERROR", nil).
                    Times(1)
            },
            wantLogSubstr: "bodyreading error",
        },
        {
            name: "success – empty body",
            file: domain.File{ID: 5, URL: "https://123.com/empty"},
            transport: func(req *http.Request) (*http.Response, error) {
                resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte{}))}
                return resp, nil
            },
            wantStatus: "",
            wantData:   []byte{},
            expectSave: func(repo *mocks.MockRepository) {
                repo.EXPECT().
                    SaveFile(gomock.Any(), gomock.Any(), 5, "", []byte{}).
                    Times(1)
            },
            wantLogSubstr: "is empty 0KB",
        },
        {
            name: "success – non empty body",
            file: domain.File{ID: 6, URL: "https://123.com/ok"},
            transport: func(req *http.Request) (*http.Response, error) {
                resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte("hello world")))}
                return resp, nil
            },
            wantStatus: "",
            wantData:   []byte("hello world"),
            expectSave: func(repo *mocks.MockRepository) {
                repo.EXPECT().
                    SaveFile(gomock.Any(), gomock.Any(), 6, "", []byte("hello world")).
                    Times(1)
            },
            wantLogSubstr: "OK (11 b)",
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            mockRepo := mocks.NewMockRepository(ctrl)
            if tc.expectSave != nil {
                tc.expectSave(mockRepo)
            }
            http.DefaultClient.Transport = roundTripperFunc(tc.transport)
            svc := NewDownloadService(mockRepo)
            var logBuf bytes.Buffer
            oldOut := log.Writer()
            log.SetOutput(&logBuf)
            defer log.SetOutput(oldOut)
            svc.downloadSingleFile(context.Background(), 123, tc.file)
            if tc.wantLogSubstr != "" {
                assert.Contains(t, logBuf.String(), tc.wantLogSubstr, "expected log to contain %q", tc.wantLogSubstr)
            }
        })
    }
}

func TestDownloadService_runBackgroundProcess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	newMock := mocks.NewMockRepository(ctrl)
	svc := NewDownloadService(newMock)

	// подмена HTTP‑транспорта
	origTransport := http.DefaultClient.Transport
	defer func() { http.DefaultClient.Transport = origTransport }()
	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("data"))),
		}, nil
	})

	bgCtx, cancel := context.WithCancel(context.Background())
	taskID := 123
	files := []domain.File{
		{ID: 1, URL: "https://123.com/1"},
		{ID: 2, URL: "https://123.com/2"},
		{ID: 3, URL: "https://123.com/3"},
	}
	for _, f := range files {
		newMock.EXPECT().
			SaveFile(gomock.Any(), taskID, f.ID, "", []byte("data")).Times(1)
	}
	newMock.EXPECT().UpdateStatus(gomock.Any(), taskID, "DONE").Times(1)
	svc.runBackgroundProcess(bgCtx, cancel, taskID, files)
}

func TestDownloadService_StartDownload(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    repoMock := mocks.NewMockRepository(ctrl)
    urls := []string{"https://123.com/one", "https://123.com/two"}
    timeout := 200 * time.Millisecond
    expectedID := 777 // произвольный id, который вернёт мок
    repoMock.EXPECT().
        TaskCreation(gomock.Any(), gomock.AssignableToTypeOf(&domain.DownloadTask{})).
        DoAndReturn(func(_ context.Context, task *domain.DownloadTask) (int, error) {
            assert.Equal(t, "PROCESS", task.Status)
            assert.Len(t, task.Files, len(urls))
            for i, f := range task.Files {
                assert.Equal(t, urls[i], f.URL)
                assert.Equal(t, 0, f.ID)
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
    http.DefaultClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
        return &http.Response{
            StatusCode: http.StatusOK,
            Body:       io.NopCloser(bytes.NewBufferString("file content")),
            Header:     make(http.Header),
        }, nil
    })}

    svc := NewDownloadService(repoMock)
    id, err := svc.StartDownload(context.Background(), urls, timeout)
    assert.NoError(t, err)
    assert.Equal(t, expectedID, id)
    timeoutWait := time.After(2 * time.Second)
    for i := 0; i < len(urls)+1; i++ {
        select {
        case <-done:
        case <-timeoutWait:
            t.Fatalf("timeout gone")
        }
    }
}

