package repository_test

import (
	"downloader/internal/domain"
	"downloader/internal/repository"
	"errors"
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
