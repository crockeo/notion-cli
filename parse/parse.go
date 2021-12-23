package parse

import (
	"time"

	"github.com/jomei/notionapi"
	"github.com/olebedev/when"

	"github.com/crockeo/notion-capture/errors"
)

func Property(propName string, propConfig notionapi.PropertyConfig, propValue string) (notionapi.Property, error) {
	var property notionapi.Property
	var err error
	if selectPropConfig, ok := propConfig.(*notionapi.SelectPropertyConfig); ok {
		property, err = ParseSelect(propValue, selectPropConfig.Select.Options)
	} else if _, ok := propConfig.(*notionapi.DatePropertyConfig); ok {
		now := time.Now()
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		property, err = ParseDate(propValue, now)
	} else {
		err = errors.ErrInvalidPropertyConfig
	}
	return property, err
}

// TODO: migrate all of the property parsing to here,
// instead of in prompt.go
// so it can be use programmatically
func ParseDate(candidate string, now time.Time) (*DateProperty, error) {
	if candidate == "" {
		return nil, nil
	}

	result, err := when.EN.Parse(candidate, now)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.ErrFailedParse
	}
	date := result.Time.Round(0)

	return &DateProperty{
		Date: DateObject{
			Start: (*TimelessDate)(&date),
		},
	}, nil
}

func ParseSelect(candidate string, options []notionapi.Option) (*notionapi.SelectProperty, error) {
	// FIXME: can a select have multiple options with the same name?
	// if so, fix this to be more robust
	for _, option := range options {
		if candidate == option.Name {
			return &notionapi.SelectProperty{
				Select: option,
			}, nil
		}
	}

	return nil, errors.ErrFailedParse
}

// this cursed block here replicates the API of notion
// while allowing us to serialize datetimes without the time part
// so that we can schedule tasks without assigning times of days
type DateProperty struct {
	ID   notionapi.ObjectID     `json:"id,omitempty"`
	Type notionapi.PropertyType `json:"type,omitempty"`
	Date DateObject             `json:"date"`
}

func (dp *DateProperty) GetType() notionapi.PropertyType {
	return dp.Type
}

type DateObject struct {
	Start *TimelessDate `json:"start"`
	End   *TimelessDate `json:"end"`
}

type TimelessDate time.Time

func (td *TimelessDate) MarshalJSON() ([]byte, error) {
	var format string
	date := (*time.Time)(td)
	if date.Hour() != 0 || date.Minute() != 0 || date.Second() != 0 || date.Nanosecond() != 0 {
		format = time.RFC3339
	} else {
		format = "2006-01-02"
	}
	result := date.Format(format)
	return []byte("\"" + result + "\""), nil
}
