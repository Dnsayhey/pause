package reminder

import "errors"

var (
	ErrAlreadyExists = errors.New("reminder already exists")
	ErrNotFound      = errors.New("reminder not found")
	ErrNameRequired  = errors.New("reminder name is required")
	ErrNameTrimmed   = errors.New("reminder name cannot have leading or trailing spaces")
	ErrIntervalRange = errors.New("reminder intervalSec must be > 0")
	ErrBreakRange    = errors.New("reminder breakSec must be > 0")
	ErrTypeRequired  = errors.New("reminder reminderType is required")
	ErrTypeInvalid   = errors.New("reminder reminderType must be rest or notify")
	ErrIDRequired    = errors.New("reminder id is required")
)
