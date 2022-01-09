package errors

import (
	"fmt"
)

type ErrFailedParse struct {
	Candidate string
	PropType string
}

func NewFailedParse(candidate, propType string) *ErrFailedParse {
	return &ErrFailedParse{
		Candidate: candidate,
		PropType: propType,
	}
}

func (e *ErrFailedParse) Error() string {
	return fmt.Sprintf("failed to parse '%s' for type '%s'", e.Candidate, e.PropType)
}


type ErrInvalidPropertyConfig struct {
	Type string
}

func NewInvalidPropertyConfig(propConfigType string) *ErrInvalidPropertyConfig {
	return &ErrInvalidPropertyConfig{Type: propConfigType}
}

func (e *ErrInvalidPropertyConfig) Error() string {
	return fmt.Sprintf("invalid property config type '%v'. did the notion API change?", e.Type)
}
