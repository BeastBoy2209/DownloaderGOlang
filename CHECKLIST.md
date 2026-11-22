пайплайн
-Аргументы терминала
-Создание задачи
-Присвоение ID и сохранение в map — Pending
-Получение задачи по ID
-Проверка состояния
-Processing → разбор URL → Completed
-Сохранение обновлённой задачи
-Вывод результата

# SIS Checkpoint 1 — Checklist

Source: requirements from `sis_checkpoint_1.md`.
`[x]` means confirmed by code or verification; `[ ]` requires student action or confirmation.
The maximum score is 5 points: 2.5 for code and 2.5 for the mandatory oral defense.

## 1. Concept and restrictions

- [x] The application manages computational tasks: input → processing → result, rather than a to-do list.
- [x] One of the allowed domains is selected: URL analysis.
- [x] A URL string is parsed into its components: scheme, host, path, and query.
- [x] No goroutines or channels are used.
- [x] No database is used.
- [x] No external API calls or other network requests are made.
- [x] No web frameworks are used; all packages come from the standard library.

Implementation: [task model and URL analysis](internal/domain/models.go), [CLI](cmd/app/main.go), [go.mod](go.mod).

## 2. Functional requirements

### Task model

- [x] Unique ID: `Task.ID`, assigned by the storage counter.
- [x] Input data: `Task.Input`, a URL string.
- [x] Current state: `Task.Status`.
- [x] Processing result: `Task.Result`.

Implementation: [Task](internal/domain/models.go), [Memory.Create](internal/repository/memory.go).

### Task lifecycle

- [x] A newly created, unprocessed task has the `Pending` state.
- [x] A task has the `Processing` state during processing.
- [x] A successfully processed task has the `Completed` state.
- [x] The allowed sequence is `Pending → Processing → Completed`.
- [x] Transitions are controlled by `Task.Process`.
- [x] Processing a completed task again returns `ErrInvalidState`.
- [x] Processing from `Processing` or an unknown state is also rejected without changing the task.
- [x] The lifecycle and allowed transitions are documented in the README.

Processing is synchronous: `Processing` is an intermediate state within the call.
The returned task and the saved task already have the `Completed` state.

Evidence: [Task.Process](internal/domain/models.go), [transition tests](internal/domain/models_test.go), [README](README.md#task-lifecycle).

### System behavior

- [x] A new task is created with input data and a unique ID.
- [x] Input is validated on creation: `http` or `https` scheme, a nonempty hostname, and valid URL syntax.
- [x] Invalid input is rejected by returning a standard `error`.
- [x] Tasks are stored in memory using `map[int]domain.Task`.
- [x] An existing task can be retrieved by ID.
- [x] A missing ID returns `ErrNotFound`.
- [x] Business logic is meaningful and deterministic: the same URL produces the same components without network access.
- [x] The result is saved inside the corresponding task, and its status is updated.
- [x] Multiple tasks are stored and processed independently without overwriting or mixing their data.

Task independence is tested with sequential processing; concurrent execution is not used in checkpoint 1.

Evidence: [service](internal/usecase/service.go), [storage](internal/repository/memory.go), [service tests](internal/usecase/service_test.go).

## 3. Go technical requirements

- [x] The domain is modeled with `Task`, `Result`, and their methods.
- [x] The pointer receiver `(*Task).Process` changes the status and result.
- [x] The value receiver `(Task).Validate` validates the task without modifying it.
- [x] A meaningful `Repository` interface keeps the usecase independent of a specific storage implementation.
- [x] A `map` collection is used to store tasks.
- [x] Errors are explicitly returned through `error`.
- [x] Ordinary application errors are not handled with `panic`.
- [x] Code is organized into domain, usecase, repository, and an entry point.
- [x] The architecture allows HTTP handlers and workers to be added on top of the service in later checkpoints.

Storage will need synchronization before concurrent processing is introduced, as documented in the README.

Evidence: [domain](internal/domain/models.go), [errors](internal/domain/errors.go), [usecase and interface](internal/usecase/service.go), [repository](internal/repository/memory.go).

## 4. Unit tests

- [x] Tests use the standard `testing` package.
- [x] Table-driven tests are included: `TestCreateInvalidInput`, `TestProcess`, and `TestProcessInvalidState`.

All nine required scenarios:

- [x] 1. Successful task creation: `TestCreate`.
- [x] 2. Invalid input error: `TestCreateInvalidInput`.
- [x] 3. Successful retrieval by ID: `TestGet`.
- [x] 4. Error when retrieving a missing task: `TestGetMissing`.
- [x] 5. Successful task processing: `TestProcess`.
- [x] 6. Result correctness: all `Result` fields are compared in `TestProcess`.
- [x] 7. `Completed` status after processing and task persistence: `TestProcess`.
- [x] 8. An invalid operation or transition: `TestProcessCompleted` and `TestProcessInvalidState`.
- [x] 9. Multiple independent tasks: `TestIndependentTasks`.

Additional checks:

- [x] Processing a missing task: `TestProcessMissing`.
- [x] Invalid input leaves the task unchanged: `TestProcessInvalidInput`.
- [x] Storage returns independent copies: `TestMemoryReturnsCopies`.
- [x] Saving a missing task returns an error without creating a record: `TestSaveMissing`.

Tests: [usecase](internal/usecase/service_test.go), [domain](internal/domain/models_test.go), [repository](internal/repository/memory_test.go).

## 5. Verification and submission

The checks below were completed on October 18, 2026. Repeat them after code changes.

- [x] The root directory contains a valid `go.mod` declaring `go 1.27.0`.
- [x] `gofmt -l .` reports no unformatted files.
- [x] `go vet ./...` completes without warnings.
- [x] `go test ./...` passes all tests.
- [x] The README documents states and transitions.
- [ ] The version from the `checkpoint1` branch is uploaded to a personal GitHub repository.
- [ ] The repository link and a specific commit or tag are submitted to Microsoft Teams before the deadline.

Verification commands:

```bash
gofmt -l .
go vet ./...
go test ./...
```

## 8. URLs for the defense demo
### Basic URL

[Go documentation](https://go.dev/doc/)

```text
https://go.dev/doc/
```

Expected: scheme `https`, host `go.dev`, path `/doc/`, empty query.

### URL with a query string

[net/url documentation](https://pkg.go.dev/net/url?tab=doc)

```text
https://pkg.go.dev/net/url?tab=doc
```

Expected: host `pkg.go.dev`, path `/net/url`, query `tab=doc`.

### Encoded path and fragment

[Go on Wikipedia](https://en.wikipedia.org/wiki/Go_%28programming_language%29#History)

```text
https://en.wikipedia.org/wiki/Go_%28programming_language%29#History
```

Expected: path `/wiki/Go_(programming_language)`, empty query.
The encoded parentheses are decoded, and the fragment `#History` is excluded from the result.

### Process all three URLs

Run this command from the repository root. Keep the single quotes around each URL.

```bash
go run ./cmd/app \
  'https://go.dev/doc/' \
  'https://pkg.go.dev/net/url?tab=doc' \
  'https://en.wikipedia.org/wiki/Go_%28programming_language%29#History'
```

Expected: three independent tasks with IDs `1`, `2`, and `3`, all `Completed`.
IDs restart from `1` on each new program run because storage is in memory.
