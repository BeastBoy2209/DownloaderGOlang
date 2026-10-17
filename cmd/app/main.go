package main

import (
	"downloader/internal/repository"
	"downloader/internal/usecase"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/app <url> [url...]")
		os.Exit(1)
	}
	service := usecase.NewService(repository.NewMemory())
	if err := run(service, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(service *usecase.Service, inputs []string) error {
	for _, input := range inputs {
		task, err := service.Create(input)
		if err != nil {
			return fmt.Errorf("create %q: %w", input, err)
		}
		id := task.ID
		task, err = service.Process(id)
		if err != nil {
			return fmt.Errorf("process task %d: %w", id, err)
		}
		fmt.Printf("Task %d: %s\nScheme: %s\nHost: %s\nPath: %s\nQuery: %s\n\n",
			task.ID, task.Status, task.Result.Scheme, task.Result.Host,
			task.Result.Path, task.Result.Query)
	}
	return nil
}
