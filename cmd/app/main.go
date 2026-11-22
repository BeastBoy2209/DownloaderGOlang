package main

import (
	"downloader/internal/repository"
	"downloader/internal/usecase"
	"fmt"
	"os"
)

const workers = 3

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
	// очередь с запасом под все аргументы, чтобы CLI не упирался в ErrQueueFull
	pool, err := usecase.NewPool(service, workers, len(inputs))
	if err != nil {
		return err
	}
	// Stop дожидается воркеров, так что после него все задачи финальные
	defer pool.Stop()

	ids := make([]int, 0, len(inputs))
	for _, input := range inputs {
		task, err := service.Create(input)
		if err != nil {
			return fmt.Errorf("create %q: %w", input, err)
		}
		if err := pool.Submit(task.ID); err != nil {
			return fmt.Errorf("submit task %d: %w", task.ID, err)
		}
		ids = append(ids, task.ID)
	}
	pool.Stop()

	for _, id := range ids {
		task, err := service.Get(id)
		if err != nil {
			return fmt.Errorf("get task %d: %w", id, err)
		}
		fmt.Printf("Task %d: %s\nScheme: %s\nHost: %s\nPath: %s\nQuery: %s\n\n",
			task.ID, task.Status, task.Result.Scheme, task.Result.Host,
			task.Result.Path, task.Result.Query)
	}
	return nil
}
