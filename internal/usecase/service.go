package usecase

import "downloader/internal/domain"

type Repository interface {
	Create(domain.Task) domain.Task
	Get(int) (domain.Task, error)
	Save(domain.Task) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(input string) (domain.Task, error) {
	task := domain.Task{Input: input, Status: domain.Pending}
	if err := task.Validate(); err != nil {
		return domain.Task{}, err
	}
	return s.repo.Create(task), nil
}

func (s *Service) Get(id int) (domain.Task, error) {
	return s.repo.Get(id)
}

func (s *Service) Process(id int) (domain.Task, error) {
	task, err := s.repo.Get(id)
	if err != nil {
		return domain.Task{}, err
	}
	if err := task.Process(); err != nil {
		return domain.Task{}, err
	}
	if err := s.repo.Save(task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}
