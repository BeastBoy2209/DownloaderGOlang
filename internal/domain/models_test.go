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

func TestLifecycleTransitions(t *testing.T) {
	task := domain.Task{Input: "https://example.com/a?b=c", Status: domain.Pending}
	if err := task.Start(); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("Start from Pending: %v", err)
	}
	if err := task.Enqueue(); err != nil || task.Status != domain.Queued {
		t.Fatalf("Enqueue: %v %s", err, task.Status)
	}
	if err := task.Enqueue(); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("double Enqueue: %v", err)
	}
	if err := task.Start(); err != nil || task.Status != domain.Processing {
		t.Fatalf("Start: %v %s", err, task.Status)
	}
	res, _ := domain.Analyze(task.Input)
	if err := task.Finish(res, nil); err != nil || task.Status != domain.Completed || task.Result.Host != "example.com" {
		t.Fatalf("Finish: %v %+v", err, task)
	}
	if err := task.Finish(res, nil); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("Finish from Completed: %v", err)
	}
}

func TestFinishFailed(t *testing.T) {
	task := domain.Task{Status: domain.Processing}
	if err := task.Finish(domain.Result{Host: "x"}, domain.ErrInvalidInput); err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.Failed || task.Result != (domain.Result{}) {
		t.Fatalf("got %+v", task)
	}
}

func TestDequeue(t *testing.T) {
	task := domain.Task{Status: domain.Queued}
	if err := task.Dequeue(); err != nil || task.Status != domain.Pending {
		t.Fatalf("got %v %s", err, task.Status)
	}
	if err := task.Dequeue(); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("got %v", err)
	}
}
