package domain

import "errors"

var (
	ErrClient   = errors.New("invalid request")
	ErrBusiness = errors.New("business rule violation")
	ErrServer   = errors.New("internal server error")
)
