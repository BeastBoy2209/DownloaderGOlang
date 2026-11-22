package usecase_test

import (
	"downloader/internal/domain"
	"downloader/internal/usecase"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

const timeout = 5 * time.Second

func newPool(t *testing.T, workers, capacity int, opts ...usecase.Option) (*usecase.Service, *usecase.Pool) {
	t.Helper()
	service := newService()
	pool, err := usecase.NewPool(service, workers, capacity, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Stop)
	return service, pool
}

// waitStatus ждёт финального статуса без sleep: поллим с Gosched и общим дедлайном.
func waitStatus(t *testing.T, s *usecase.Service, id int, want domain.Status) domain.Task {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task, err := s.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if task.Status == want {
			return task
		}
		runtime.Gosched()
	}
	t.Fatalf("task %d did not reach %s", id, want)
	return domain.Task{}
}

// blockingAnalyzer стоит на gate, пока тест не откроет, и сообщает о старте через started.
func blockingAnalyzer(started chan<- string, gate <-chan struct{}) usecase.Analyzer {
	return func(input string) (domain.Result, error) {
		started <- input
		<-gate
		return domain.Analyze(input)
	}
}

// gateRelease отпускает заблокированных воркеров ровно один раз; в Cleanup,
// чтобы упавший тест не вешал Stop. Вызывать после newPool (cleanup идут LIFO).
func gateRelease(t *testing.T, gate chan struct{}) func() {
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	t.Cleanup(release)
	return release
}

func TestPoolStartup(t *testing.T) {
	service := newService()
	for _, c := range []struct{ workers, capacity int }{{0, 1}, {1, 1}, {2, 0}, {2, -1}} {
		if _, err := usecase.NewPool(service, c.workers, c.capacity); !errors.Is(err, domain.ErrInvalidPool) {
			t.Fatalf("workers=%d cap=%d: got %v, want ErrInvalidPool", c.workers, c.capacity, err)
		}
	}
	_, pool := newPool(t, 2, 1)
	if pool == nil {
		t.Fatal("pool not started")
	}
}

func TestSubmitAndProcessMany(t *testing.T) {
	service, pool := newPool(t, 3, 20)
	want := map[int]string{}
	for i := 0; i < 20; i++ {
		host := "host" + string(rune('a'+i)) + ".example"
		task := createTask(t, service, "https://"+host+"/p")
		want[task.ID] = host
		if err := pool.Submit(task.ID); err != nil {
			t.Fatal(err)
		}
	}
	pool.Stop()
	// результат каждой задачи должен лежать в ней самой, а не в чужой
	for id, host := range want {
		got, err := service.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.Completed || got.Result.Host != host || got.Result.Path != "/p" || got.Result.Scheme != "https" {
			t.Fatalf("task %d: %+v, want host %s", id, got, host)
		}
	}
}

func TestSubmitDoesNotWait(t *testing.T) {
	started := make(chan string, 1)
	gate := make(chan struct{})
	service, pool := newPool(t, 2, 1, usecase.WithAnalyzer(blockingAnalyzer(started, gate)))
	release := gateRelease(t, gate)
	task := createTask(t, service, "https://example.com")

	// анализатор заблокирован, но Submit обязан вернуться
	if err := pool.Submit(task.ID); err != nil {
		t.Fatal(err)
	}
	<-started
	if got := waitStatus(t, service, task.ID, domain.Processing); got.Result != (domain.Result{}) {
		t.Fatalf("result set before finish: %+v", got)
	}
	release()
	waitStatus(t, service, task.ID, domain.Completed)
}

func TestConcurrentProcessing(t *testing.T) {
	started := make(chan string, 2)
	gate := make(chan struct{})
	service, pool := newPool(t, 2, 2, usecase.WithAnalyzer(blockingAnalyzer(started, gate)))
	release := gateRelease(t, gate)
	a := createTask(t, service, "https://a.example")
	b := createTask(t, service, "https://b.example")
	for _, id := range []int{a.ID, b.ID} {
		if err := pool.Submit(id); err != nil {
			t.Fatal(err)
		}
	}
	// обе задачи стартуют, пока ни одна не завершилась = реальное перекрытие
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(timeout):
			t.Fatal("tasks are not processed concurrently")
		}
	}
	waitStatus(t, service, a.ID, domain.Processing)
	waitStatus(t, service, b.ID, domain.Processing)
	release()
	waitStatus(t, service, a.ID, domain.Completed)
	waitStatus(t, service, b.ID, domain.Completed)
}

