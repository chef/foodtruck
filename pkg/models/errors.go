package models

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("Not Found")
var ErrNoTasks = fmt.Errorf("No tasks available: %w", ErrNotFound)
