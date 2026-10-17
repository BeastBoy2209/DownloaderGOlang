package domain_test

import (
	"downloader/internal/domain"
	"errors"
	"testing"
)

func TestProcessInvalidState(t *testing.T) {
	for _, status := range []domain.Status{domain.Processing, domain.Completed, "", "Unknown"} {
		t.Run(string(status), func(t *testing.T) {
			task := domain.Task{ID: 1, Input: "https://example.com", Status: status}
			before := task
			if err := task.Process(); !errors.Is(err, domain.ErrInvalidState) {
				t.Fatalf("got %v, want ErrInvalidState", err)
			}
			if task != before {
				t.Fatal("invalid transition changed task")
			}
		})
	}
}

func TestProcessInvalidInput(t *testing.T) {
	task := domain.Task{ID: 1, Status: domain.Pending}
	before := task
	if err := task.Process(); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("got %v, want ErrInvalidInput", err)
	}
	if task != before {
		t.Fatal("invalid input changed task")
	}
}
