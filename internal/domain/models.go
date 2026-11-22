package domain

import "net/url"

type Status string

const (
	Pending    Status = "Pending"    // создана, ещё не отправлена в очередь
	Queued     Status = "Queued"     // лежит в очереди, ждёт воркера
	Processing Status = "Processing" // воркер прямо сейчас её обрабатывает
	Completed  Status = "Completed"  // готово, Result заполнен
	Failed     Status = "Failed"     // обработка упала, Result пустой
)

type Result struct {
	Scheme string
	Host   string
	Path   string
	Query  string
}

type Task struct {
	ID     int
	Input  string
	Status Status
	Result Result
}

func (t Task) Validate() error {
	_, err := parseURL(t.Input)
	return err
}

// Analyze - чистая функция разбора URL, без состояния. Её воркеры зовут вне лока.
func Analyze(input string) (Result, error) {
	parsed, err := parseURL(input)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   parsed.Path,
		Query:  parsed.RawQuery,
	}, nil
}

// Process - синхронный путь из CP1: Pending -> Completed в одном вызове.
func (t *Task) Process() error {
	if t.Status != Pending {
		return ErrInvalidState
	}
	result, err := Analyze(t.Input)
	if err != nil {
		return err
	}
	t.Result = result
	t.Status = Completed
	return nil
}

// Enqueue: Pending -> Queued.
func (t *Task) Enqueue() error {
	if t.Status != Pending {
		return ErrInvalidState
	}
	t.Status = Queued
	return nil
}

// Dequeue откатывает Queued -> Pending, если очередь оказалась полной.
func (t *Task) Dequeue() error {
	if t.Status != Queued {
		return ErrInvalidState
	}
	t.Status = Pending
	return nil
}

// Start: Queued -> Processing. Именно здесь воркер "забирает" задачу,
// второй воркер уже получит ErrInvalidState.
func (t *Task) Start() error {
	if t.Status != Queued {
		return ErrInvalidState
	}
	t.Status = Processing
	return nil
}

// Finish: Processing -> Completed или Failed (если processErr != nil).
func (t *Task) Finish(result Result, processErr error) error {
	if t.Status != Processing {
		return ErrInvalidState
	}
	if processErr != nil {
		t.Status = Failed
		return nil
	}
	t.Result = result
	t.Status = Completed
	return nil
}

func parseURL(input string) (*url.URL, error) {
	parsed, err := url.Parse(input)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, ErrInvalidInput
	}
	return parsed, nil
}