func TestWorkerLimit(t *testing.T) {
	started := make(chan string, 4)
	gate := make(chan struct{})
	service, pool := newPool(t, 2, 4, usecase.WithAnalyzer(blockingAnalyzer(started, gate)))
	release := gateRelease(t, gate)
	ids := make([]int, 4)
	for i := range ids {
		ids[i] = createTask(t, service, "https://example.com").ID
		if err := pool.Submit(ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	<-started
	<-started
	// два воркера заняты, остальные две задачи обязаны оставаться в Queued
	queued := 0
	for _, id := range ids {
		task, _ := service.Get(id)
		if task.Status == domain.Queued {
			queued++
		}
	}
	if queued != 2 {
		t.Fatalf("got %d queued, want 2", queued)
	}
	release()
	pool.Stop()
	for _, id := range ids {
		if task, _ := service.Get(id); task.Status != domain.Completed {
			t.Fatalf("task %d: %s", id, task.Status)
		}
	}
}

func TestSubmitInvalidAndDuplicate(t *testing.T) {
	started := make(chan string, 1)
	gate := make(chan struct{})
	service, pool := newPool(t, 2, 2, usecase.WithAnalyzer(blockingAnalyzer(started, gate)))
	release := gateRelease(t, gate)

	if err := pool.Submit(999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	task := createTask(t, service, "https://example.com")
	if err := pool.Submit(task.ID); err != nil {
		t.Fatal(err)
	}
	// Queued/Processing повторно отправлять нельзя
	if err := pool.Submit(task.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("duplicate: got %v, want ErrInvalidState", err)
	}
	<-started
	if err := pool.Submit(task.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("processing: got %v, want ErrInvalidState", err)
	}
	// синхронный Process на чужой задаче тоже не пройдёт
	if _, err := service.Process(task.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("sync process: got %v, want ErrInvalidState", err)
	}
	release()
	waitStatus(t, service, task.ID, domain.Completed)
	if err := pool.Submit(task.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("completed: got %v, want ErrInvalidState", err)
	}
}

func TestQueueFull(t *testing.T) {
	started := make(chan string, 4) // буфер, чтобы воркеры не вставали на отправке в started
	gate := make(chan struct{})
	// 2 воркера, очередь на 1
	service, pool := newPool(t, 2, 1, usecase.WithAnalyzer(blockingAnalyzer(started, gate)))
	release := gateRelease(t, gate)
	busy := []int{createTask(t, service, "https://w1.example").ID, createTask(t, service, "https://w2.example").ID}
	// после каждого submit ждём Processing: пока воркер не забрал id, он ещё занимает слот очереди
	for _, id := range busy {
		if err := pool.Submit(id); err != nil {
			t.Fatal(err)
		}
		waitStatus(t, service, id, domain.Processing)
	}
	queued := createTask(t, service, "https://q.example")
	if err := pool.Submit(queued.ID); err != nil {
		t.Fatal(err)
	}
	extra := createTask(t, service, "https://extra.example")
	if err := pool.Submit(extra.ID); !errors.Is(err, domain.ErrQueueFull) {
		t.Fatalf("got %v, want ErrQueueFull", err)
	}
	// отклонённая задача осталась Pending и её можно отправить позже
	if got, _ := service.Get(extra.ID); got.Status != domain.Pending {
		t.Fatalf("rejected task is %s, want Pending", got.Status)
	}
	release()
	waitStatus(t, service, queued.ID, domain.Completed)
	if err := pool.Submit(extra.ID); err != nil {
		t.Fatalf("resubmit after space freed: %v", err)
	}
	waitStatus(t, service, extra.ID, domain.Completed)
}

func TestFailedProcessing(t *testing.T) {
	boom := errors.New("boom")
	service, pool := newPool(t, 2, 2, usecase.WithAnalyzer(func(string) (domain.Result, error) {
		return domain.Result{Host: "garbage"}, boom
	}))
	task := createTask(t, service, "https://example.com")
	if err := pool.Submit(task.ID); err != nil {
		t.Fatal(err)
	}
	got := waitStatus(t, service, task.ID, domain.Failed)
	if got.Result != (domain.Result{}) {
		t.Fatalf("failed task kept result: %+v", got.Result)
	}
}

func TestConcurrentAccess(t *testing.T) {
	service, pool := newPool(t, 4, 64)
	var wg sync.WaitGroup
	ids := make(chan int, 64)
	// 8 горутин одновременно создают, сабмитят и читают
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
				task, err := service.Create("https://example.com/x")
				if err != nil {
					t.Error(err)
					return
				}
				if err := pool.Submit(task.ID); err != nil {
					t.Error(err)
					return
				}
				if _, err := service.Get(task.ID); err != nil {
					t.Error(err)
				}
				ids <- task.ID
			}
		}()
	}
	wg.Wait()
	close(ids)
	pool.Stop()
	seen := map[int]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
		if task, _ := service.Get(id); task.Status != domain.Completed {
			t.Fatalf("task %d: %s", id, task.Status)
		}
	}
	if len(seen) != 64 {
		t.Fatalf("got %d tasks, want 64", len(seen))
	}
}

