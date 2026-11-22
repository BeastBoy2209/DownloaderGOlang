package repository_test

import (
	"downloader/internal/domain"
	"downloader/internal/repository"
	"errors"
	"sync"
	"testing"
)

func TestMemoryReturnsCopies(t *testing.T) {
	repo := repository.NewMemory()
	task := repo.Create(domain.Task{Input: "https://example.com", Status: domain.Pending})
	original := task
	task.Input = "changed"
	stored, err := repo.Get(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored != original {
		t.Fatal("changing created task changed storage")
	}
	stored.Result.Host = "changed"
	again, err := repo.Get(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again != original {
		t.Fatal("changing retrieved task changed storage")
	}
}

func TestSaveMissing(t *testing.T) {
	repo := repository.NewMemory()
	if err := repo.Save(domain.Task{ID: 42}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	if _, err := repo.Get(42); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("Save inserted a missing task")
	}
}

func TestUpdateAtomic(t *testing.T) {
	repo := repository.NewMemory()
	task := repo.Create(domain.Task{Input: "https://example.com", Status: domain.Pending})
	// 50 горутин пытаются забрать одну задачу: Enqueue должен пройти ровно у одной
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := repo.Update(task.ID, func(t *domain.Task) error { return t.Enqueue() }); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("got %d winners, want 1", wins)
	}
}

func TestUpdateErrorKeepsTask(t *testing.T) {
	repo := repository.NewMemory()
	task := repo.Create(domain.Task{Input: "https://example.com", Status: domain.Pending})
	boom := errors.New("boom")
	_, err := repo.Update(task.ID, func(t *domain.Task) error {
		t.Status = domain.Completed
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if got, _ := repo.Get(task.ID); got != task {
		t.Fatalf("failed update changed task: %+v", got)
	}
	if _, err := repo.Update(42, func(*domain.Task) error { return nil }); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestConcurrentCreateUniqueIDs(t *testing.T) {
	repo := repository.NewMemory()
	var wg sync.WaitGroup
	ids := make(chan int, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- repo.Create(domain.Task{Status: domain.Pending}).ID
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[int]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
	}
}
