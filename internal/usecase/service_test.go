package usecase_test // _test‑пакет → тестируем внешний API

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"downloader/internal/domain"
	"downloader/internal/mocks"
	"downloader/internal/usecase"
)

func TestDownloadService_GetDownload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockRepository(ctrl)
	expectedTask := domain.DownloadTask{
		ID:     42,
		Status: "finished",
		Files: []domain.File{
			{ID: 1, URL: "https://example.com/file1"},
		},
	}
	mockRepo.EXPECT().
		RecieveDownload(gomock.Any(), int(42)). 
		Return(expectedTask, nil).
		Times(1) 

	svc := usecase.NewDownloadService(mockRepo)
	gotTask, err := svc.GetDownload(context.Background(), 42)

	assert.NoError(t, err, "ошибка не ожидается")
	assert.Equal(t, expectedTask, gotTask, "полученная задача должна совпадать с тем, что вернул мок")
}

func TestDownloadService_GetFileContent(t *testing.T) {
	type testCase struct {
		name        string
		taskID      int
		fileID      int
		
		mockSetup   func(repo *mocks.MockRepository)
		wantData    []byte
		wantErr     error
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

			svc := usecase.NewDownloadService(mockRepo)
			data, err := svc.GetFileContent(context.Background(), tc.taskID, tc.fileID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr,
					"expected error %v, got %v", tc.wantErr, err)
			} else {
				assert.NoError(t, err, "unexpected error")
			}
			
			assert.Equal(t, tc.wantData, data,
				"returned file bytes do not match expected")
		})
	}
}


// func TestDownloadService_<MethodName>(t *testing.T) {
//     ctrl := gomock.NewController(t)
//     defer ctrl.Finish()

//     mockRepo := mocks.NewMockRepository(ctrl)

//     // Ожидания (можно вынести в локальную функцию)
//     //   mockRepo.EXPECT()...   // любой набор EXPECT

//     svc := usecase.NewDownloadService(mockRepo)

//     // Вызов
//     //   got, err := svc.<MethodName>(ctx, …)

//     // Assertions
//     //   assert.NoError / assert.ErrorIs
//     //   assert.Equal / assert.Empty
// }