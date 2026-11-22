package usecase

import "downloader/internal/domain"

type Repository interface {
	Create(domain.Task) domain.Task
	Get(int) (domain.Task, error)
	Save(domain.Task) error
	// Update атомарно меняет задачу под локом репозитория.
	Update(id int, fn func(*domain.Task) error) (domain.Task, error)
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

// Process - синхронная обработка (CP1). Через Update, чтобы не гоняться с воркерами.
func (s *Service) Process(id int) (domain.Task, error) {
	return s.repo.Update(id, func(t *domain.Task) error { return t.Process() })
}
