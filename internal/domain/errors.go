package domain

import "errors"

var (
	ErrInvalidInput = errors.New("URL must use http or https and have a host")
	ErrNotFound     = errors.New("task not found")
	ErrInvalidState = errors.New("task cannot be processed in its current state")
	ErrQueueFull    = errors.New("task queue is full")
	ErrPoolStopped  = errors.New("worker pool is stopped")
	ErrInvalidPool  = errors.New("pool needs at least 2 workers and queue capacity >= 1")
)
