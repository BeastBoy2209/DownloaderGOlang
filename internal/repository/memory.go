package repository

import "downloader/internal/domain"

type Memory struct {
	tasks  map[int]domain.Task
	nextID int
}

func NewMemory() *Memory {
	return &Memory{tasks: make(map[int]domain.Task)}
}

func (m *Memory) Create(task domain.Task) domain.Task {
	m.nextID++
	task.ID = m.nextID
	m.tasks[task.ID] = task
	return task
}

func (m *Memory) Get(id int) (domain.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrNotFound
	}
	return task, nil
}

func (m *Memory) Save(task domain.Task) error {
	if _, ok := m.tasks[task.ID]; !ok {
		return domain.ErrNotFound
	}
	m.tasks[task.ID] = task
	return nil
}
