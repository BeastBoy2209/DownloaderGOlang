beasty@Mac DownloaderGOlang % golangci-lint run
internal/usecase/service.go:115:39: Non-inherited new context, use function like `context.WithXXX` instead (contextcheck)
        if err := d.repo.UpdateDownloadStatus(context.Background(), taskID, "DONE"); err != nil {
                                             ^
internal/repository/postgres.go:20:1: calculated cyclomatic complexity for function CreateDownloadAndFiles is 16, max is 15 (cyclop)
func (r *PostgresRepo) CreateDownloadAndFiles(ctx context.Context, task *domain.DownloadTask) (int, error) {
^
internal/transport/handler.go:26:40: string `error` has 3 occurrences, make it a constant (goconst)
                return c.JSON(400, map[string]string{"error": err.Error()})
                                                     ^
cmd/app/main.go:47:3: exitAfterDefer: os.Exit will exit, and `defer cancel()` will not run (gocritic)
                os.Exit(1)
                ^
internal/repository/postgres.go:94:1: paramTypeCombine: func(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error could be replaced with func(ctx context.Context, taskID, fileID int, errCode string, content []byte) error (gocritic)
func (r *PostgresRepo) UpdateFile(ctx context.Context, taskID int, fileID int, errCode string, content []byte) error {
^
internal/repository/postgres.go:141:1: paramTypeCombine: func(ctx context.Context, taskID int, fileID int) ([]byte, error) could be replaced with func(ctx context.Context, taskID, fileID int) ([]byte, error) (gocritic)
func (r *PostgresRepo) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
^
internal/usecase/service.go:39:1: paramTypeCombine: func(ctx context.Context, taskID int, fileID int) ([]byte, error) could be replaced with func(ctx context.Context, taskID, fileID int) ([]byte, error) (gocritic)
func (d *DownloadService) GetFileContent(ctx context.Context, taskID int, fileID int) ([]byte, error) {
^
internal/usecase/service.go:49:14: httpNoBody: http.NoBody should be preferred to the nil request body (gocritic)
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
                    ^
cmd/app/main.go:4:1: File is not properly formatted (gofumpt)
        "context"
^
internal/usecase/service.go:10:1: File is not properly formatted (gofumpt)

^
internal/usecase/service_test.go:5:1: File is not properly formatted (gofumpt)
        "context"
^
cmd/app/main.go:60:13: G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server (gosec)
        server := &http.Server{
                Addr:    addr,
                Handler: e,
        }
internal/usecase/service.go:140:2: G118: Goroutine uses context.Background/TODO while request-scoped context is available (gosec)
        go d.runBackgroundProcess(bgCtx, cancel, id, task.Files)
        ^
internal/usecase/service_test.go:298:3: for loop can be changed to use an integer range (Go 1.22+) (intrange)
                for i := 0; i < len(urls)+1; i++ {
                ^
internal/repository/postgres.go:76:1: The line is 131 characters long, which exceeds the maximum of 100 characters. (lll)
                        return 0, fmt.Errorf("create file for download %d: file id was not returned: %w", taskID, domain.ErrServer)
^
internal/repository/postgres.go:82:1: The line is 112 characters long, which exceeds the maximum of 100 characters. (lll)
                        return 0, fmt.Errorf("scan created file id for download %d file %d: %w", taskID, i, err)
^
internal/repository/postgres.go:117:1: The line is 111 characters long, which exceeds the maximum of 100 characters. (lll)
                return fmt.Errorf("execute file update query for download %d file %d: %w", taskID, fileID, err)
^
internal/repository/postgres.go:123:1: The line is 102 characters long, which exceeds the maximum of 100 characters. (lll)
func (r *PostgresRepo) UpdateDownloadStatus(ctx context.Context, taskID int, newStatus string) error {
^
internal/repository/postgres.go:135:1: The line is 106 characters long, which exceeds the maximum of 100 characters. (lll)
                return fmt.Errorf("execute download status update query for download %d: %w", taskID, err)
^
internal/repository/postgres.go:160:1: The line is 126 characters long, which exceeds the maximum of 100 characters. (lll)
                return nil, fmt.Errorf("file content for download %d file %d not found: %w", taskID, fileID, domain.ErrClient)
^
internal/repository/postgres.go:191:1: The line is 115 characters long, which exceeds the maximum of 100 characters. (lll)
                return domain.DownloadTask{}, fmt.Errorf("execute download query for download %d: %w", taskID, err)
^
internal/repository/postgres.go:198:1: The line is 111 characters long, which exceeds the maximum of 100 characters. (lll)
                return domain.DownloadTask{}, fmt.Errorf("download %d not found: %w", taskID, domain.ErrClient)
^
internal/usecase/service.go:52:1: The line is 101 characters long, which exceeds the maximum of 100 characters. (lll)
                if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
^
internal/usecase/service.go:53:1: The line is 126 characters long, which exceeds the maximum of 100 characters. (lll)
                        log.Printf("task %d file %d (%s): failed to persist failure state: %v", taskID, file.ID, url, saveErr)
^
internal/usecase/service.go:58:1: The line is 151 characters long, which exceeds the maximum of 100 characters. (lll)
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
^
internal/usecase/service.go:63:1: The line is 101 characters long, which exceeds the maximum of 100 characters. (lll)
                if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
^
internal/usecase/service.go:64:1: The line is 126 characters long, which exceeds the maximum of 100 characters. (lll)
                        log.Printf("task %d file %d (%s): failed to persist failure state: %v", taskID, file.ID, url, saveErr)
^
internal/usecase/service.go:74:1: The line is 101 characters long, which exceeds the maximum of 100 characters. (lll)
                if saveErr := d.repo.UpdateFile(ctx, taskID, file.ID, "ERROR", nil); saveErr != nil {
^
internal/usecase/service.go:100:1: The line is 129 characters long, which exceeds the maximum of 100 characters. (lll)
func (d *DownloadService) runBackgroundProcess(ctx context.Context, cancel context.CancelFunc, taskID int, files []domain.File) {
^
internal/usecase/service_test.go:128:1: The line is 123 characters long, which exceeds the maximum of 100 characters. (lll)
                        httpClient: newMockHTTPClient(func(req *http.Request) (*http.Response, error) { return nil, nil }),
^
internal/usecase/service_test.go:130:1: The line is 110 characters long, which exceeds the maximum of 100 characters. (lll)
                                repo.EXPECT().UpdateFile(gomock.Any(), gomock.Any(), 1, "ERROR", nil).Times(1)
^
internal/usecase/service_test.go:141:1: The line is 110 characters long, which exceeds the maximum of 100 characters. (lll)
                                repo.EXPECT().UpdateFile(gomock.Any(), gomock.Any(), 2, "ERROR", nil).Times(1)
^
internal/usecase/service_test.go:155:1: The line is 110 characters long, which exceeds the maximum of 100 characters. (lll)
                                repo.EXPECT().UpdateFile(gomock.Any(), gomock.Any(), 3, "ERROR", nil).Times(1)
^
internal/usecase/service_test.go:193:1: The line is 105 characters long, which exceeds the maximum of 100 characters. (lll)
                                        Body:       io.NopCloser(bytes.NewReader([]byte("hello world"))),
^
internal/usecase/service_test.go:197:1: The line is 123 characters long, which exceeds the maximum of 100 characters. (lll)
                                repo.EXPECT().UpdateFile(gomock.Any(), gomock.Any(), 6, "", []byte("hello world")).Times(1)
^
internal/usecase/service_test.go:260:1: The line is 112 characters long, which exceeds the maximum of 100 characters. (lll)
                        CreateDownloadAndFiles(gomock.Any(), gomock.AssignableToTypeOf(&domain.DownloadTask{})).
^
internal/usecase/service_test.go:275:1: The line is 111 characters long, which exceeds the maximum of 100 characters. (lll)
                                UpdateFile(gomock.Any(), expectedID, gomock.Any(), gomock.Any(), gomock.Any()).
^
cmd/app/main.go:41:66: Magic number: 15, in <argument> detected (mnd)
        startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
                                                                        ^
cmd/app/main.go:78:79: Magic number: 15, in <argument> detected (mnd)
        ctx, cancelShutdown := context.WithTimeout(context.Background(), time.Second*15)
                                                                                     ^
internal/transport/handler.go:29:17: Magic number: 422, in <argument> detected (mnd)
                return c.JSON(422, map[string]string{"error": err.Error()})
                              ^
internal/transport/handler.go:32:16: Magic number: 500, in <argument> detected (mnd)
        return c.JSON(500, map[string]string{"error": "internal server error"})
                      ^
internal/transport/handler.go:61:16: Magic number: 200, in <argument> detected (mnd)
        return c.JSON(200, map[string]any{
                      ^
internal/transport/handler.go:79:16: Magic number: 200, in <argument> detected (mnd)
        return c.JSON(200, task)
                      ^
internal/transport/handler.go:95:16: Magic number: 200, in <argument> detected (mnd)
        return c.Blob(200, "application/octet-stream", content)
                      ^
internal/usecase/service.go:104:14: Magic number: 10, in <argument> detected (mnd)
        eg.SetLimit(10)
                    ^
internal/config/config.go:35:2: return with no blank line before (nlreturn)
        return &cfg
        ^
internal/repository/postgres.go:91:2: return with no blank line before (nlreturn)
        return taskID, nil
        ^
internal/transport/handler.go:103:2: return with no blank line before (nlreturn)
        return e
        ^
internal/config/config.go:28:5: avoid inline error handling using `if err := ...; err != nil`; use plain assignment `err := ...` (noinlineerr)
        if err := godotenv.Load(); err != nil {
           ^
internal/config/config.go:32:5: avoid inline error handling using `if err := ...; err != nil`; use plain assignment `err := ...` (noinlineerr)
        if err := env.Parse(&cfg); err != nil {
           ^
internal/repository/postgres.go:51:5: avoid inline error handling using `if err := ...; err != nil`; use plain assignment `err := ...` (noinlineerr)
        if err := rows.Scan(&taskID); err != nil {
           ^
internal/usecase/service_test.go:39:1: Function TestDownloadService_GetDownload missing the call to method parallel (paralleltest)
func TestDownloadService_GetDownload(t *testing.T) {
^
internal/usecase/service_test.go:61:1: Function TestDownloadService_GetFileContent missing the call to method parallel (paralleltest)
func TestDownloadService_GetFileContent(t *testing.T) {
^
internal/usecase/service_test.go:97:2: Range statement for test TestDownloadService_GetFileContent missing the call to method parallel in test Run (paralleltest)
        for _, tc := range cases {
        ^
internal/usecase/service_test.go:117:1: Function TestDownloadService_downloadSingleFile missing the call to method parallel (paralleltest)
func TestDownloadService_downloadSingleFile(t *testing.T) {
^
internal/usecase/service_test.go:203:2: Range statement for test TestDownloadService_downloadSingleFile missing the call to method parallel in test Run (paralleltest)
        for _, tc := range cases {
        ^
internal/usecase/service_test.go:226:1: Function TestDownloadService_runBackgroundProcess missing the call to method parallel (paralleltest)
func TestDownloadService_runBackgroundProcess(t *testing.T) {
^
internal/usecase/service_test.go:249:1: Function TestDownloadService_StartDownload missing the call to method parallel (paralleltest)
func TestDownloadService_StartDownload(t *testing.T) {
^
internal/usecase/service_test.go:250:2: Function TestDownloadService_StartDownload missing the call to method parallel in the test run (paralleltest)
        t.Run("success", func(t *testing.T) {
        ^
internal/usecase/service_test.go:307:2: Function TestDownloadService_StartDownload missing the call to method parallel in the test run (paralleltest)
        t.Run("empty urls returns business error", func(t *testing.T) {
        ^
cmd/app/main.go:66:13: message should be lowercased (sloglint)
                slog.Info("Server started", slog.Int("port", int(cfg.Server.Port)))
                          ^
cmd/app/main.go:77:12: message should be lowercased (sloglint)
        slog.Info("Starting shutdown...")
                  ^
internal/usecase/service_test.go:108:5: require-error: for error assertions use require (testifylint)
                                assert.ErrorIs(t, err, tc.wantErr)
                                ^
internal/usecase/service_test.go:110:5: require-error: for error assertions use require (testifylint)
                                assert.NoError(t, err)
                                ^
internal/usecase/service_test.go:294:3: require-error: for error assertions use require (testifylint)
                assert.NoError(t, err)
                ^
65 issues:
* contextcheck: 1
* cyclop: 1
* goconst: 1
* gocritic: 5
* gofumpt: 3
* gosec: 2
* intrange: 1
* lll: 23
* mnd: 8
* nlreturn: 3
* noinlineerr: 3
* paralleltest: 9
* sloglint: 2
* testifylint: 3