func TestSameTaskSubmittedConcurrently(t *testing.T) {
	var mu sync.Mutex
	runs := 0
	service, pool := newPool(t, 4, 16, usecase.WithAnalyzer(func(in string) (domain.Result, error) {
		mu.Lock()
		runs++
		mu.Unlock()
		return domain.Analyze(in)
	}))
	task := createTask(t, service, "https://example.com")
	var wg sync.WaitGroup
	var okMu sync.Mutex
	accepted := 0
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if pool.Submit(task.ID) == nil {
				okMu.Lock()
				accepted++
				okMu.Unlock()
			}
		}()
	}
	wg.Wait()
	pool.Stop()
	// ровно один submit проходит, задача обработана ровно один раз
	if accepted != 1 || runs != 1 {
		t.Fatalf("accepted=%d runs=%d, want 1/1", accepted, runs)
	}
}

func TestStopFinishesAcceptedTasks(t *testing.T) {
	started := make(chan string, 8)
	gate := make(chan struct{})
	service, pool := newPool(t, 2, 6, usecase.WithAnalyzer(blockingAnalyzer(started, gate)))
	release := gateRelease(t, gate)
	ids := make([]int, 6)
	for i := range ids {
		ids[i] = createTask(t, service, "https://example.com").ID
		if err := pool.Submit(ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	<-started
	stopped := make(chan struct{})
	go func() { pool.Stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("Stop returned while tasks were still running")
	case <-time.After(50 * time.Millisecond):
	}
	release()
	select {
	case <-stopped:
	case <-time.After(timeout):
		t.Fatal("Stop did not return: workers leaked or deadlocked")
	}
	for _, id := range ids {
		if task, _ := service.Get(id); task.Status != domain.Completed {
			t.Fatalf("accepted task %d lost: %s", id, task.Status)
		}
	}
}

func TestSubmitAfterStop(t *testing.T) {
	service, pool := newPool(t, 2, 2)
	task := createTask(t, service, "https://example.com")
	pool.Stop()
	if err := pool.Submit(task.ID); !errors.Is(err, domain.ErrPoolStopped) {
		t.Fatalf("got %v, want ErrPoolStopped", err)
	}
	// отклонённая задача не тронута
	if got, _ := service.Get(task.ID); got.Status != domain.Pending {
		t.Fatalf("got %s, want Pending", got.Status)
	}
}

func TestStopTwice(t *testing.T) {
	_, pool := newPool(t, 2, 2)
	pool.Stop()
	pool.Stop() // не паникует и не зависает
}

func TestWorkersTerminate(t *testing.T) {
	before := runtime.NumGoroutine()
	_, pool := newPool(t, 4, 4)
	pool.Stop()
	deadline := time.Now().Add(timeout)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutines leaked: before=%d after=%d", before, after)
	}
}

func TestSubmitRacesWithStop(t *testing.T) {
	service, pool := newPool(t, 2, 4)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		task := createTask(t, service, "https://example.com")
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pool.Submit(task.ID)
			if err != nil && !errors.Is(err, domain.ErrPoolStopped) && !errors.Is(err, domain.ErrQueueFull) {
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	pool.Stop() // параллельно с Submit: паники send on closed channel быть не должно
	wg.Wait()
	// всё, что приняли, должно быть дообработано
	for id := 1; id <= 50; id++ {
		task, _ := service.Get(id)
		if task.Status == domain.Queued || task.Status == domain.Processing {
			t.Fatalf("task %d stuck in %s", id, task.Status)
		}
	}
}
