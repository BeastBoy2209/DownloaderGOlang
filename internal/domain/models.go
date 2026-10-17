package domain

import "net/url"

type Status string

const (
	Pending    Status = "Pending"
	Processing Status = "Processing"
	Completed  Status = "Completed"
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

func (t *Task) Process() error {
	if t.Status != Pending {
		return ErrInvalidState
	}
	parsed, err := parseURL(t.Input)
	if err != nil {
		return err
	}
	t.Status = Processing
	t.Result = Result{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   parsed.Path,
		Query:  parsed.RawQuery,
	}
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
