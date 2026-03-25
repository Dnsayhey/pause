package reminder

import "errors"

var (
	ErrAlreadyExists = errors.New("reminder already exists")
	ErrNotFound      = errors.New("reminder not found")
)
