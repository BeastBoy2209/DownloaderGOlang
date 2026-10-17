package domain

import "errors"

var (
	ErrInvalidInput = errors.New("URL must use http or https and have a host")
	ErrNotFound     = errors.New("task not found")
	ErrInvalidState = errors.New("task cannot be processed in its current state")
)
