package parse

import (
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jomei/notionapi"
	"github.com/nyaruka/phonenumbers"
	"github.com/olebedev/when"

	"github.com/crockeo/notion-cli/errors"
)

func Property(propName string, propConfig notionapi.PropertyConfig, propValue string) (notionapi.Property, error) {
	var property notionapi.Property
	var err error

	switch propConfig := propConfig.(type) {
	case *notionapi.RichTextPropertyConfig:
		property, err = ParseRichText(propValue)
	case *notionapi.NumberPropertyConfig:
		property, err = ParseNumber(propValue)
	case *notionapi.SelectPropertyConfig:
		property, err = ParseSelect(propValue, propConfig.Select.Options)
	case *notionapi.MultiSelectPropertyConfig:
		property, err = ParseMultiSelect(propValue, propConfig.MultiSelect.Options)
	case *notionapi.DatePropertyConfig:
		now := time.Now()
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		property, err = ParseDate(propValue, now)
	case *notionapi.CheckboxPropertyConfig:
		property, err = ParseCheckbox(propValue)
	case *notionapi.URLPropertyConfig:
		property, err = ParseURL(propValue)
	case *notionapi.EmailPropertyConfig:
		property, err = ParseEmail(propValue)
	case *notionapi.PhoneNumberPropertyConfig:
		property, err = ParsePhoneNumber(propValue)
	default:
		err = errors.ErrInvalidPropertyConfig
	}
	return property, err
}

func ParseRichText(candidate string) (*notionapi.RichTextProperty, error) {
	// TODO: do something more advanced here,
	// e.g. use some sort of markup language
	// to parse this and render it as RichText
	return &notionapi.RichTextProperty{
		RichText: []notionapi.RichText{
			{Text: notionapi.Text{Content: candidate}},
		},
	}, nil
}

func ParseNumber(candidate string) (*notionapi.NumberProperty, error) {
	number, err := strconv.ParseFloat(candidate, 64)
	if err != nil {
		return nil, err
	}
	return &notionapi.NumberProperty{
		Number: number,
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

func ParseMultiSelect(candidate string, options []notionapi.Option) (*notionapi.MultiSelectProperty, error) {
	return nil, errors.ErrFailedParse
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

func ParseCheckbox(candidate string) (*notionapi.CheckboxProperty, error) {
	candidate = strings.ToLower(candidate)
	positiveOptions := []string{
		"true",
		"y",
		"yes",
	}
	negativeOptions := []string{
		"false",
		"n",
		"no",
	}

	for _, positiveOption := range positiveOptions {
		if candidate == positiveOption {
			return &notionapi.CheckboxProperty{
				Checkbox: true,
			}, nil
		}
	}

	for _, negativeOption := range negativeOptions {
		if candidate == negativeOption {
			return &notionapi.CheckboxProperty{
				Checkbox: false,
			}, nil
		}
	}

	return nil, errors.ErrFailedParse
}

func ParseURL(candidate string) (*notionapi.URLProperty, error) {
	_, err := url.Parse(candidate)
	if err != nil {
		return nil, err
	}
	return &notionapi.URLProperty{
		URL: candidate,
	}, err
}

func ParseEmail(candidate string) (*notionapi.EmailProperty, error) {
	address, err := mail.ParseAddress(candidate)
	if err != nil {
		return nil, err
	}
	return &notionapi.EmailProperty{
		Email: address.Address,
	}, nil
}

func ParsePhoneNumber(candidate string) (*notionapi.PhoneNumberProperty, error) {
	// notion wants a string which contains the phone number
	// so despite the fact that we parse a structured phone number
	// we actually don't care about it :)
	_, err := phonenumbers.Parse(candidate, "US")
	if err != nil {
		return nil, err
	}
	return &notionapi.PhoneNumberProperty{
		PhoneNumber: candidate,
	}, nil
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
