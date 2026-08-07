package domain

import "errors"

var (
	ErrClient   = errors.New("client error")
	ErrBusiness = errors.New("business error")
	ErrServer   = errors.New("server error")
)
