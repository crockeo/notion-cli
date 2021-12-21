package errors

import (
	"errors"
)

var (
	ErrFailedDateParse       = errors.New("failed to parse date")
	ErrFailedParse           = errors.New("failed to parse")
	ErrInvalidPropertyConfig = errors.New("invalid property config type. did the notion API change?")
)
