package repository

import (
	"downloader/internal/domain"
	"sync"
)

// Memory - мапа под RWMutex. Мьютекс защищает и мапу, и nextID.
// Задачи хранятся по значению, поэтому наружу уходят копии - чужие поля не трогаем.
type Memory struct {
	mu     sync.RWMutex
	tasks  map[int]domain.Task
	nextID int
}

func NewMemory() *Memory {
	return &Memory{tasks: make(map[int]domain.Task)}
}

func (m *Memory) Create(task domain.Task) domain.Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	task.ID = m.nextID
	m.tasks[task.ID] = task
	return task
}

func (m *Memory) Get(id int) (domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, ok := m.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrNotFound
	}
	return task, nil
}

func (m *Memory) Save(task domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[task.ID]; !ok {
		return domain.ErrNotFound
	}
	m.tasks[task.ID] = task
	return nil
}

// Update - атомарный read-modify-write: fn работает с копией под локом,
// сохраняем только если fn вернула nil. fn должна быть быстрой (никакой тяжёлой работы).
func (m *Memory) Update(id int, fn func(*domain.Task) error) (domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrNotFound
	}
	if err := fn(&task); err != nil {
		return domain.Task{}, err
	}
	m.tasks[id] = task
	return task, nil
}
