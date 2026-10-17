# SIS Checkpoint 1 — URL Analyzer

[Requirements and submission checklist](CHECKLIST.md)

A synchronous Go application for managing URL analysis tasks.
The input is a URL string. The result contains the scheme, host with port, path,
and query string. The application makes no network requests and uses only the
standard library.

## Running

Go 1.27 is required by the checkpoint specification.

```bash
go run ./cmd/app 'https://example.com:8080/docs?q=go' 'http://example.org'
```

Each argument creates a separate task. Tasks are processed sequentially, and
results are printed to the terminal. On error, the program exits with code 1.
Data stays in memory until the program exits.

A URL must use the `http` or `https` scheme and have a nonempty hostname.
Empty input, relative addresses, and malformed URLs are rejected with an `error`.
`Path` contains the decoded path; `Query` contains the original query string
without `?`. The fragment after `#` is not included in the result.

## Task lifecycle

```text
Pending → Processing → Completed
```

`Create` validates the input and saves the task as `Pending`.
`Process` is allowed only for `Pending` tasks: it changes the status to
`Processing`, computes the result, and sets `Completed`. Processing is
synchronous, so the caller receives a completed task.
Repeated processing, processing from `Processing`, and processing from an
unknown state return `ErrInvalidState` without changing the task.
Invalid input also leaves the task unchanged. A missing ID returns `ErrNotFound`.

## Structure

- `internal/domain` — task, result, states, validation, and URL analysis.
- `internal/usecase` — creation, retrieval, processing, and the `Repository` interface.
- `internal/repository` — storage using `map[int]domain.Task` and an ID counter.
- `cmd/app` — command-line arguments and dependency setup.

The domain layer does not depend on the other layers. The usecase layer depends
on the domain and a repository interface; the implementation is passed through
the constructor. Future HTTP handlers or workers can call the same service.
The current storage supports sequential access; synchronization must be added
before introducing goroutines.

`Task.Validate` uses a value receiver because it does not modify the task.
`Task.Process` uses a pointer receiver to change the status and result.
Storage returns copies of structs: tasks contain no slices, maps, or pointers,
so modifying a returned copy does not change the stored task.
After processing, the service explicitly saves the updated task through `Save`.

Errors are declared with `errors.New` and can be checked with `errors.Is`.

## Verification

```bash
gofmt -l .
go vet ./...
go test ./...
```

Tests use the standard `testing` package and include table-driven cases.
They cover creation, validation, retrieval, missing IDs, processing, exact
results, persisted status, invalid transitions, independent tasks, and copies
returned by storage.
