package usecase

import (
	"downloader/internal/domain"
	"sync"
)

// Analyzer - то, что реально считает результат. Выносим, чтобы в тестах подменять.
type Analyzer func(input string) (domain.Result, error)

type Option func(*Pool)

func WithAnalyzer(a Analyzer) Option {
	return func(p *Pool) { p.analyze = a }
}

// Pool - фиксированное число воркеров + ограниченная очередь id задач.
//
// Владелец канала queue - Pool: только он его создаёт и закрывает (в Stop).
// mu защищает closed и сам факт send/close: Submit держит RLock на время
// неблокирующей отправки, Stop берёт Lock, так что send в закрытый канал невозможен.
type Pool struct {
	service *Service
	analyze Analyzer
	queue   chan int
	wg      sync.WaitGroup

	mu     sync.RWMutex
	closed bool
}

// NewPool сразу стартует воркеров. Нужно минимум 2 воркера и очередь >= 1.
func NewPool(service *Service, workers, capacity int, opts ...Option) (*Pool, error) {
	if workers < 2 || capacity < 1 {
		return nil, domain.ErrInvalidPool
	}
	p := &Pool{
		service: service,
		analyze: domain.Analyze,
		queue:   make(chan int, capacity),
	}
	for _, opt := range opts {
		opt(p)
	}
	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.worker()
	}
	return p, nil
}

// Submit кладёт задачу в очередь и сразу возвращается (не ждёт обработки).
// Очередь полна -> ErrQueueFull (не блокируем, задача откатывается в Pending).
func (p *Pool) Submit(id int) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return domain.ErrPoolStopped
	}
	// Pending -> Queued атомарно: двойной submit одной задачи получит ErrInvalidState.
	if _, err := p.service.repo.Update(id, func(t *domain.Task) error { return t.Enqueue() }); err != nil {
		return err
	}
	select {
	case p.queue <- id:
		return nil
	default:
		// в очереди места нет - возвращаем задачу как было
		_, _ = p.service.repo.Update(id, func(t *domain.Task) error { return t.Dequeue() })
		return domain.ErrQueueFull
	}
}

// Stop закрывает приём, воркеры доделывают всё, что уже в очереди, и выходят.
// Повторный вызов безопасен (просто снова ждёт wg).
func (p *Pool) Stop() {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.queue)
	}
	p.mu.Unlock()
	p.wg.Wait()
}

// worker живёт, пока канал не закрыт и не опустел (range по закрытому буферу дочитывает остаток).
func (p *Pool) worker() {
	defer p.wg.Done()
	for id := range p.queue {
		p.run(id)
	}
}

func (p *Pool) run(id int) {
	repo := p.service.repo
	// "захват" задачи: Queued -> Processing под локом
	task, err := repo.Update(id, func(t *domain.Task) error { return t.Start() })
	if err != nil {
		return
	}
	// тяжёлая часть - вне лока, иначе воркеры встанут в очередь друг за другом
	result, procErr := p.analyze(task.Input)
	_, _ = repo.Update(id, func(t *domain.Task) error { return t.Finish(result, procErr) })
}
