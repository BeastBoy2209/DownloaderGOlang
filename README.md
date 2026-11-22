# SIS Checkpoint 2 — URL Analyzer (concurrent)

[Requirements and submission checklist](CHECKLIST.md)

A Go application for managing URL analysis tasks, now processed concurrently by a worker pool.
The input is a URL string. The result contains the scheme, host with port, path,
and query string. The application makes no network requests and uses only the
standard library.

## Running

Go 1.27 is required.

```bash
go run ./cmd/app 'https://example.com:8080/docs?q=go' 'http://example.org'
```

Each argument becomes a task: it is created, submitted to the pool, processed
by a worker, and printed after the pool stops. A URL must use `http`/`https`
and have a host; otherwise the program exits with code 1. Data lives in memory.

## Concurrency design

```text
Create -> Submit -> [ bounded queue (chan int) ] -> Worker 1..N -> repo.Update
                                                                    ^
                     Get (any goroutine) ---------- RWMutex -------+
```

- **Workers:** fixed number, at least 2 (`NewPool` returns `ErrInvalidPool` otherwise). The CLI uses 3.
  Workers are long-lived: `for id := range queue`.
- **Queue:** buffered `chan int` (task IDs) with a fixed capacity (CLI: one slot per argument).
- **Full queue:** `Submit` never blocks. It returns `ErrQueueFull`, and the task goes back to `Pending`
  so it can be submitted again later.
- **Submit** returns right after the task is queued; it does not wait for processing.
- **Processing order** is not guaranteed; nothing depends on it.

### Task lifecycle

```text
Pending -> Queued -> Processing -> Completed
              |            \-> Failed
              \-> Pending (rollback if queue is full)
```

Allowed transitions are enforced in `domain.Task` (`Enqueue`, `Dequeue`, `Start`, `Finish`).
Anything else returns `ErrInvalidState`: submitting a Queued, Processing, Completed
or Failed task is an error (no retries). The CP1 synchronous `Service.Process`
(Pending -> Completed) still works and goes through the same atomic update.
Unknown ID -> `ErrNotFound`.

### Synchronization

- `repository.Memory` guards the map and the ID counter with one `sync.RWMutex`; `Get` uses `RLock`.
- `Memory.Update(id, fn)` is an atomic read-modify-write under the write lock. All state
  transitions go through it, so two workers (or two `Submit` calls) can never claim the same task:
  only one `Queued -> Processing` / `Pending -> Queued` succeeds.
- Tasks are stored by value and returned as copies, so no shared mutable fields leak out.
- The actual URL analysis runs **outside** the lock (`Analyze` is pure); the lock is held only
  for the short status/result writes.
- `Pool.mu` (RWMutex) + `closed` flag: `Submit` holds `RLock` during its non-blocking send,
  `Stop` takes `Lock` before `close(queue)`. So a send on a closed channel is impossible and
  `Submit` cannot block forever on a stopped pool. Only `Pool` creates and closes the channel.

### Stopping

`Pool.Stop()` marks the pool closed (new `Submit` -> `ErrPoolStopped`, task stays untouched),
closes the queue, and waits on a `sync.WaitGroup`. Workers drain what is already in the queue,
so accepted tasks are not lost, then exit. Calling `Stop` again is a safe no-op.

## Structure

- `internal/domain` — task, states, transitions, pure `Analyze`.
- `internal/usecase` — `Service` (create/get/sync process) and `Pool` (queue + workers).
- `internal/repository` — thread-safe in-memory storage.
- `cmd/app` — CLI wiring.

Errors are declared with `errors.New` and checked with `errors.Is`.

## Verification

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
```

Tests (standard `testing`) cover the CP1 behavior plus: pool startup and config validation,
async submit, many tasks with correct per-task results, real overlap of two tasks
(workers block on a channel gate until both have started), worker limit, duplicate/invalid
submits, full queue and recovery, failed processing, concurrent create/submit/get,
single processing of a task submitted from many goroutines, `Stop` draining accepted tasks,
rejection after stop, double `Stop`, goroutine leak check, and `Submit` racing with `Stop`.
