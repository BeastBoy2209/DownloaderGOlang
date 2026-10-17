package usecase_test

import (
	"downloader/internal/domain"
	"downloader/internal/repository"
	"downloader/internal/usecase"
	"errors"
	"testing"
)

func newService() *usecase.Service {
	return usecase.NewService(repository.NewMemory())
}

func createTask(t *testing.T, service *usecase.Service, input string) domain.Task {
	t.Helper()
	task, err := service.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestCreate(t *testing.T) {
	service := newService()
	task := createTask(t, service, "https://example.com/docs")
	if task.ID <= 0 || task.Input != "https://example.com/docs" || task.Status != domain.Pending || task.Result != (domain.Result{}) {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestCreateInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"relative", "/docs"},
		{"unsupported scheme", "ftp://example.com/file"},
		{"missing host", "https:///docs"},
		{"missing hostname", "https://:8080/docs"},
		{"invalid escape", "https://example.com/%zz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newService()
			if _, err := service.Create(tt.input); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("got %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	service := newService()
	created := createTask(t, service, "https://example.com")
	got, err := service.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Fatalf("got %+v, want %+v", got, created)
	}
}

func TestGetMissing(t *testing.T) {
	if _, err := newService().Get(42); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  domain.Result
	}{
		{"full URL", "https://example.com:8080/docs/intro?q=go&lang=en#top",
			domain.Result{Scheme: "https", Host: "example.com:8080", Path: "/docs/intro", Query: "q=go&lang=en"}},
		{"host only", "http://example.org",
			domain.Result{Scheme: "http", Host: "example.org"}},
		{"escaped path", "https://example.com/hello%20world",
			domain.Result{Scheme: "https", Host: "example.com", Path: "/hello world"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newService()
			created := createTask(t, service, tt.input)
			got, err := service.Process(created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Result != tt.want {
				t.Fatalf("got result %+v, want %+v", got.Result, tt.want)
			}
			if got.Status != domain.Completed {
				t.Fatalf("got status %s, want Completed", got.Status)
			}
			stored, err := service.Get(created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored != got {
				t.Fatalf("stored %+v, want %+v", stored, got)
			}
		})
	}
}

func TestProcessMissing(t *testing.T) {
	if _, err := newService().Process(42); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestProcessCompleted(t *testing.T) {
	service := newService()
	created := createTask(t, service, "https://example.com")
	completed, err := service.Process(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Process(created.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("got %v, want ErrInvalidState", err)
	}
	stored, err := service.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored != completed {
		t.Fatalf("repeated processing changed task: %+v", stored)
	}
}

func TestIndependentTasks(t *testing.T) {
	service := newService()
	first := createTask(t, service, "https://first.example/a")
	second := createTask(t, service, "http://second.example/b")
	if first.ID == second.ID {
		t.Fatal("task IDs must be unique")
	}
	first, err := service.Process(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := service.Get(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pending != second {
		t.Fatalf("processing first task changed second: %+v", pending)
	}
	second, err = service.Process(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored != first || first.Result.Host != "first.example" || second.Result.Host != "second.example" {
		t.Fatalf("tasks are not independent: %+v, %+v", stored, second)
	}
}
