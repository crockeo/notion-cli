package errors

import (
	"errors"
	"fmt"
)

var (
	ErrFailedParse = errors.New("failed to parse")
)

type ErrInvalidPropertyConfig struct {
	Type string
}

func NewInvalidPropertyConfig(propConfigType string) *ErrInvalidPropertyConfig {
	return &ErrInvalidPropertyConfig{Type: propConfigType}
}

func (e *ErrInvalidPropertyConfig) Error() string {
	return fmt.Sprintf("invalid property config type '%v'. did the notion API change?", e.Type)
}
